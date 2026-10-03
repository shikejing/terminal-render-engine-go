package render

import (
	"bytes"
	"fmt"
	"io"
)

// Renderer 负责把 Buffer 差分写入终端输出流。
// 每一帧先在内存中拼好完整字节序列再一次写入，
// 保证不会出现半个控制序列。
type Renderer struct {
	w        io.Writer
	caps     Caps
	prev     *Buffer
	cols     int
	rows     int
	entered  bool
	curStyle Style
	hasStyle bool
}

// NewRenderer 创建渲染器，cols/rows 为终端初始尺寸。
func NewRenderer(w io.Writer, caps Caps, cols, rows int) *Renderer {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return &Renderer{w: w, caps: caps, cols: cols, rows: rows}
}

// Size 返回渲染器当前使用的终端尺寸。
func (r *Renderer) Size() (cols, rows int) { return r.cols, r.rows }

// Resize 通知渲染器终端尺寸变化，下一帧按新尺寸全量重绘。
func (r *Renderer) Resize(cols, rows int) {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols == r.cols && rows == r.rows {
		return
	}
	r.cols, r.rows = cols, rows
	r.prev = nil // 尺寸变化后强制全量重绘，避免残影
}

// Enter 进入绘制模式（备用屏幕、隐藏光标等）。
func (r *Renderer) Enter() error {
	if r.entered {
		return nil
	}
	var b bytes.Buffer
	if r.caps.AltScreen {
		b.WriteString("\x1b[?1049h") // 进入备用屏幕
	}
	b.WriteString("\x1b[?25l") // 隐藏光标
	b.WriteString("\x1b[2J")   // 清屏
	b.WriteString("\x1b[H")    // 光标归位
	_, err := r.w.Write(b.Bytes())
	if err == nil {
		r.entered = true
	}
	return err
}

// Exit 恢复终端到进入前的状态。可重复调用。
func (r *Renderer) Exit() error {
	if !r.entered {
		return nil
	}
	var b bytes.Buffer
	b.WriteString("\x1b[0m")   // 复位样式
	b.WriteString("\x1b[?25h") // 显示光标
	if r.caps.AltScreen {
		b.WriteString("\x1b[?1049l") // 离开备用屏幕
	}
	_, err := r.w.Write(b.Bytes())
	r.entered = false
	r.prev = nil
	return err
}

// Render 把 buf 与上一帧差分后写入输出，只写变化单元格。
// buf 尺寸会被裁剪到终端尺寸内，防止越界输出。
func (r *Renderer) Render(buf *Buffer) error {
	if !r.entered {
		if err := r.Enter(); err != nil {
			return err
		}
	}
	var frame bytes.Buffer
	if r.caps.SyncUpdate {
		frame.WriteString("\x1b[?2026h") // BSU：开始同步更新
	}
	if r.prev == nil {
		// 全量重绘前先清屏，消除尺寸缩小后的残影
		frame.WriteString("\x1b[0m\x1b[2J")
		r.hasStyle = false
	}

	diff := buf.Diff(r.prev)
	cx, cy := -1, -1 // 当前光标位置（未知）
	for _, p := range diff {
		if p.X >= r.cols || p.Y >= r.rows {
			continue // 裁剪到终端尺寸内
		}
		c := buf.Get(p.X, p.Y)
		if c.Width == 0 {
			continue // 宽字符延续格由主格一并输出
		}
		if cx != p.X || cy != p.Y {
			fmt.Fprintf(&frame, "\x1b[%d;%dH", p.Y+1, p.X+1)
		}
		r.writeStyle(&frame, c.Style)
		frame.WriteString(c.Cluster)
		cx, cy = p.X+c.Width, p.Y
	}
	if r.caps.SyncUpdate {
		frame.WriteString("\x1b[?2026l") // ESU：结束同步更新
	}
	if frame.Len() == 0 {
		r.prev = buf.Clone()
		return nil
	}
	if _, err := r.w.Write(frame.Bytes()); err != nil {
		return err
	}
	r.prev = buf.Clone()
	return nil
}

// writeStyle 当目标样式与当前样式不同时输出 SGR 序列。
func (r *Renderer) writeStyle(frame *bytes.Buffer, st Style) {
	if r.hasStyle && r.curStyle.Equal(st) {
		return
	}
	frame.WriteString("\x1b[0m") // 先复位，再按需开启属性
	if st.Bold {
		frame.WriteString("\x1b[1m")
	}
	if st.Underline {
		frame.WriteString("\x1b[4m")
	}
	if st.Reverse {
		frame.WriteString("\x1b[7m")
	}
	r.writeColor(frame, st.Fg, true)
	r.writeColor(frame, st.Bg, false)
	r.curStyle = st
	r.hasStyle = true
}

func (r *Renderer) writeColor(frame *bytes.Buffer, c Color, fg bool) {
	c = downgrade(c, r.caps.Color)
	switch c.Mode {
	case ColorNone:
		// 无颜色能力：不输出任何颜色序列
	case Color16:
		v := int(c.Value) & 0xF
		base := 30
		if !fg {
			base = 40
		}
		if v >= 8 {
			fmt.Fprintf(frame, "\x1b[%dm", base+60+v-8)
		} else {
			fmt.Fprintf(frame, "\x1b[%dm", base+v)
		}
	case Color256:
		if fg {
			fmt.Fprintf(frame, "\x1b[38;5;%dm", c.Value&0xFF)
		} else {
			fmt.Fprintf(frame, "\x1b[48;5;%dm", c.Value&0xFF)
		}
	case ColorTrue:
		rr, gg, bb := c.Value>>16&0xFF, c.Value>>8&0xFF, c.Value&0xFF
		if fg {
			fmt.Fprintf(frame, "\x1b[38;2;%d;%d;%dm", rr, gg, bb)
		} else {
			fmt.Fprintf(frame, "\x1b[48;2;%d;%d;%dm", rr, gg, bb)
		}
	}
}
