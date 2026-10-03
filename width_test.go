package terminalrender

import "testing"

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		{'A', 1},
		{'中', 2},
		{'é', 1},
		{0x0301, 0},  // combining acute
		{0x200D, 0},  // zero-width joiner
		{0xFE0F, 0},  // variation selector
		{0x1F469, 2}, // woman emoji
	}
	for _, tt := range tests {
		if got := RuneWidth(tt.r); got != tt.want {
			t.Errorf("RuneWidth(%U) = %d, want %d", tt.r, got, tt.want)
		}
	}
}

func TestSegmentText(t *testing.T) {
	clusters := segmentText("á好👩‍💻a‍b")
	want := []graphemeCluster{
		{text: "á", width: 1},
		{text: "好", width: 2},
		{text: "👩‍💻", width: 2},
		{text: "a‍", width: 1},
		{text: "b", width: 1},
	}
	if len(clusters) != len(want) {
		t.Fatalf("got %d clusters, want %d: %#v", len(clusters), len(want), clusters)
	}
	for i := range want {
		if clusters[i].text != want[i].text || clusters[i].width != want[i].width {
			t.Errorf("cluster %d = %#v, want %#v", i, clusters[i], want[i])
		}
	}
}
