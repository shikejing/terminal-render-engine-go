# terminal-render-engine-go

一个终端界面渲染引擎：把应用状态以字符网格的形式稳定地绘制到终端上，
作为可复用的渲染内核供多个界面共用。仅依赖 Go 标准库。

## 架构

```
生产者 goroutine ─┐
生产者 goroutine ─┼─ Submit(State) ─> Engine（合并/背压/帧节奏）
生产者 goroutine ─┘                        │
                                           v
                              DrawFunc(State) -> Buffer（字符网格）
                                           │
                                           v
                              Renderer（与上一帧 Diff）-> 终端字节流
```

- `render.Cell` / `render.Buffer`：字符网格。每个单元格保存字素簇、样式与占用列宽；
  宽字符的第二个单元格为延续格（`Width == 0`）。
- `render.Renderer`：把 Buffer 与上一帧差分后写入终端。
- `render.Engine`：并发安全的渲染内核，负责合并提交、帧节奏、尺寸变化与终端恢复。
- `render.Caps` / `DetectCaps`：终端能力探测与颜色降级。

## 渲染语义

1. **局部刷新**：每帧只写入与上一帧不同的单元格；未变化区域不重复输出，
   不整屏重绘。帧内容先在内存拼好，一次 `Write` 写出，不会出现半个控制序列；
   支持同步更新（BSU/ESU）的终端会包裹 `\x1b[?2026h/l` 进一步防闪烁。
2. **尺寸变化**：`Engine.Resize(cols, rows)` 后下一帧按新尺寸重建缓冲、
   清屏并全量重绘，无错位、残影或越界；停止前会把最后一个待渲染快照渲染出去，不丢帧。
3. **宽度正确**：宽字符（CJK、全角、Emoji）占 2 列，组合字符与零宽字符占 0 列，
   控制字符被丢弃。宽字符在右边界放不下时截断；覆盖宽字符任意一半时另一半自动置空，
   中英文混排时边框与对齐不错位。
4. **一致性快照**：`Submit` 提交完整状态快照，一帧只渲染某一个已提交快照，
   画面不会出现字段级混合。
5. **帧节奏与背压**：`Config.FPS` 限制帧率；同一帧窗口内的多次提交被合并，
   仅保留最新快照（丢弃较旧的未渲染快照）。`Submit` 永不阻塞，慢终端不会拖垮生产者。
6. **终端恢复**：`Run` 返回时（含绘制回调 panic）自动恢复光标、样式与备用屏幕。
7. **能力降级**：按 `COLORTERM`/`TERM`/`NO_COLOR` 探测真彩色 / 256 色 / 16 色 / 无颜色，
   颜色自动降级（truecolor → 256 → 16），能力不足的终端不会收到无法识别的控制序列。

## 配置项

| 配置 | 说明 |
|---|---|
| `Config.FPS` | 帧率上限；`<= 0` 表示每次提交立即渲染 |
| `Config.AltScreen` | 是否使用备用屏幕（退出后恢复原屏幕内容） |
| `Caps.Color` | 颜色能力：`ColorNone` / `Color16` / `Color256` / `ColorTrue`，可手动覆盖探测结果 |
| `Caps.SyncUpdate` | 是否使用同步更新序列防闪烁 |

## 用法

```go
caps := render.DetectCaps(os.Getenv)
e := render.NewEngine(os.Stdout, caps,
    render.Config{FPS: 60, AltScreen: true},
    func(buf *render.Buffer, s render.State) {
        st := s.(MyState)
        buf.SetString(0, 0, fmt.Sprintf("计数 %d", st.N), render.Style{})
    }, 80, 24)

go e.Run()
e.Submit(MyState{N: 1})   // 任意多个 goroutine 可并发调用
e.Resize(100, 40)         // 终端尺寸变化时调用
e.Stop()                  // 停止并恢复终端
```

演示程序：`go run ./cmd/demo`（真实终端中运行，Ctrl-C 退出）。

## 验证方法

```sh
go test ./render/          # 单元测试
go test ./render/ -race    # 含竞态检测（并发提交场景）
go vet ./...
```

测试覆盖：

- **单元格差分**（`buffer_test.go` / `renderer_test.go`）：相同缓冲零差异、
  宽字符修改产生主格+延续格差异、尺寸变化全量差异、仅样式变化触发重绘、
  未变化帧不输出单元格。
- **宽字符宽度**（`width_test.go`）：CJK/全角/Emoji 占 2 列，组合字符/ZWJ 占 0 列，
  字素簇切分，中英文混排宽度。
- **尺寸变化**（`renderer_test.go` / `engine_test.go`）：resize 后清屏全量重绘、
  缓冲大于终端时裁剪不越界。
- **并发更新**（`engine_test.go`，`-race` 下运行）：8 个生产者并发提交 1600 个快照，
  校验所有输出帧均为一致快照；提交合并后最终画面为最新快照；
  无消费者时 `Submit` 不阻塞（背压）。
- **能力降级**（`renderer_test.go`）：真彩色/256/16/无颜色四种终端的序列输出与降级，
  `TERM`/`COLORTERM`/`NO_COLOR` 探测。
