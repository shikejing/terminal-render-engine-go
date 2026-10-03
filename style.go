package terminalrender

// Style describes the visual appearance of one terminal cell.
type Style struct {
	FG            Color
	BG            Color
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Hidden        bool
	Strikethrough bool
}

func (s Style) Equal(other Style) bool { return s == other }
