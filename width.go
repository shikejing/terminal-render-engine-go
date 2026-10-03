package terminalrender

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var wideRanges = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x1100, Hi: 0x115F, Stride: 1}, // Hangul Jamo
		{Lo: 0x231A, Hi: 0x231B, Stride: 1},
		{Lo: 0x2329, Hi: 0x232A, Stride: 1},
		{Lo: 0x23E9, Hi: 0x23EC, Stride: 1},
		{Lo: 0x23F0, Hi: 0x23F0, Stride: 1},
		{Lo: 0x23F3, Hi: 0x23F3, Stride: 1},
		{Lo: 0x25FD, Hi: 0x25FE, Stride: 1},
		{Lo: 0x2614, Hi: 0x2615, Stride: 1},
		{Lo: 0x2648, Hi: 0x2653, Stride: 1},
		{Lo: 0x267F, Hi: 0x267F, Stride: 1},
		{Lo: 0x2693, Hi: 0x2693, Stride: 1},
		{Lo: 0x26A1, Hi: 0x26A1, Stride: 1},
		{Lo: 0x26AA, Hi: 0x26AB, Stride: 1},
		{Lo: 0x26BD, Hi: 0x26BE, Stride: 1},
		{Lo: 0x26C4, Hi: 0x26C5, Stride: 1},
		{Lo: 0x26CE, Hi: 0x26CE, Stride: 1},
		{Lo: 0x26D4, Hi: 0x26D4, Stride: 1},
		{Lo: 0x26EA, Hi: 0x26EA, Stride: 1},
		{Lo: 0x26F2, Hi: 0x26F3, Stride: 1},
		{Lo: 0x26F5, Hi: 0x26F5, Stride: 1},
		{Lo: 0x26FA, Hi: 0x26FA, Stride: 1},
		{Lo: 0x26FD, Hi: 0x26FD, Stride: 1},
		{Lo: 0x2702, Hi: 0x2702, Stride: 1},
		{Lo: 0x2705, Hi: 0x2705, Stride: 1},
		{Lo: 0x2708, Hi: 0x270D, Stride: 1},
		{Lo: 0x270F, Hi: 0x270F, Stride: 1},
		{Lo: 0x2712, Hi: 0x2712, Stride: 1},
		{Lo: 0x2714, Hi: 0x2714, Stride: 1},
		{Lo: 0x2716, Hi: 0x2716, Stride: 1},
		{Lo: 0x271D, Hi: 0x271D, Stride: 1},
		{Lo: 0x2721, Hi: 0x2721, Stride: 1},
		{Lo: 0x2728, Hi: 0x2728, Stride: 1},
		{Lo: 0x2733, Hi: 0x2734, Stride: 1},
		{Lo: 0x2744, Hi: 0x2744, Stride: 1},
		{Lo: 0x2747, Hi: 0x2747, Stride: 1},
		{Lo: 0x274C, Hi: 0x274C, Stride: 1},
		{Lo: 0x274E, Hi: 0x274E, Stride: 1},
		{Lo: 0x2753, Hi: 0x2755, Stride: 1},
		{Lo: 0x2757, Hi: 0x2757, Stride: 1},
		{Lo: 0x2763, Hi: 0x2764, Stride: 1},
		{Lo: 0x2795, Hi: 0x2797, Stride: 1},
		{Lo: 0x27A1, Hi: 0x27A1, Stride: 1},
		{Lo: 0x27B0, Hi: 0x27B0, Stride: 1},
		{Lo: 0x27BF, Hi: 0x27BF, Stride: 1},
		{Lo: 0x2934, Hi: 0x2935, Stride: 1},
		{Lo: 0x2B05, Hi: 0x2B07, Stride: 1},
		{Lo: 0x2B1B, Hi: 0x2B1C, Stride: 1},
		{Lo: 0x2B50, Hi: 0x2B50, Stride: 1},
		{Lo: 0x2B55, Hi: 0x2B55, Stride: 1},
		{Lo: 0x3000, Hi: 0x303E, Stride: 1},
		{Lo: 0x3041, Hi: 0x3096, Stride: 1},
		{Lo: 0x3099, Hi: 0x30FF, Stride: 1},
		{Lo: 0x3105, Hi: 0x312F, Stride: 1},
		{Lo: 0x3131, Hi: 0x318F, Stride: 1},
		{Lo: 0x3190, Hi: 0x319F, Stride: 1},
		{Lo: 0x31C0, Hi: 0x31EF, Stride: 1},
		{Lo: 0x31F0, Hi: 0x31FF, Stride: 1},
		{Lo: 0x3200, Hi: 0x32FF, Stride: 1},
		{Lo: 0x3300, Hi: 0x33FF, Stride: 1},
		{Lo: 0x3400, Hi: 0x4DBF, Stride: 1},
		{Lo: 0x4DC0, Hi: 0x4DFF, Stride: 1},
		{Lo: 0x4E00, Hi: 0x9FFF, Stride: 1},
		{Lo: 0xA000, Hi: 0xA4CF, Stride: 1},
		{Lo: 0xA960, Hi: 0xA97F, Stride: 1},
		{Lo: 0xAC00, Hi: 0xD7A3, Stride: 1},
		{Lo: 0xF900, Hi: 0xFAFF, Stride: 1},
		{Lo: 0xFE10, Hi: 0xFE19, Stride: 1},
		{Lo: 0xFE30, Hi: 0xFE4F, Stride: 1},
		{Lo: 0xFE50, Hi: 0xFE6F, Stride: 1},
		{Lo: 0xFF00, Hi: 0xFF60, Stride: 1},
		{Lo: 0xFFE0, Hi: 0xFFE6, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x1F000, Hi: 0x1F0FF, Stride: 1},
		{Lo: 0x1F100, Hi: 0x1F1FF, Stride: 1},
		{Lo: 0x1F200, Hi: 0x1F2FF, Stride: 1},
		{Lo: 0x1F300, Hi: 0x1F5FF, Stride: 1},
		{Lo: 0x1F600, Hi: 0x1F64F, Stride: 1},
		{Lo: 0x1F680, Hi: 0x1F6FF, Stride: 1},
		{Lo: 0x1F700, Hi: 0x1F77F, Stride: 1},
		{Lo: 0x1F780, Hi: 0x1F7FF, Stride: 1},
		{Lo: 0x1F800, Hi: 0x1F8FF, Stride: 1},
		{Lo: 0x1F900, Hi: 0x1F9FF, Stride: 1},
		{Lo: 0x1FA00, Hi: 0x1FA6F, Stride: 1},
		{Lo: 0x1FA70, Hi: 0x1FAFF, Stride: 1},
		{Lo: 0x20000, Hi: 0x2FFFD, Stride: 1},
		{Lo: 0x30000, Hi: 0x3FFFD, Stride: 1},
	},
}

