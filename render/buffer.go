package render

// Buffer 是一个宽 w、高 h 的单元格网格，坐标 (0,0) 在左上角。
type Buffer struct {
	w, h  int
	cells []Cell
}

// NewBuffer 创建一个全空白缓冲。
func NewBuffer(w, h int) *Buffer {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	b := &Buffer{w: w, h: h, cells: make([]Cell, w*h)}
	for i := range b.cells {
		b.cells[i] = BlankCell()
	}
	return b
}

// Size 返回缓冲宽高。
func (b *Buffer) Size() (w, h int) { return b.w, b.h }

func (b *Buffer) inBounds(x, y int) bool { return x >= 0 && x < b.w && y >= 0 && y < b.h }

// Get 返回 (x,y) 处的单元格，越界返回空白格。
func (b *Buffer) Get(x, y int) Cell {
	if !b.inBounds(x, y) {
		return BlankCell()
	}
	return b.cells[y*b.w+x]
}

func (b *Buffer) setRaw(x, y int, c Cell) {
	if b.inBounds(x, y) {
		b.cells[y*b.w+x] = c
	}
}

// clearWideAt 清理 (x,y) 处单元格可能关联的宽字符另一半：
// 若 (x,y) 是宽字符的延续格，置空其左侧主格；若是宽字符主格，置空其右侧延续格。
func (b *Buffer) clearWideAt(x, y int) {
	c := b.Get(x, y)
	if c.Width == 0 && x > 0 {
		b.setRaw(x-1, y, BlankCell())
	}
	if c.Width == 2 {
		b.setRaw(x+1, y, BlankCell())
	}
}

// Set 在 (x,y) 放置一个字素簇，自动按宽度占用后续列并清理被覆盖的宽字符。
// 宽字符在右边界放不下时忽略本次写入。
func (b *Buffer) Set(x, y int, cluster string, st Style) {
	if !b.inBounds(x, y) {
		return
	}
	w := GraphemeWidth(cluster)
	if w == 2 && x+1 >= b.w {
		return
	}
	// 清理写入范围两端可能存在的宽字符残留
	b.clearWideAt(x, y)
	if w == 2 {
		b.clearWideAt(x+1, y)
	}
	b.setRaw(x, y, Cell{Cluster: cluster, Style: st, Width: w})
	if w == 2 {
		b.setRaw(x+1, y, Cell{Style: st, Width: 0})
	}
}

// SetString 从 (x,y) 起横向写入字符串，超出右边界截断。
func (b *Buffer) SetString(x, y int, s string, st Style) {
	if y < 0 || y >= b.h {
		return
	}
	for _, g := range Graphemes(s) {
		if x >= b.w {
			break
		}
		w := GraphemeWidth(g)
		if w == 2 && x+1 >= b.w {
			break // 宽字符放不下，截断
		}
		b.Set(x, y, g, st)
		x += w
	}
}

// Fill 用样式填充矩形区域（内容置空），自动裁剪到缓冲内。
func (b *Buffer) Fill(x, y, w, h int, st Style) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if b.inBounds(xx, yy) {
				b.Set(xx, yy, " ", st)
			}
		}
	}
}

// Clear 重置为全空白。
func (b *Buffer) Clear() {
	for i := range b.cells {
		b.cells[i] = BlankCell()
	}
}

// Clone 返回缓冲的深拷贝。
func (b *Buffer) Clone() *Buffer {
	nb := &Buffer{w: b.w, h: b.h, cells: make([]Cell, len(b.cells))}
	copy(nb.cells, b.cells)
	return nb
}

// Diff 返回把 prev 变成 b 所需修改的单元格坐标列表（按行优先顺序）。
// prev 为 nil 或尺寸不同视为全量差异。
func (b *Buffer) Diff(prev *Buffer) []Pos {
	var out []Pos
	if prev == nil || prev.w != b.w || prev.h != b.h {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				out = append(out, Pos{x, y})
			}
		}
		return out
	}
	for y := 0; y < b.h; y++ {
		base := y * b.w
		for x := 0; x < b.w; x++ {
			if !b.cells[base+x].Equal(prev.cells[base+x]) {
				out = append(out, Pos{x, y})
			}
		}
	}
	return out
}

// Pos 是一个单元格坐标。
type Pos struct{ X, Y int }
