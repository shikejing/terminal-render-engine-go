package render

// RuneWidth 返回单个 rune 在终端中的显示列宽：
// 0 表示零宽（组合字符、控制字符等），2 表示东亚宽字符，其余为 1。
func RuneWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		// 控制字符：不输出，按零宽处理
		return 0
	}
	if isZeroWidth(r) {
		return 0
	}
	if isWide(r) {
		return 2
	}
	return 1
}

// StringWidth 返回字符串按终端列宽计算的总宽度。
func StringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += RuneWidth(r)
	}
	return w
}

// Graphemes 把字符串切分为字素簇（基础字符 + 后续零宽组合字符）。
// 控制字符被丢弃。
func Graphemes(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range s {
		w := RuneWidth(r)
		if w == 0 {
			if r >= 32 || r == 0 {
				// 零宽组合字符附加到当前簇
				cur = append(cur, r)
			}
			// 控制字符直接丢弃
			continue
		}
		flush()
		cur = append(cur, r)
	}
	flush()
	return out
}

// GraphemeWidth 返回一个字素簇的显示宽度（取簇内最大宽度）。
func GraphemeWidth(g string) int {
	w := 0
	for _, r := range g {
		if rw := RuneWidth(r); rw > w {
			w = rw
		}
	}
	if w == 0 {
		w = 1 // 纯组合字符簇保底占 1 列
	}
	return w
}

// isZeroWidth 判断组合字符与其他零宽字符。
func isZeroWidth(r rune) bool {
	switch {
	case r >= 0x0300 && r <= 0x036F, // Combining Diacritical Marks
		r >= 0x0483 && r <= 0x0489,
		r >= 0x0591 && r <= 0x05BD,
		r >= 0x0610 && r <= 0x061A,
		r >= 0x064B && r <= 0x065F,
		r >= 0x0670 && r <= 0x0670,
		r >= 0x06D6 && r <= 0x06DC,
		r >= 0x0E31 && r <= 0x0E3A,
		r >= 0x0E47 && r <= 0x0E4E,
		r >= 0x1AB0 && r <= 0x1AFF,
		r >= 0x1DC0 && r <= 0x1DFF,
		r >= 0x20D0 && r <= 0x20FF,                // Combining Diacritical Marks for Symbols
		r >= 0xFE20 && r <= 0xFE2F,                // Combining Half Marks
		r >= 0xFE00 && r <= 0xFE0F,                // Variation Selectors
		r == 0x200B || r == 0x200C || r == 0x200D, // ZWSP/ZWNJ/ZWJ
		r == 0xFEFF:                               // BOM / ZWNBSP
		return true
	}
	return false
}

// isWide 判断东亚宽字符（W/F 类，基于 Unicode East Asian Width）。
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E,   // CJK Radicals .. CJK Symbols
		r >= 0x3041 && r <= 0x33FF,   // Hiragana .. CJK Compat
		r >= 0x3400 && r <= 0x4DBF,   // CJK Ext A
		r >= 0x4E00 && r <= 0x9FFF,   // CJK Unified
		r >= 0xA000 && r <= 0xA4CF,   // Yi
		r >= 0xAC00 && r <= 0xD7A3,   // Hangul Syllables
		r >= 0xF900 && r <= 0xFAFF,   // CJK Compat Ideographs
		r >= 0xFE30 && r <= 0xFE4F,   // CJK Compat Forms
		r >= 0xFF00 && r <= 0xFF60,   // Fullwidth Forms
		r >= 0xFFE0 && r <= 0xFFE6,   // Fullwidth signs
		r >= 0x1F300 && r <= 0x1F64F, // Emoji
		r >= 0x20000 && r <= 0x3FFFD: // CJK Ext B+
		return true
	}
	return false
}