var zeroRanges = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x061C, Hi: 0x061C, Stride: 1},
		{Lo: 0x17B4, Hi: 0x17B5, Stride: 1},
		{Lo: 0x180B, Hi: 0x180D, Stride: 1},
		{Lo: 0x180E, Hi: 0x180E, Stride: 1},
		{Lo: 0x200B, Hi: 0x200F, Stride: 1},
		{Lo: 0x202A, Hi: 0x202E, Stride: 1},
		{Lo: 0x2060, Hi: 0x2064, Stride: 1},
		{Lo: 0x2066, Hi: 0x206F, Stride: 1},
		{Lo: 0xFE00, Hi: 0xFE0F, Stride: 1},
		{Lo: 0xFEFF, Hi: 0xFEFF, Stride: 1},
		{Lo: 0xFFF9, Hi: 0xFFFB, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x1BCA0, Hi: 0x1BCA3, Stride: 1},
		{Lo: 0x1D173, Hi: 0x1D17A, Stride: 1},
		{Lo: 0xE0001, Hi: 0xE0001, Stride: 1},
		{Lo: 0xE0020, Hi: 0xE007F, Stride: 1},
		{Lo: 0xE0100, Hi: 0xE01EF, Stride: 1},
	},
}

// RuneWidth reports the terminal column width of a single rune.
func RuneWidth(r rune) int {
	if r == utf8.RuneError {
		return 1
	}
	if unicode.IsControl(r) {
		return 0
	}
	if unicode.In(r, unicode.Mn, unicode.Me, zeroRanges) {
		return 0
	}
	if unicode.In(r, wideRanges) {
		return 2
	}
	return 1
}

type graphemeCluster struct {
	text  string
	width int
}

func isVariationSelector(r rune) bool {
	return r >= 0xFE00 && r <= 0xFE0F || r >= 0xE0100 && r <= 0xE01EF
}

func isEmojiModifier(r rune) bool {
	return r >= 0x1F3FB && r <= 0x1F3FF
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func isEmojiBase(r rune) bool {
	return r >= 0x1F000 && r <= 0x1FAFF || r >= 0x2600 && r <= 0x27FF
}

// segmentText splits text into terminal-displayable grapheme clusters.
// Zero-width clusters are retained so callers can attach them to an existing cell.
func segmentText(text string) []graphemeCluster {
	clusters := make([]graphemeCluster, 0, len(text))
	var current strings.Builder
	width := 0
	afterZWJ := false
	emoji := false
	riCount := 0

	flush := func() {
		if current.Len() > 0 {
			clusters = append(clusters, graphemeCluster{text: current.String(), width: width})
			current.Reset()
			width = 0
			afterZWJ = false
			emoji = false
			riCount = 0
		}
	}

	for _, r := range text {
		if unicode.IsControl(r) && r != 0x200D {
			continue
		}
		w := RuneWidth(r)
		if current.Len() == 0 {
			current.WriteRune(r)
			width = w
			afterZWJ = r == 0x200D
			emoji = isEmojiBase(r)
			if isRegionalIndicator(r) {
				riCount = 1
			}
			continue
		}

		switch {
		case w == 0:
			current.WriteRune(r)
			afterZWJ = r == 0x200D
		case afterZWJ:
			if isEmojiBase(r) {
				current.WriteRune(r)
				afterZWJ = false
				emoji = true
			} else {
				flush()
				current.WriteRune(r)
				width = w
				afterZWJ = false
				emoji = isEmojiBase(r)
				if isRegionalIndicator(r) {
					riCount = 1
				}
			}
		case isRegionalIndicator(r) && riCount == 1:
			current.WriteRune(r)
			riCount = 2
		case isEmojiModifier(r) && emoji:
			current.WriteRune(r)
		case isVariationSelector(r):
			current.WriteRune(r)
		default:
			flush()
			current.WriteRune(r)
			width = w
			afterZWJ = false
			emoji = isEmojiBase(r)
			if isRegionalIndicator(r) {
				riCount = 1
			}
		}
	}
	flush()
	return clusters
}
