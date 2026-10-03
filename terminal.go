package terminalrender

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// Terminal abstracts the output device and its size.
type Terminal interface {
	io.Writer
	Size() (width, height int, err error)
	IsTerminal() bool
}

// Capabilities describes the terminal features selected for rendering.
type Capabilities struct {
	Colors     ColorMode
	Attributes bool
}

// ColorMode controls color degradation.
type ColorMode int

const (
	ColorModeAuto ColorMode = iota
	ColorModeNone
	ColorMode16
	ColorMode256
	ColorModeTrueColor
)

// FileTerminal adapts an *os.File to the Terminal interface.
type FileTerminal struct {
	file *os.File
}

func NewFileTerminal(file *os.File) *FileTerminal {
	if file == nil {
		file = os.Stdout
	}
	return &FileTerminal{file: file}
}

func (t *FileTerminal) Write(p []byte) (int, error) {
	if t == nil || t.file == nil {
		return 0, errors.New("terminal is not initialized")
	}
	return t.file.Write(p)
}

func (t *FileTerminal) Size() (int, int, error) {
	if t == nil || t.file == nil {
		return 0, 0, errors.New("terminal is not initialized")
	}
	var window struct {
		rows, cols, xPixels, yPixels uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		t.file.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return 0, 0, errno
	}
	if window.cols == 0 || window.rows == 0 {
		return 0, 0, errors.New("invalid terminal size")
	}
	return int(window.cols), int(window.rows), nil
}

func (t *FileTerminal) IsTerminal() bool {
	if t == nil || t.file == nil {
		return false
	}
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		t.file.Fd(),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&termios)),
	)
	return errno == 0
}

func (t *FileTerminal) makeRaw() (func(), error) {
	if t == nil || t.file == nil {
		return nil, errors.New("terminal is not initialized")
	}
	var original syscall.Termios
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		t.file.Fd(),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&original)),
	); errno != 0 {
		return nil, errno
	}

	raw := original
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.IEXTEN
	// Keep ISIG so Ctrl-C still generates SIGINT.
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		t.file.Fd(),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&raw)),
	); errno != 0 {
		return nil, errno
	}
	return func() {
		_, _, _ = syscall.Syscall(
			syscall.SYS_IOCTL,
			t.file.Fd(),
			uintptr(syscall.TCSETS),
			uintptr(unsafe.Pointer(&original)),
		)
	}, nil
}

// Detect selects capabilities from the terminal and environment.
func Detect(term Terminal, mode ColorMode) Capabilities {
	if mode != ColorModeAuto {
		return Capabilities{Colors: mode, Attributes: true}
	}
	if os.Getenv("NO_COLOR") != "" {
		return Capabilities{Colors: ColorModeNone, Attributes: true}
	}
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorTerm == "truecolor" || colorTerm == "24bit" {
		return Capabilities{Colors: ColorModeTrueColor, Attributes: true}
	}
	termName := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(termName, "256") {
		return Capabilities{Colors: ColorMode256, Attributes: true}
	}
	if termName == "" || termName == "dumb" {
		return Capabilities{Colors: ColorModeNone, Attributes: false}
	}
	return Capabilities{Colors: ColorMode16, Attributes: true}
}
