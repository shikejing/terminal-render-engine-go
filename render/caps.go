package render

import "strings"

// Caps 描述终端能力。
type Caps struct {
	Color      ColorMode // 颜色能力
	AltScreen  bool      // 是否支持备用屏幕
	SyncUpdate bool      // 是否支持同步更新（BSU/ESU）防闪烁
}

// DetectCaps 根据环境变量（TERM、COLORTERM 等）探测终端能力。
// env 为取值函数（如 os.Getenv），便于测试注入。
func DetectCaps(env func(string) string) Caps {
	caps := Caps{Color: Color16, AltScreen: true}
	term := strings.ToLower(env("TERM"))
	colorTerm := strings.ToLower(env("COLORTERM"))

	if term == "" || term == "dumb" {
		return Caps{Color: ColorNone, AltScreen: false}
	}
	if strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") ||
		strings.Contains(term, "alacritty") || strings.Contains(term, "foot") {
		caps.SyncUpdate = true
	}

	switch {
	case colorTerm == "truecolor" || colorTerm == "24bit":
		caps.Color = ColorTrue
	case strings.Contains(term, "256color"):
		caps.Color = Color256
	case strings.Contains(term, "16color") || strings.HasPrefix(term, "xterm") ||
		strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") ||
		strings.HasPrefix(term, "linux") || strings.HasPrefix(term, "vt"):
		caps.Color = Color16
	default:
		caps.Color = Color16
	}
	if env("NO_COLOR") != "" {
		caps.Color = ColorNone
	}
	return caps
}

// downgrade 把颜色降级到 mode 能力内可表达的颜色。
func downgrade(c Color, mode ColorMode) Color {
	if c.Mode <= mode {
		return c
	}
	switch mode {
	case ColorNone:
		return Color{Mode: ColorNone}
	case Color16:
		if c.Mode == Color256 {
			return Color{Mode: Color16, Value: uint32(x256to16(int(c.Value)))}
		}
		// truecolor -> 16
		return Color{Mode: Color16, Value: uint32(rgbTo16(int(c.Value>>16&0xFF), int(c.Value>>8&0xFF), int(c.Value&0xFF)))}
	case Color256:
		// truecolor -> 256
		return Color{Mode: Color256, Value: uint32(rgbTo256(int(c.Value>>16&0xFF), int(c.Value>>8&0xFF), int(c.Value&0xFF)))}
	}
	return c
}

// rgbTo256 把 24bit RGB 映射到 xterm 256 色索引。
func rgbTo256(r, g, b int) int {
	// 灰度判断
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return 232 + (r-8)/10
	}
	to6 := func(v int) int {
		if v < 48 {
			return 0
		}
		if v < 115 {
			return 1
		}
		return (v - 35) / 40
	}
	return 16 + 36*to6(r) + 6*to6(g) + to6(b)
}

// x256to16 把 256 色索引近似映射到 16 色。
func x256to16(i int) int {
	if i < 16 {
		return i
	}
	if i >= 232 { // 灰度阶梯
		if i < 244 {
			return 8 // 亮黑（灰）
		}
		return 7 // 白
	}
	i -= 16
	r, g, b := i/36, (i/6)%6, i%6
	toBit := func(v int) int {
		if v >= 3 {
			return 1
		}
		return 0
	}
	bright := 0
	if r+g+b >= 9 {
		bright = 8
	}
	return bright | toBit(r)<<0 | toBit(g)<<1 | toBit(b)<<2
}

// rgbTo16 把 24bit RGB 映射到最近的 16 色索引。
func rgbTo16(r, g, b int) int {
	bright := 0
	if r+g+b >= 384 {
		bright = 8
	}
	bit := func(v int) int {
		if v >= 128 {
			return 1
		}
		return 0
	}
	return bright | bit(r)<<0 | bit(g)<<1 | bit(b)<<2
}
