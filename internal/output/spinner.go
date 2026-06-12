package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Spinner 是 scan 期间在 stderr 上显示的"正在扫描 1245 files..."进度反馈。
//
// 设计要点（plan-design-review D5 决议）：
//   - stderr only：不污染 stdout，pipe 友好
//   - 仅 TTY + 非 NO_COLOR 启用，其他场景 Start/Stop 都是 no-op
//   - 200ms throttle，省 CPU 又够流畅
//   - braille 字符 ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏（10 帧）
//   - 结束时 \r 清行，让 dashboard 输出从干净的行开始
//
// 计数原子，调用方在 walker / extract 路径上 IncFiles() / IncExtracted() / SetCurrent()。
type Spinner struct {
	out       io.Writer
	files     atomic.Int64
	extracted atomic.Int64
	current   atomic.Value // string: basename of file currently being extracted
	stopCh    chan struct{}
	doneCh    chan struct{}
	enabled   bool
	stopOnce  sync.Once
}

// NewSpinner 构造一个 spinner。
// out=nil 时用 os.Stderr。enabled 综合 TTY/NO_COLOR/IMFD_NO_SPINNER 探测。
func NewSpinner(out io.Writer) *Spinner {
	if out == nil {
		out = os.Stderr
	}
	stderr := os.Stderr
	enabled := IsTTY(stderr) && !NoColor() && os.Getenv("IMFD_NO_SPINNER") == ""
	return &Spinner{
		out:     out,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		enabled: enabled,
	}
}

// IncFiles 把"已发现文件数"+1，并发安全。
func (s *Spinner) IncFiles() { s.files.Add(1) }

// IncExtracted 把"已完成提取的 record 数"+1，并发安全。
func (s *Spinner) IncExtracted() { s.extracted.Add(1) }

// SetCurrent 记录当前正在提取的文件名（basename），便于用户判断是慢还是卡死。
// 并发安全；传空字符串清除显示。
func (s *Spinner) SetCurrent(basename string) { s.current.Store(basename) }

// Start 启动后台刷新 goroutine。enabled=false 时是 no-op。
func (s *Spinner) Start() {
	if !s.enabled {
		close(s.doneCh)
		return
	}
	go s.loop()
}

// Stop 停止 spinner 并清行。enabled=false 时是 no-op。
// idempotent：多次调用安全（用 sync.Once 保护），允许 caller 显式 Stop + defer Stop 兜底。
// 必须在 dashboard 输出前调用，否则会和数据混在一起。
func (s *Spinner) Stop() {
	if !s.enabled {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		<-s.doneCh
		// 清行：\r 回到行首 + \033[K 清到行末
		fmt.Fprint(s.out, "\r\033[K")
	})
}

const spinnerInterval = 200 * time.Millisecond

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

func (s *Spinner) loop() {
	defer close(s.doneCh)
	tick := time.NewTicker(spinnerInterval)
	defer tick.Stop()
	frame := 0
	for {
		select {
		case <-s.stopCh:
			return
		case <-tick.C:
			s.render(frame)
			frame = (frame + 1) % len(spinnerFrames)
		}
	}
}

func (s *Spinner) render(frame int) {
	f := s.files.Load()
	e := s.extracted.Load()
	inFlight := f - e

	cur, _ := s.current.Load().(string)
	// 文件名截断：超过 32 字符时保留后 29 字符（最有辨识度的部分）
	if len(cur) > 32 {
		cur = "…" + cur[len(cur)-31:]
	}

	var suffix string
	if inFlight > 0 && cur != "" {
		suffix = fmt.Sprintf(" · %d in flight · %s", inFlight, cur)
	} else if inFlight > 0 {
		suffix = fmt.Sprintf(" · %d in flight", inFlight)
	}

	// \r 回到行首，\033[K 清到行末，避免上一帧残留
	fmt.Fprintf(s.out, "\r\033[K%c scanned %d · %d extracted%s", spinnerFrames[frame], f, e, suffix)
}
