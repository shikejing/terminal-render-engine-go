// demo 演示渲染引擎：中英文混排边框、计数器并发更新、终端尺寸自适应。
// 运行：go run ./cmd/demo   （在真实终端中，按 Ctrl-C 退出）
package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shikejing/terminal-render-engine-go/render"
)

type appState struct {
	counter int64
	cols    int
	rows    int
}

func draw(buf *render.Buffer, s render.State) {
	st := s.(appState)
	w, h := buf.Size()
	border := render.Style{Fg: render.Color{Mode: render.Color256, Value: 39}}
	title := render.Style{Fg: render.Color{Mode: render.ColorTrue, Value: 0xFF8800}, Bold: true}

	// 边框（宽字符环境下不错位）
	if w >= 2 && h >= 2 {
		buf.SetString(0, 0, "┌", border)
		buf.SetString(w-1, 0, "┐", border)
		buf.SetString(0, h-1, "└", border)
		buf.SetString(w-1, h-1, "┘", border)
		for x := 1; x < w-1; x++ {
			buf.SetString(x, 0, "─", border)
			buf.SetString(x, h-1, "─", border)
		}
		for y := 1; y < h-1; y++ {
			buf.SetString(0, y, "│", border)
			buf.SetString(w-1, y, "│", border)
		}
	}
	buf.SetString(2, 1, "终端渲染引擎 Demo", title)
	buf.SetString(2, 2, fmt.Sprintf("计数: %d", st.counter), render.Style{})
	buf.SetString(2, 3, fmt.Sprintf("尺寸: %dx%d  中英文混排 mixed 测试", st.cols, st.rows), render.Style{})
}

func main() {
	caps := render.DetectCaps(os.Getenv)
	cols, rows := 80, 24
	if ws, err := terminalSize(); err == nil {
		cols, rows = ws[0], ws[1]
	}

	e := render.NewEngine(os.Stdout, caps, render.Config{FPS: 60, AltScreen: true}, draw, cols, rows)

	var counter int64
	// 多个生产者并发提交
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
					n := atomic.AddInt64(&counter, 1)
					e.Submit(appState{counter: n, cols: cols, rows: rows})
					time.Sleep(3 * time.Millisecond)
				}
			}
		}()
	}

	// 监听尺寸变化与退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range sig {
			if s == syscall.SIGWINCH {
				if ws, err := terminalSize(); err == nil {
					cols, rows = ws[0], ws[1]
					e.Resize(cols, rows)
				}
				continue
			}
			e.Stop()
		}
	}()

	_ = e.Run()
	close(stop)
}

// terminalSize 通过 stty 获取终端尺寸（避免引入外部依赖）。
func terminalSize() ([2]int, error) {
	var out [2]int
	c := execCommand()
	var rows, cols int
	if _, err := fmt.Sscanf(c, "%d %d", &rows, &cols); err != nil {
		return out, err
	}
	out[0], out[1] = cols, rows
	return out, nil
}
