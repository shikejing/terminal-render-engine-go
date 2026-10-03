package terminalrender

import (
	"fmt"
	"strconv"
	"strings"
)

// ColorKind identifies how a terminal color is represented.
type ColorKind uint8

const (
	ColorDefault ColorKind = iota
	ColorANSI
	ColorPalette
	ColorRGB
)

// Color is a terminal color. The zero value is the terminal default color.
type Color struct {
	kind  ColorKind
	value uint32
}

func ANSI(index uint8) Color    { return Color{kind: ColorANSI, value: uint32(index)} }
func Palette(index uint8) Color { return Color{kind: ColorPalette, value: uint32(index)} }
func RGB(r, g, b uint8) Color {
	return Color{kind: ColorRGB, value: uint32(r)<<16 | uint32(g)<<8 | uint32(b)}
}

func (c Color) Kind() ColorKind { return c.kind }
func (c Color) IsZero() bool    { return c.kind == ColorDefault }

func (c Color) rgb() (uint8, uint8, uint8) {
	r := uint8(c.value >> 16)
	g := uint8(c.value >> 8)
	b := uint8(c.value)
	return r, g, b
}

var ansiRGB = [16][3]uint8{
	{0, 0, 0},
	{128, 0, 0},
	{0, 128, 0},
	{128, 128, 0},
	{0, 0, 128},
	{128, 0, 128},
	{0, 128, 128},
	{192, 192, 192},
	{128, 128, 128},
	{255, 0, 0},
	{0, 255, 0},
	{255, 255, 0},
	{0, 0, 255},
	{255, 0, 255},
	{0, 255, 255},
	{255, 255, 255},
}

var cubeValues = [6]uint8{0, 95, 135, 175, 215, 255}

func paletteRGB(index uint8) (uint8, uint8, uint8) {
	if index < 16 {
		c := ansiRGB[index]
		return c[0], c[1], c[2]
	}
	if index >= 232 {
		v := uint8(8 + (int(index)-232)*10)
		return v, v, v
	}
	n := int(index) - 16
	r := cubeValues[n/36]
	g := cubeValues[(n/6)%6]
	b := cubeValues[n%6]
	return r, g, b
}

func colorDistance(r1, g1, b1, r2, g2, b2 int) int {
	dr := int(r1) - int(r2)
	dg := int(g1) - int(g2)
	db := int(b1) - int(b2)
	return dr*dr + dg*dg + db*db
}

func nearestANSI(r, g, b uint8) uint8 {
	best := uint8(0)
	bestDistance := -1
	for i, c := range ansiRGB {
		d := colorDistance(int(r), int(g), int(b), int(c[0]), int(c[1]), int(c[2]))
		if bestDistance < 0 || d < bestDistance {
			bestDistance = d
			best = uint8(i)
		}
	}
	return best
}

func nearestPalette(r, g, b uint8) uint8 {
	best := uint8(16)
	bestDistance := -1
	for i := 16; i < 256; i++ {
		pr, pg, pb := paletteRGB(uint8(i))
		d := colorDistance(int(r), int(g), int(b), int(pr), int(pg), int(pb))
		if bestDistance < 0 || d < bestDistance {
			bestDistance = d
			best = uint8(i)
		}
	}
	return best
}

func ansiSGR(index uint8, background bool) string {
	base := 30
	if index >= 8 {
		base = 90
		index -= 8
	}
	if background {
		base += 10
	}
	return strconv.Itoa(base + int(index))
}

func (c Color) sgr(background bool, caps Capabilities) string {
	if c.kind == ColorDefault || caps.Colors == ColorModeNone {
		return ""
	}
	base := 38
	if background {
		base = 48
	}
	switch c.kind {
	case ColorANSI:
		return ansiSGR(uint8(c.value), background)
	case ColorPalette:
		idx := uint8(c.value)
		if caps.Colors == ColorMode16 {
			r, g, b := paletteRGB(idx)
			return ansiSGR(nearestANSI(r, g, b), background)
		}
		return fmt.Sprintf("%d;5;%d", base, idx)
	case ColorRGB:
		r, g, b := c.rgb()
		switch caps.Colors {
		case ColorModeTrueColor:
			return fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b)
		case ColorMode256:
			return fmt.Sprintf("%d;5;%d", base, nearestPalette(r, g, b))
		case ColorMode16:
			return ansiSGR(nearestANSI(r, g, b), background)
		}
	}
	return ""
}

// SGR returns the full SGR escape sequence for a style, or a reset sequence
// when no attributes or colors are available.
func (s Style) SGR(caps Capabilities) string {
	var b strings.Builder
	b.WriteString("\x1b[0")
	if caps.Attributes {
		if s.Bold {
			b.WriteString(";1")
		}
		if s.Dim {
			b.WriteString(";2")
		}
		if s.Italic {
			b.WriteString(";3")
		}
		if s.Underline {
			b.WriteString(";4")
		}
		if s.Blink {
			b.WriteString(";5")
		}
		if s.Reverse {
			b.WriteString(";7")
		}
		if s.Hidden {
			b.WriteString(";8")
		}
		if s.Strikethrough {
			b.WriteString(";9")
		}
	}
	if fg := s.FG.sgr(false, caps); fg != "" {
		b.WriteByte(';')
		b.WriteString(fg)
	}
	if bg := s.BG.sgr(true, caps); bg != "" {
		b.WriteByte(';')
		b.WriteString(bg)
	}
	b.WriteByte('m')
	return b.String()
}
