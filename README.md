# terminal-render-engine-go

一个可复用的终端字符网格渲染内核。它把应用状态渲染成固定大小的字符网格，并以局部刷新的方式输出到终端。

## 特性

- **局部刷新**：逐帧比较单元格内容、样式和占位宽度，只输出变化的单元格。
- **整帧写入**：每一帧的光标定位、样式和文本先组装到内存，再一次性写入，避免输出半个控制序列。
- **尺寸自适应**：监听 `SIGWINCH`，也可通过 `NotifyResize()` 主动触发；尺寸变化后清空旧网格并按新尺寸重排。
- **正确占位**：支持 ASCII、中英文、宽字符、组合字符、零宽字符和常见 emoji ZWJ 序列。
- **并发快照**：多个生产者可以并发调用 `Submit`，引擎在固定帧窗口内合并更新，最终画面是某个已提交状态的一致快照。
- **慢输出背压**：`Submit` 永不阻塞；终端写入较慢时，未渲染的中间状态会被更新的状态合并。
- **终端恢复**：进入 alt screen、隐藏光标和原始模式后，会在上下文取消、进程中断或渲染错误时恢复终端。
- **颜色降级**：自动探测真彩色、256 色、16 色和无颜色能力，并将 RGB/256 色最近似降级。

## 安装

```bash
go get github.com/shikejing/terminal-render-engine-go
```

## 快速开始

```go
package main

import (
	"context"
	"time"

	terminalrender "github.com/shikejing/terminal-render-engine-go"
)

type AppState struct {
	Message string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	view := func(state AppState, buffer *terminalrender.Buffer) {
		style := terminalrender.Style{
			FG: terminalrender.RGB(80, 200, 120),
		}
		buffer.Set(0, 0, state.Message, style)
	}

	engine := terminalrender.New(
		view,
		terminalrender.WithFrameInterval(16*time.Millisecond),
	)

	go func() {
		_ = engine.Run(ctx)
	}()

	engine.Submit(AppState{Message: "hello"})
	time.Sleep(time.Second)
	cancel()
}
```

## 渲染语义

### 状态快照

`Submit(state)` 保存状态的一个副本。状态应为不可变值；如果状态包含需要深拷贝的引用类型，可以实现 `Clone() T` 方法，引擎会在提交时调用：

```go
func (s AppState) Clone() AppState {
	copy := s
	copy.Values = append([]string(nil), s.Values...)
	return copy
}
```

渲染协程只在帧边界读取最新状态，因此不会出现一个状态的部分字段与另一个状态的部分字段混合的问题。

### 帧合并与背压

引擎按 `FrameInterval` 节拍渲染。同一个帧窗口内的多次 `Submit` 会被合并为最新状态；如果终端输出较慢，`Submit` 也不会阻塞生产者。

### 局部刷新

渲染器保存上一帧网格。下一帧只输出满足以下任一条件的单元格：

- 文本内容变化；
- 样式变化；
- 占位宽度变化。

连续的变化单元格会共用同一个光标定位序列。尺寸变化或首帧会强制整屏重绘。

### 字符宽度

`Buffer.Set` 会把文本切分为可显示 grapheme cluster：

- 中文、韩文、日文和 emoji 等宽字符占 2 列；
- 组合字符、零宽空格、ZWJ 和变体选择符占 0 列，并挂载到当前单元格；
- emoji ZWJ 序列作为一个整体占 2 列；
- 超出右边界的宽字符不会写入，避免越界。

### 终端恢复

`Run` 会在以下情况恢复终端：

- `ctx` 被取消；
- 收到 `SIGINT` 或 `SIGTERM`；
- 终端写入或尺寸读取发生错误。

恢复内容包括重置样式、显示光标和离开 alt screen。原始终端模式也会通过 `termios` 恢复。

## 配置项

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `WithFrameInterval` | `16ms` | 渲染节拍，约 60fps |
| `WithColorMode` | `ColorModeAuto` | 颜色模式：自动、无、16 色、256 色、真彩色 |
| `WithTerminal` | `os.Stdout` | 自定义终端输出与尺寸来源 |
| `WithAltScreen` | `true` | 是否使用备用屏幕 |
| `WithHideCursor` | `true` | 是否隐藏光标 |
| `WithRawMode` | `true` | 是否进入原始终端模式 |
| `WithAttributes` | `true` | 是否输出粗体、下划线等属性 SGR |

颜色自动探测规则：

- `NO_COLOR` 非空：不输出颜色；
- `COLORTERM=truecolor` 或 `COLORTERM=24bit`：真彩色；
- `TERM` 包含 `256`：256 色；
- `TERM=dumb` 或为空：不输出颜色和属性；
- 其他情况：16 色。

能力不足时，RGB 会转换为最接近的 256 色或 16 色，256 色会转换为最接近的 16 色。

## 验证方法

运行全部测试：

```bash
go test ./...
```

运行竞态检查：

```bash
go test -race ./...
```

运行静态检查：

```bash
go vet ./...
```

测试覆盖：

- 单元格局部差分与无变化帧；
- 宽字符、组合字符、零宽字符和 emoji ZWJ 序列；
- 缓冲区越界与填充；
- RGB 到 256 色/16 色的降级；
- 多生产者并发提交的快照一致性；
- 慢输出下的非阻塞提交；
- 终端尺寸变化后的整屏重绘；
- 交互式终端的 alt screen、光标和模式恢复。
