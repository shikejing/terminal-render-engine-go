package terminalrender

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeTerminal struct {
	mu          sync.Mutex
	buf         bytes.Buffer
	frames      [][]byte
	width       int
	height      int
	interactive bool
	slow        time.Duration
}

func (f *fakeTerminal) Write(p []byte) (int, error) {
	if f.slow > 0 {
		time.Sleep(f.slow)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	frame := append([]byte(nil), p...)
	f.frames = append(f.frames, frame)
	f.buf.Write(p)
	return len(p), nil
}

func (f *fakeTerminal) Size() (int, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.width, f.height, nil
}

func (f *fakeTerminal) IsTerminal() bool { return f.interactive }

func (f *fakeTerminal) setSize(width, height int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.width, f.height = width, height
}

func (f *fakeTerminal) snapshot() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.String()
}

func (f *fakeTerminal) frameCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.frames)
}

func (f *fakeTerminal) lastFrame() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.frames) == 0 {
		return nil
	}
	return append([]byte(nil), f.frames[len(f.frames)-1]...)
}

func waitFor(t *testing.T, term *fakeTerminal, substr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(term.snapshot(), substr) {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in %q", substr, term.snapshot())
}

func waitForFrameCount(t *testing.T, term *fakeTerminal, count int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if term.frameCount() >= count {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d frames, got %d", count, term.frameCount())
}

type engineState struct {
	Left  string
	Right string
}

func newTestEngine(term *fakeTerminal, view ViewFunc[engineState]) *Engine[engineState] {
	return New(view,
		WithTerminal(term),
		WithFrameInterval(2*time.Millisecond),
		WithColorMode(ColorModeNone),
		WithAltScreen(false),
		WithHideCursor(false),
		WithRawMode(false),
	)
}

func TestEngineCoalescesAndDiffs(t *testing.T) {
	term := &fakeTerminal{width: 10, height: 5}
	e := newTestEngine(term, func(state engineState, b *Buffer) {
		b.Set(0, 0, state.Left, Style{})
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = e.Run(ctx) }()

	e.Submit(engineState{Left: "A0"})
	waitFor(t, term, "A0")
	frames := term.frameCount()

	e.Submit(engineState{Left: "B0"})
	waitForFrameCount(t, term, frames+1)
	last := string(term.lastFrame())
	if !strings.Contains(last, "B") {
		t.Fatalf("latest state missing from frame: %q", last)
	}
	if strings.Contains(last, "A") {
		t.Fatalf("superseded state was rewritten: %q", last)
	}
}

func TestEngineConcurrentSubmissionsUseConsistentSnapshot(t *testing.T) {
	term := &fakeTerminal{width: 10, height: 5}
	var renderedMu sync.Mutex
	var rendered engineState
	view := func(state engineState, b *Buffer) {
		renderedMu.Lock()
		rendered = state
		renderedMu.Unlock()
		b.Set(0, 0, state.Left, Style{})
		b.Set(1, 0, state.Right, Style{})
	}
	e := newTestEngine(term, view)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = e.Run(ctx) }()

	const count = 26
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			letter := string(rune('A' + i))
			e.Submit(engineState{Left: letter, Right: letter})
		}(i)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	renderedMu.Lock()
	got := rendered
	renderedMu.Unlock()
	if got.Left != got.Right || len(got.Left) != 1 || got.Left[0] < 'A' || got.Left[0] > 'Z' {
		t.Fatalf("rendered inconsistent snapshot: %+v", got)
	}
	last := string(term.lastFrame())
	if !strings.Contains(last, got.Left) || !strings.Contains(last, got.Right) {
		t.Fatalf("frame does not contain complete snapshot %+v: %q", got, last)
	}
}

func TestEngineResizeForcesFullRedraw(t *testing.T) {
	term := &fakeTerminal{width: 10, height: 5}
	e := newTestEngine(term, func(state engineState, b *Buffer) {
		b.Set(0, 0, state.Left, Style{})
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = e.Run(ctx) }()

	e.Submit(engineState{Left: "A0"})
	waitFor(t, term, "A0")
	frames := term.frameCount()

	term.setSize(12, 6)
	e.NotifyResize()
	waitForFrameCount(t, term, frames+1)
	last := string(term.lastFrame())
	if !strings.HasPrefix(last, "\x1b[H\x1b[2J") {
		t.Fatalf("resize did not force clear: %q", last)
	}
	if !strings.Contains(last, "A0") {
		t.Fatalf("resize redraw lost state: %q", last)
	}
}

func TestEngineRestoresInteractiveTerminal(t *testing.T) {
	term := &fakeTerminal{width: 10, height: 5, interactive: true}
	e := New(func(state engineState, b *Buffer) {},
		WithTerminal(term),
		WithFrameInterval(2*time.Millisecond),
		WithColorMode(ColorModeNone),
		WithAltScreen(true),
		WithHideCursor(true),
		WithRawMode(false),
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = e.Run(ctx)
		close(done)
	}()

	waitFor(t, term, "\x1b[?1049h")
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("engine did not stop")
	}
	out := term.snapshot()
	if !strings.Contains(out, "\x1b[?25h") || !strings.Contains(out, "\x1b[?1049l") {
		t.Fatalf("terminal was not restored: %q", out)
	}
}

func TestEngineSubmitDoesNotBlockOnSlowOutput(t *testing.T) {
	term := &fakeTerminal{width: 10, height: 5, slow: 20 * time.Millisecond}
	e := newTestEngine(term, func(state engineState, b *Buffer) {
		b.Set(0, 0, state.Left, Style{})
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = e.Run(ctx)
		close(done)
	}()

	start := time.Now()
	for i := 0; i < 100; i++ {
		e.Submit(engineState{Left: fmt.Sprintf("L%d", i)})
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("submissions blocked for %s", elapsed)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("engine did not stop")
	}
}
