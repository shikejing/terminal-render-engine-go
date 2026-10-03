package terminalrender

import "testing"

func TestBufferSetAndWideCharacters(t *testing.T) {
	b := NewBuffer(6, 2)
	style := Style{FG: ANSI(1)}
	b.Set(0, 0, "á好", style)

	if got := b.CellAt(0, 0); got.Content != "á" || got.Width != 1 {
		t.Fatalf("cell 0,0 = %#v", got)
	}
	if got := b.CellAt(1, 0); got.Content != "好" || got.Width != 2 {
		t.Fatalf("cell 1,0 = %#v", got)
	}
	if got := b.CellAt(3, 0).Content; got != " " {
		t.Fatalf("cell 3,0 = %q, want blank", got)
	}
}

func TestBufferZeroWidthAttachesToCell(t *testing.T) {
	b := NewBuffer(2, 1)
	b.Set(0, 0, "e", Style{})
	b.Set(0, 0, "́", Style{})
	if got := b.CellAt(0, 0).Content; got != "é" {
		t.Fatalf("content = %q, want e\\u0301", got)
	}
}

func TestBufferClipsOutOfBounds(t *testing.T) {
	b := NewBuffer(2, 1)
	b.Set(1, 0, "好", Style{}) // starts on last column, does not fit
	if got := b.CellAt(1, 0).Content; got != " " {
		t.Fatalf("out-of-range wide char wrote %q", got)
	}
	b.Set(0, 0, "ok", Style{})
	if got0, got1 := b.CellAt(0, 0).Content, b.CellAt(1, 0).Content; got0 != "o" || got1 != "k" {
		t.Fatalf("bounds text = %q,%q", got0, got1)
	}
}

func TestBufferFillAndClear(t *testing.T) {
	b := NewBuffer(2, 2)
	style := Style{BG: ANSI(4)}
	b.Fill(0, 0, 2, 2, style)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if cell := b.CellAt(x, y); cell.Content != " " || cell.Style != style {
				t.Fatalf("cell %d,%d = %#v", x, y, cell)
			}
		}
	}
	b.Clear()
	if cell := b.CellAt(0, 0); cell != blankCell() {
		t.Fatalf("cleared cell = %#v", cell)
	}
}
