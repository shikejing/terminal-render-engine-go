package terminalrender

// Cell is one terminal cell. Content is a display-safe grapheme cluster.
type Cell struct {
	Content string
	Style   Style
	Width   int
}

// Buffer is a fixed-size character grid.
type Buffer struct {
	width  int
	height int
	cells  []Cell
}

func blankCell() Cell {
	return Cell{Content: " ", Width: 1}
}

func NewBuffer(width, height int) *Buffer {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	cells := make([]Cell, width*height)
	for i := range cells {
		cells[i] = blankCell()
	}
	return &Buffer{width: width, height: height, cells: cells}
}

func (b *Buffer) Size() (int, int) { return b.width, b.height }

func (b *Buffer) Clear() {
	if b == nil {
		return
	}
	for i := range b.cells {
		b.cells[i] = blankCell()
	}
}

// Set writes text at (x, y). Text is split into grapheme clusters; wide clusters
// occupy two columns and combining/zero-width clusters attach to the current cell.
func (b *Buffer) Set(x, y int, text string, style Style) {
	if b == nil || y < 0 || y >= b.height || b.width == 0 {
		return
	}
	for _, cluster := range segmentText(text) {
		if cluster.width == 0 {
			if x >= 0 && x < b.width {
				idx := y*b.width + x
				if b.cells[idx].Width > 0 {
					b.cells[idx].Content += cluster.text
				}
			}
			continue
		}
		if x < 0 || x+cluster.width > b.width {
			x += cluster.width
			continue
		}
		idx := y*b.width + x
		b.cells[idx] = Cell{Content: cluster.text, Style: style, Width: cluster.width}
		x += cluster.width
	}
}

// Fill fills a rectangular area with spaces and the given style.
func (b *Buffer) Fill(x, y, width, height int, style Style) {
	if b == nil {
		return
	}
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			if col >= 0 && col < b.width && row >= 0 && row < b.height {
				b.cells[row*b.width+col] = Cell{Content: " ", Style: style, Width: 1}
			}
		}
	}
}

func (b *Buffer) CellAt(x, y int) Cell {
	if b == nil || x < 0 || y < 0 || x >= b.width || y >= b.height {
		return blankCell()
	}
	return b.cells[y*b.width+x]
}

func (b *Buffer) clone() *Buffer {
	if b == nil {
		return NewBuffer(0, 0)
	}
	cells := make([]Cell, len(b.cells))
	copy(cells, b.cells)
	return &Buffer{width: b.width, height: b.height, cells: cells}
}
