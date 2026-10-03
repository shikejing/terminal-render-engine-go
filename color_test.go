package terminalrender

import "testing"

func TestStyleSGRDegradation(t *testing.T) {
	style := Style{FG: RGB(255, 0, 0), Bold: true}
	tests := []struct {
		name string
		caps Capabilities
		want string
	}{
		{"truecolor", Capabilities{Colors: ColorModeTrueColor, Attributes: true}, "\x1b[0;1;38;2;255;0;0m"},
		{"256", Capabilities{Colors: ColorMode256, Attributes: true}, "\x1b[0;1;38;5;196m"},
		{"16", Capabilities{Colors: ColorMode16, Attributes: true}, "\x1b[0;1;91m"},
		{"none", Capabilities{Colors: ColorModeNone, Attributes: true}, "\x1b[0;1m"},
		{"dumb", Capabilities{Colors: ColorModeNone, Attributes: false}, "\x1b[0m"},
	}
	for _, tt := range tests {
		if got := style.SGR(tt.caps); got != tt.want {
			t.Errorf("%s: SGR = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectColorMode(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	if got := Detect(nil, ColorModeAuto); got.Colors != ColorMode256 {
		t.Fatalf("256color TERM = %v", got.Colors)
	}
	t.Setenv("COLORTERM", "truecolor")
	if got := Detect(nil, ColorModeAuto); got.Colors != ColorModeTrueColor {
		t.Fatalf("truecolor COLORTERM = %v", got.Colors)
	}
	t.Setenv("NO_COLOR", "1")
	if got := Detect(nil, ColorModeAuto); got.Colors != ColorModeNone {
		t.Fatalf("NO_COLOR = %v", got.Colors)
	}
}
