package render

// ColorMode 表示终端支持的颜色能力等级。
type ColorMode int

const (
	ColorNone ColorMode = iota // 无颜色能力
	Color16                    // 16 色（ANSI 标准/高亮）
	Color256                   // 256 色
	ColorTrue                  // 真彩色（24bit）
)

// Color 表示一个颜色，Mode 决定 Value 的解释方式。
// Color16: Value 为 0-15；Color256: Value 为 0-255；ColorTrue: Value 为 0xRRGGBB。
type Color struct {
	Mode  ColorMode
	Value uint32
}

// Style 描述单元格的显示样式。
type Style struct {
	Fg, Bg    Color
	Bold      bool
	Underline bool
	Reverse   bool
}

// Equal 比较两个样式是否完全一致。
func (s Style) Equal(o Style) bool {
	return s.Fg == o.Fg && s.Bg == o.Bg &&
		s.Bold == o.Bold && s.Underline == o.Underline && s.Reverse == o.Reverse
}

// Cell 是字符网格中的一个单元格。
// Cluster 为该单元格显示的字素簇（可能含组合字符），
// Width 为占用的终端列数（1 或 2）；宽字符的第二个单元格 Width 为 0，
// 表示它是前一个宽字符的延续（continuation），不独立输出。
type Cell struct {
	Cluster string
	Style   Style
	Width   int
}

// Equal 比较两个单元格内容是否一致。
func (c Cell) Equal(o Cell) bool {
	return c.Cluster == o.Cluster && c.Width == o.Width && c.Style.Equal(o.Style)
}

// BlankCell 返回一个空白单元格。
func BlankCell() Cell { return Cell{Cluster: " ", Width: 1} }
