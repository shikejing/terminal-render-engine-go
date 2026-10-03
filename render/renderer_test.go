package render

import (
	"bytes"
	"strings"
	"testing"
)

// writeRecorder 记录每次 Write 调用，用于验证一帧一次写入（无半个序列）。
type writeRecorder struct {
	writes [][]byte
}

func (w *writeRecorder) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	w.writes = append(w.writes, cp)
	return len(p), nil
}

func (w *writeRecorder) all() string {
	var b bytes.Buffer
	for _, p := range w.writes {
		b.Write(p)
	}
	return b.String()
}

func testCaps() Caps { return Caps{Color: ColorTrue, AltScreen: true, SyncUpdate: true} }

func TestRendererEnterExit(t *testing.T) {
	rec := &writeRecorder{}
	r := NewRenderer(rec, testCaps(), 10, 5)
	if err := r.Enter(); err != nil {
		t.Fatal(err)
	}
	out := rec.all()
	if !strings.Contains(out, "\x1b[?1049h") || !strings.Contains(out, "\x1b[?25l") {
		t.Errorf("enter output missing alt-screen/hide-cursor: %q", out)
	}
	if err := r.Exit(); err != nil {
		t.Fatal(err)
	}
	out = rec.all()
	if !strings.Contains(out, "\x1b[?1049l") || !strings.Contains(out, "\x1b[?25h") || !strings.Contains(out, "\x1b[0m") {
		t.Errorf("exit output missing restore sequences: %q", out)
	}
	// Exit 可重复调用且幂等
	n := len(rec.writes)
	_ = r.Exit()
	if len(rec.writes) != n {
		t.Errorf("second Exit wrote more output")
	}
}

func TestRendererDiffOnlyWritesChanged(t *testing.T) {
	rec := &writeRecorder{}
	r := NewRenderer(rec, testCaps(), 10, 3)
	buf1 := NewBuffer(10, 3)
	buf1.SetString(0, 0, "hello", Style{})
	if err := r.Render(buf1); err != nil {
		t.Fatal(err)
	}
	// 第二帧只改一个单元格
	rec.writes = nil
	buf2 := buf1.Clone()
	buf2.Set(0, 0, "H", Style{})
	if err := r.Render(buf2); err != nil {
		t.Fatal(err)
	}
	out := rec.all()
	if strings.Contains(out, "ello") {
		t.Errorf("unchanged cells were rewritten: %q", out)
	}
	if !strings.Contains(out, "\x1b[1;1H") || !strings.Contains(out, "H") {
		t.Errorf("changed cell not written: %q", out)
	}
	// 第三帧无变化：不应有任何单元格输出
	rec.writes = nil
	if err := r.Render(buf2.Clone()); err != nil {
		t.Fatal(err)
	}
	if out := rec.all(); strings.Contains(out, "H") {
		t.Errorf("unchanged frame produced cell output: %q", out)
	}
}

func TestRendererFrameIsSingleWrite(t *testing.T) {
	rec := &writeRecorder{}
	r := NewRenderer(rec, testCaps(), 10, 3)
	buf := NewBuffer(10, 3)
	buf.SetString(0, 0, "ab", Style{})
	buf.SetString(0, 1, "cd", Style{})
	if err := r.Render(buf); err != nil {
		t.Fatal(err)
	}
	// Enter 一次 + Render 一次，Render 必须恰好是一次 Write
	if len(rec.writes) != 2 {
		t.Fatalf("writes = %d, want 2 (enter + single frame)", len(rec.writes))
	}
	frame := string(rec.writes[1])
	if !strings.HasPrefix(frame, "\x1b[?2026h") || !strings.HasSuffix(frame, "\x1b[?2026l") {
		t.Errorf("frame not wrapped in sync update: %q", frame)
	}
}

