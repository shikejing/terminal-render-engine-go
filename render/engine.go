package render

import (
	"io"
	"sync"
	"time"
)

// Engine 是并发安全的渲染内核：多个生产者提交状态快照，
// 引擎按帧节奏合并、差分并输出。
//
// 一致性保证：每次 Submit 提交的是一个完整状态快照，引擎某一帧
// 只渲染某一个已提交快照，绝不混合多个快照的字段。
//
// 背压策略：Submit 永不阻塞。同一帧窗口内的多次提交被合并，
// 仅保留最新快照（丢弃较旧的未渲染快照），因此慢终端不会拖垮生产者。
type Engine struct {
	r    *Renderer
	cfg  Config
	draw DrawFunc

	mu         sync.Mutex
	pending    State
	hasPending bool
	cols, rows int
	resized    bool

	notify   chan struct{} // 容量 1：合并同一帧窗口内的多次提交
	stopCh   chan struct{}
	stopOnce sync.Once
}

// Config 引擎配置。
type Config struct {
	FPS       int  // 帧率上限，<=0 表示每次提交立即渲染
	AltScreen bool // 是否使用备用屏幕
}

// State 是应用状态快照，由 Draw 回调绘制到 Buffer。
type State interface{}

// DrawFunc 把状态绘制到缓冲上。
type DrawFunc func(buf *Buffer, s State)

// NewEngine 创建引擎。caps 为终端能力，cols/rows 为初始尺寸。
func NewEngine(w io.Writer, caps Caps, cfg Config, draw DrawFunc, cols, rows int) *Engine {
	if !cfg.AltScreen {
		caps.AltScreen = false
	}
	return &Engine{
		r:      NewRenderer(w, caps, cols, rows),
		cfg:    cfg,
		draw:   draw,
		cols:   cols,
		rows:   rows,
		notify: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
}

// Submit 提交一个新状态快照，非阻塞；同一帧窗口内的多次提交被合并，
// 仅保留最新快照（丢弃较旧的未渲染快照）。
func (e *Engine) Submit(s State) {
	e.mu.Lock()
	e.pending = s
	e.hasPending = true
	e.mu.Unlock()
	select {
	case e.notify <- struct{}{}:
	default: // 已有待处理通知，本次提交被合并
	}
}

// Resize 通知终端尺寸变化，下一帧按新尺寸重排并全量重绘。
func (e *Engine) Resize(cols, rows int) {
	e.mu.Lock()
	e.cols, e.rows = cols, rows
	e.resized = true
	e.mu.Unlock()
	select {
	case e.notify <- struct{}{}:
	default:
	}
}

// Stop 请求渲染循环停止。Run 返回前会渲染最后一个待渲染快照并恢复终端。
func (e *Engine) Stop() {
	e.stopOnce.Do(func() { close(e.stopCh) })
}

// Run 启动渲染循环，直到 Stop 被调用；返回时终端已恢复。
// 即使绘制回调 panic，终端也会被恢复。
func (e *Engine) Run() (err error) {
	defer func() {
		// 恢复终端，保证进程退出或异常后终端可用
		if eErr := e.r.Exit(); eErr != nil && err == nil {
			err = eErr
		}
	}()

	if e.cfg.FPS <= 0 {
		// 不限帧率：每次提交立即渲染
		for {
			select {
			case <-e.stopCh:
				e.renderFrame() // 渲染最后一个待渲染快照，不丢帧
				return nil
			case <-e.notify:
				e.renderFrame()
			}
		}
	}

	// 限帧率：tick 统一出帧，同一帧窗口内的提交被合并
	t := time.NewTicker(time.Second / time.Duration(e.cfg.FPS))
	defer t.Stop()
	for {
		select {
		case <-e.stopCh:
			e.renderFrame() // 渲染最后一个待渲染快照，不丢帧
			return nil
		case <-e.notify: // 仅作唤醒，实际出帧由 tick 控制
		case <-t.C:
			e.renderFrame()
		}
	}
}

// renderFrame 取出一个待渲染快照并渲染一帧。
func (e *Engine) renderFrame() {
	e.mu.Lock()
	resized := e.resized
	cols, rows := e.cols, e.rows
	var s State
	if e.hasPending {
		s = e.pending
		e.pending = nil
		e.hasPending = false
	}
	if resized {
		e.resized = false
	}
	e.mu.Unlock()

	if resized {
		e.r.Resize(cols, rows)
	}
	if s == nil && !resized {
		return
	}
	buf := NewBuffer(cols, rows)
	if s != nil {
		e.drawSafe(buf, s)
	}
	_ = e.r.Render(buf)
}

// drawSafe 调用绘制回调并恢复 panic，避免单个异常帧杀死渲染循环。
func (e *Engine) drawSafe(buf *Buffer, s State) {
	defer func() { _ = recover() }()
	e.draw(buf, s)
}
