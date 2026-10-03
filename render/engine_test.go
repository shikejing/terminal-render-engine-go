package render

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// appState 是一个示例应用状态：两个字段必须保持一致（name 含 id）。
type appState struct {
	id   int
	name string
}

func drawApp(buf *Buffer, s State) {
	st := s.(appState)
	buf.SetString(0, 0, fmt.Sprintf("%d:%s", st.id, st.name), Style{})
}

func newState(i int) appState {
	return appState{id: i, name: fmt.Sprintf("item-%d", i)}
}

func TestEngineConcurrentSubmitConsistent(t *testing.T) {
	rec := &writeRecorder{}
	e := NewEngine(rec, testCaps(), Config{FPS: 200}, drawApp, 40, 5)
	done := make(chan struct{})
	go func() { _ = e.Run(); close(done) }()

	// 多个生产者并发提交
	var wg sync.WaitGroup
	for p := 0; p < 8; p++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				e.Submit(newState(base*1000 + i))
			}
		}(p)
	}
	wg.Wait()
	// 等渲染追上
	time.Sleep(50 * time.Millisecond)
	e.Stop()
	<-done

	// 校验所有输出帧：任何一帧的内容都必须是某个一致快照，
	// 即输出中不得出现 id 与 name 不匹配的组合。
	out := rec.all()
	// 精确检查：输出中出现的每个 "N:item-M" 片段必须满足 N==M
	rest := out
	for {
		i := strings.Index(rest, ":item-")
		if i < 0 {
			break
		}
		// 向前找数字起点
		j := i - 1
		for j >= 0 && rest[j] >= '0' && rest[j] <= '9' {
			j--
		}
		idStr := rest[j+1 : i]
		k := i + 6
		for k < len(rest) && rest[k] >= '0' && rest[k] <= '9' {
			k++
		}
		nameStr := rest[i+6 : k]
		if idStr != nameStr {
			t.Errorf("inconsistent frame: id=%s name=%s", idStr, nameStr)
		}
		rest = rest[k:]
	}
}

func TestEngineCoalescing(t *testing.T) {
	rec := &writeRecorder{}
	// FPS=20 → 帧窗口 50ms，窗口内 100 次提交应被合并
	e := NewEngine(rec, testCaps(), Config{FPS: 20}, drawApp, 40, 5)
	done := make(chan struct{})
	go func() { _ = e.Run(); close(done) }()
	for i := 0; i < 100; i++ {
		e.Submit(newState(i))
	}
	time.Sleep(120 * time.Millisecond)
	e.Stop()
	<-done
	// 最终画面必须包含最后提交的快照（合并保留最新）
	if !strings.Contains(rec.all(), "99:item-99") {
		t.Errorf("latest snapshot not rendered: %q", rec.all())
	}
}

func TestEngineSubmitNeverBlocks(t *testing.T) {
	rec := &writeRecorder{}
	// 不启动 Run，Submit 也不得阻塞（背压：丢弃未渲染快照）
	e := NewEngine(rec, testCaps(), Config{FPS: 1}, drawApp, 10, 2)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10000; i++ {
			e.Submit(newState(i))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Submit blocked without consumer")
	}
	e.Stop()
}

func TestEngineResize(t *testing.T) {
	rec := &writeRecorder{}
	e := NewEngine(rec, testCaps(), Config{FPS: 100}, drawApp, 10, 3)
	done := make(chan struct{})
	go func() { _ = e.Run(); close(done) }()
	e.Submit(newState(1))
	time.Sleep(30 * time.Millisecond)
	e.Resize(20, 6)
	e.Submit(newState(2))
	time.Sleep(50 * time.Millisecond)
	e.Stop()
	<-done
	out := rec.all()
	// resize 后应清屏重绘且包含新内容
	if !strings.Contains(out, "\x1b[2J") {
		t.Errorf("resize did not clear screen")
	}
	if !strings.Contains(out, "2:item-2") {
		t.Errorf("post-resize content missing: %q", out)
	}
}

func TestEngineStopRestoresTerminal(t *testing.T) {
	rec := &writeRecorder{}
	e := NewEngine(rec, testCaps(), Config{FPS: 100, AltScreen: true}, drawApp, 10, 3)
	done := make(chan struct{})
	go func() { _ = e.Run(); close(done) }()
	e.Submit(newState(1))
	time.Sleep(30 * time.Millisecond)
	e.Stop()
	<-done
	out := rec.all()
	if !strings.Contains(out, "\x1b[?25h") || !strings.Contains(out, "\x1b[?1049l") {
		t.Errorf("terminal not restored after stop: %q", out)
	}
}

func TestEngineDrawPanicStillRestores(t *testing.T) {
	rec := &writeRecorder{}
	bad := func(buf *Buffer, s State) { panic("boom") }
	e := NewEngine(rec, testCaps(), Config{FPS: 0}, bad, 10, 3)
	done := make(chan struct{})
	go func() { _ = e.Run(); close(done) }()
	e.Submit(newState(1))
	time.Sleep(30 * time.Millisecond)
	e.Stop()
	<-done
	if !strings.Contains(rec.all(), "\x1b[?25h") {
		t.Errorf("terminal not restored after draw panic")
	}
}