func TestRendererResizeFullRedraw(t *testing.T) {
	rec := &writeRecorder{}
	r := NewRenderer(rec, testCaps(), 10, 3)
	buf := NewBuffer(10, 3)
	buf.SetString(0, 0, "abc", Style{})
	_ = r.Render(buf)

	r.Resize(20, 4)
	rec.writes = nil
	buf2 := NewBuffer(20, 4)
	buf2.SetString(0, 0, "abc", Style{})
	_ = r.Render(buf2)
	out := rec.all()
	// 尺寸变化后需清屏全量重绘，避免残影
	if !strings.Contains(out, "\x1b[2J") {
		t.Errorf("resize did not trigger full clear: %q", out)
	}
	if !strings.Contains(out, "abc") {
		t.Errorf("resize frame missing content: %q", out)
	}
}

func TestRendererClipsToTerminalSize(t *testing.T) {
	rec := &writeRecorder{}
	r := NewRenderer(rec, testCaps(), 4, 2)
	buf := NewBuffer(10, 5) // 缓冲大于终端
	buf.SetString(0, 4, "overflow", Style{})
	if err := r.Render(buf); err != nil {
		t.Fatal(err)
	}
	out := rec.all()
	if strings.Contains(out, "overflow") {
		t.Errorf("out-of-bounds cells were written: %q", out)
	}
}

func TestRendererColorDowngrade(t *testing.T) {
	trueRed := Color{Mode: ColorTrue, Value: 0xFF0000}
	st := Style{Fg: trueRed}

	// 真彩色终端：输出 38;2 序列
	rec := &writeRecorder{}
	r := NewRenderer(rec, Caps{Color: ColorTrue}, 5, 1)
	buf := NewBuffer(5, 1)
	buf.Set(0, 0, "x", st)
	_ = r.Render(buf)
	if !strings.Contains(rec.all(), "\x1b[38;2;255;0;0m") {
		t.Errorf("truecolor output missing: %q", rec.all())
	}

	// 256 色终端：降级为 38;5
	rec = &writeRecorder{}
	r = NewRenderer(rec, Caps{Color: Color256}, 5, 1)
	_ = r.Render(buf)
	out := rec.all()
	if !strings.Contains(out, "\x1b[38;5;") || strings.Contains(out, "38;2;") {
		t.Errorf("256-color downgrade wrong: %q", out)
	}

	// 16 色终端：降级为基本色，不得出现 38;5 或 38;2
	rec = &writeRecorder{}
	r = NewRenderer(rec, Caps{Color: Color16}, 5, 1)
	_ = r.Render(buf)
	out = rec.all()
	if strings.Contains(out, "38;5;") || strings.Contains(out, "38;2;") {
		t.Errorf("16-color terminal got unsupported sequence: %q", out)
	}
	if !strings.Contains(out, "\x1b[31m") && !strings.Contains(out, "\x1b[91m") {
		t.Errorf("16-color fallback missing: %q", out)
	}

	// 无颜色终端：不得输出任何颜色序列
	rec = &writeRecorder{}
	r = NewRenderer(rec, Caps{Color: ColorNone}, 5, 1)
	_ = r.Render(buf)
	out = rec.all()
	if strings.Contains(out, "38;") || strings.Contains(out, "[31m") || strings.Contains(out, "[91m") {
		t.Errorf("no-color terminal got color sequence: %q", out)
	}
}

func TestDetectCaps(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	if c := DetectCaps(env(map[string]string{"TERM": "xterm-256color", "COLORTERM": "truecolor"})); c.Color != ColorTrue {
		t.Errorf("truecolor detect = %v", c.Color)
	}
	if c := DetectCaps(env(map[string]string{"TERM": "xterm-256color"})); c.Color != Color256 {
		t.Errorf("256 detect = %v", c.Color)
	}
	if c := DetectCaps(env(map[string]string{"TERM": "linux"})); c.Color != Color16 {
		t.Errorf("16 detect = %v", c.Color)
	}
	if c := DetectCaps(env(map[string]string{"TERM": "dumb"})); c.Color != ColorNone || c.AltScreen {
		t.Errorf("dumb detect = %+v", c)
	}
	if c := DetectCaps(env(map[string]string{"TERM": "xterm", "NO_COLOR": "1"})); c.Color != ColorNone {
		t.Errorf("NO_COLOR detect = %v", c.Color)
	}
}
