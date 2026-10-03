package terminalrender

import (
	"bytes"
	"fmt"
	"io"
)

// Renderer writes cell buffers to a terminal with local updates.
type Renderer struct {
	out  io.Writer
	caps Capabilities
}

func NewRenderer(out io.Writer, caps Capabilities) *Renderer {
	if out == nil {
		out = io.Discard
	}
	return &Renderer{out: out, caps: caps}
}

// Render writes only cells that differ from old. A size mismatch or force flag
// triggers a full redraw. The complete frame is assembled before writing so a
// slow writer cannot interleave or truncate a control sequence.
func (r *Renderer) Render(old, next *Buffer, force bool) error {
	if r == nil || next == nil {
		return nil
	}
	if old == nil || old.width != next.width || old.height != next.height {
		force = true
	}

	var frame bytes.Buffer
	if force {
		frame.WriteString("\x1b[H\x1b[2J")
	}

	var current Style
	styleValid := false
	for y := 0; y < next.height; y++ {
		for x := 0; x < next.width; {
			idx := y*next.width + x
			cell := next.cells[idx]
			if force {
				if isDefaultCell(cell) {
					x++
					continue
				}
			} else if cell.equal(old.cells[idx]) {
				x++
				continue
			}

			fmt.Fprintf(&frame, "\x1b[%d;%dH", y+1, x+1)
			for x < next.width {
				cell = next.cells[idx]
				if !force && cell.equal(old.cells[idx]) {
					break
				}
				if !styleValid || !current.Equal(cell.Style) {
					if r.caps.Colors != ColorModeNone || r.caps.Attributes {
						frame.WriteString(cell.Style.SGR(r.caps))
					}
					current = cell.Style
					styleValid = true
				}
				frame.WriteString(cell.Content)
				x += cell.Width
				if cell.Width <= 0 {
					x++
				}
				idx = y*next.width + x
			}
		}
	}
	frame.WriteString("\x1b[0m")
	return writeAll(r.out, frame.Bytes())
}

func (r *Renderer) Enter(altScreen, hideCursor bool) error {
	if r == nil {
		return nil
	}
	var frame bytes.Buffer
	if altScreen {
		frame.WriteString("\x1b[?1049h")
	}
	if hideCursor {
		frame.WriteString("\x1b[?25l")
	}
	return writeAll(r.out, frame.Bytes())
}

func (r *Renderer) Leave(altScreen, showCursor bool) error {
	if r == nil {
		return nil
	}
	var frame bytes.Buffer
	frame.WriteString("\x1b[0m")
	if showCursor {
		frame.WriteString("\x1b[?25h")
	}
	if altScreen {
		frame.WriteString("\x1b[?1049l")
	}
	return writeAll(r.out, frame.Bytes())
}

func isDefaultCell(cell Cell) bool {
	return cell.Content == " " && cell.Style == (Style{})
}

func (c Cell) equal(other Cell) bool {
	return c.Content == other.Content && c.Style == other.Style && c.Width == other.Width
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
