package render

import "testing"

func TestRuneWidth(t *testing.T) {
	cases := []struct {
		r rune
		w int
	}{
		{'a', 1},
		{' ', 1},
		{'中', 2},
		{'文', 2},
		{'あ', 2},
		{'한', 2},
		{'，', 2},     // 全角标点
		{'Ａ', 2},     // 全角字母
		{'́', 0},     // 组合重音符 U+0301
		{'‍', 0},     // ZWJ
		{'\n', 0},    // 控制字符
		{0x1F600, 2}, // Emoji
	}
	for _, c := range cases {
		if got := RuneWidth(c.r); got != c.w {
			t.Errorf("RuneWidth(%q) = %d, want %d", c.r, got, c.w)
		}
	}
}

func TestStringWidth(t *testing.T) {
	if got := StringWidth("hello"); got != 5 {
		t.Errorf("StringWidth(hello) = %d, want 5", got)
	}
	if got := StringWidth("中文"); got != 4 {
		t.Errorf("StringWidth(中文) = %d, want 4", got)
	}
	// 中英文混排
	if got := StringWidth("a中b"); got != 4 {
		t.Errorf("StringWidth(a中b) = %d, want 4", got)
	}
	// 组合字符不占宽
	if got := StringWidth("é"); got != 1 {
		t.Errorf("StringWidth(e+combining) = %d, want 1", got)
	}
}

func TestGraphemes(t *testing.T) {
	gs := Graphemes("ab́中")
	if len(gs) != 3 {
		t.Fatalf("Graphemes len = %d, want 3: %q", len(gs), gs)
	}
	if gs[1] != "b́" {
		t.Errorf("gs[1] = %q, want combining cluster", gs[1])
	}
	if gs[2] != "中" {
		t.Errorf("gs[2] = %q, want 中", gs[2])
	}
	// 控制字符被丢弃
	gs = Graphemes("a\nb")
	if len(gs) != 2 {
		t.Fatalf("Graphemes with control char len = %d, want 2", len(gs))
	}
}
