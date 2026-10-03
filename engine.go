package terminalrender

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ViewFunc draws an immutable state snapshot into a buffer.
type ViewFunc[S any] func(state S, buffer *Buffer)

// Config controls engine timing and terminal behavior.
type Config struct {
	FrameInterval time.Duration
	ColorMode     ColorMode
	AltScreen     bool
	HideCursor    bool
	RawMode       bool
	Attributes    bool
	Terminal      Terminal
}

// Engine coalesces concurrent state updates and renders snapshots.
type Engine[S any] struct {
	view ViewFunc[S]

	configMu sync.Mutex
	config   Config

	stateMu  sync.Mutex
	latest   S
	hasState bool
	dirty    bool

	resizeCh chan struct{}
	cancelMu sync.Mutex
	cancel   context.CancelFunc
}

// Option configures an Engine.
type Option func(*Config)

func WithFrameInterval(interval time.Duration) Option {
	return func(c *Config) { c.FrameInterval = interval }
}

func WithColorMode(mode ColorMode) Option {
	return func(c *Config) { c.ColorMode = mode }
}

func WithTerminal(term Terminal) Option {
	return func(c *Config) { c.Terminal = term }
}

func WithAltScreen(enabled bool) Option {
	return func(c *Config) { c.AltScreen = enabled }
}

func WithHideCursor(enabled bool) Option {
	return func(c *Config) { c.HideCursor = enabled }
}

func WithRawMode(enabled bool) Option {
	return func(c *Config) { c.RawMode = enabled }
}

func WithAttributes(enabled bool) Option {
	return func(c *Config) { c.Attributes = enabled }
}

// New creates an engine. The view must be safe to call with distinct state
// values and must not retain the buffer after returning.
func New[S any](view ViewFunc[S], opts ...Option) *Engine[S] {
	cfg := Config{
		FrameInterval: 16 * time.Millisecond,
		ColorMode:     ColorModeAuto,
		AltScreen:     true,
		HideCursor:    true,
		RawMode:       true,
		Attributes:    true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.FrameInterval <= 0 {
		cfg.FrameInterval = 16 * time.Millisecond
	}
	return &Engine[S]{
		view:     view,
		config:   cfg,
		resizeCh: make(chan struct{}, 1),
	}
}

// Submit publishes a state snapshot. It never blocks: updates arriving while a
// frame is being rendered are coalesced into the latest snapshot.
func (e *Engine[S]) Submit(state S) {
	if e == nil {
		return
	}
	if cloner, ok := any(state).(interface{ Clone() S }); ok {
		state = cloner.Clone()
	}
	e.stateMu.Lock()
	e.latest = state
	e.hasState = true
	e.dirty = true
	e.stateMu.Unlock()
}

// NotifyResize requests an immediate terminal size check.
func (e *Engine[S]) NotifyResize() {
	if e == nil {
		return
	}
	select {
	case e.resizeCh <- struct{}{}:
	default:
	}
}

// Stop cancels a running engine. Run restores the terminal before returning.
func (e *Engine[S]) Stop() {
	if e == nil {
		return
	}
	e.cancelMu.Lock()
	cancel := e.cancel
	e.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Run renders until ctx is canceled or an interrupt signal arrives.
func (e *Engine[S]) Run(ctx context.Context) error {
	if e == nil || e.view == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	e.cancelMu.Lock()
	e.cancel = stop
	e.cancelMu.Unlock()
	defer func() {
		e.cancelMu.Lock()
		e.cancel = nil
		e.cancelMu.Unlock()
	}()

	e.configMu.Lock()
	cfg := e.config
	e.configMu.Unlock()
	if cfg.Terminal == nil {
		cfg.Terminal = NewFileTerminal(os.Stdout)
	}

	term := cfg.Terminal
	caps := Detect(term, cfg.ColorMode)
	if !cfg.Attributes {
		caps.Attributes = false
	}
	renderer := NewRenderer(term, caps)
	interactive := term.IsTerminal()
	if interactive && cfg.RawMode {
		if fileTerm, ok := term.(*FileTerminal); ok {
			restore, err := fileTerm.makeRaw()
			if err != nil {
				return err
			}
			defer restore()
		}
	}
	if interactive {
		if err := renderer.Enter(cfg.AltScreen, cfg.HideCursor); err != nil {
			return err
		}
		defer func() {
			_ = renderer.Leave(cfg.AltScreen, cfg.HideCursor)
		}()
	}

	width, height := 80, 24
	if w, h, err := term.Size(); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}
	back := NewBuffer(width, height)
	front := NewBuffer(width, height)
	var previous *Buffer
	force := true

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	render := func() error {
		e.stateMu.Lock()
		state := e.latest
		hasState := e.hasState
		e.dirty = false
		e.stateMu.Unlock()
		if !hasState {
			return nil
		}
		back.Clear()
		e.view(state, back)
		if err := renderer.Render(previous, back, force); err != nil {
			return err
		}
		previous = back
		back, front = front, back
		force = false
		return nil
	}

	applyResize := func() error {
		w, h, err := term.Size()
		if err != nil || w <= 0 || h <= 0 {
			return nil
		}
		if back != nil && w == back.width && h == back.height && previous != nil {
			return nil
		}
		back = NewBuffer(w, h)
		front = NewBuffer(w, h)
		previous = nil
		force = true
		return render()
	}

	if err := render(); err != nil {
		return err
	}

	ticker := time.NewTicker(cfg.FrameInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-winch:
			if err := applyResize(); err != nil {
				return err
			}
		case <-e.resizeCh:
			if err := applyResize(); err != nil {
				return err
			}
		case <-ticker.C:
			e.stateMu.Lock()
			dirty := e.dirty
			e.stateMu.Unlock()
			if dirty {
				if err := render(); err != nil {
					return err
				}
			}
		}
	}
}
