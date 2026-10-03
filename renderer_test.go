package terminalrender

import (
	"bytes"
	"strings"
	"testing"
)

func TestRendererLocalDiff(t *testing.T) {
	old := NewBuffer(4, 1)
	old.Set(0, 0, "A", Style{})
	next := old.clone()
	next.Set(0, 0, "B", Style{})
	next.Set(1, 0, "C", Style{})

	var out bytes.Buffer
	r := NewRenderer(&out, Capabilities{Colors: ColorModeNone, Attributes: false})
	if err := r.Render(old, next, false); err != nil {
		t.Fatal(err)
	}
	frame := out.String()
	if !strings.Contains(frame, "\x1b[1;1HBC") {
		t.Fatalf("missing contiguous changed cells: %q", frame)
	}
	if strings.Contains(frame, "A") {
		t.Fatalf("unchanged old content was rewritten: %q", frame)
	}
	if !strings.HasSuffix(frame, "\x1b[0m") {
		t.Fatalf("frame does not end with reset: %q", frame)
	}
}

func TestRendererNoChangeWritesNothing(t *testing.T) {
	b := NewBuffer(2, 1)
	b.Set(0, 0, "A", Style{})
	var out bytes.Buffer
	r := NewRenderer(&out, Capabilities{Colors: ColorModeNone, Attributes: false})
	if err := r.Render(b, b.clone(), false); err != nil {
		t.Fatal(err)
	}
	if out.String() != "\x1b[0m" {
		t.Fatalf("unexpected output for unchanged frame: %q", out.String())
	}
}

func TestRendererForceFullRedraw(t *testing.T) {
	old := NewBuffer(2, 1)
	next := NewBuffer(2, 1)
	next.Set(0, 0, "好", Style{})
	var out bytes.Buffer
	r := NewRenderer(&out, Capabilities{Colors: ColorModeNone, Attributes: false})
	if err := r.Render(old, next, true); err != nil {
		t.Fatal(err)
	}
	frame := out.String()
	if !strings.HasPrefix(frame, "\x1b[H\x1b[2J") {
		t.Fatalf("force redraw does not clear: %q", frame)
	}
	if !strings.Contains(frame, "好") {
		t.Fatalf("force redraw missing wide content: %q", frame)
	}
}

func TestRendererWideCharacterPositioning(t *testing.T) {
	old := NewBuffer(4, 1)
	next := NewBuffer(4, 1)
	next.Set(0, 0, "好", Style{})
	var out bytes.Buffer
	r := NewRenderer(&out, Capabilities{Colors: ColorModeNone, Attributes: false})
	if err := r.Render(old, next, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[1;1H好") {
		t.Fatalf("wide char was not positioned correctly: %q", out.String())
	}
}
