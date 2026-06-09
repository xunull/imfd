package output

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

// 使用 bytes.Buffer + 手动强制 enabled 来在非 TTY 测试环境下也能验证 spinner 输出
type spinnerHarness struct {
	*Spinner
	buf *bytes.Buffer
	mu  sync.Mutex
}

func newSpinnerHarness() *spinnerHarness {
	buf := &bytes.Buffer{}
	s := &Spinner{
		out:     buf,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		enabled: true,
	}
	return &spinnerHarness{Spinner: s, buf: buf}
}

func (h *spinnerHarness) output() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.buf.String()
}

func TestSpinner_NoOpWhenDisabled(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &Spinner{out: buf, stopCh: make(chan struct{}), doneCh: make(chan struct{}), enabled: false}
	s.Start()
	s.IncFiles()
	s.IncExtracted()
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	if buf.Len() != 0 {
		t.Errorf("disabled spinner should write nothing, got %q", buf.String())
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	h := newSpinnerHarness()
	h.Start()
	time.Sleep(50 * time.Millisecond)
	// 两次 Stop 不该 panic
	h.Stop()
	h.Stop()
}

func TestSpinner_AtomicCounters_RaceSafe(t *testing.T) {
	h := newSpinnerHarness()
	h.Start()
	defer h.Stop()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.IncFiles()
			h.IncExtracted()
		}()
	}
	wg.Wait()

	// 等一帧 + 一点 buffer
	time.Sleep(spinnerInterval + 50*time.Millisecond)
	h.Stop()

	if got := h.Spinner.files.Load(); got != 100 {
		t.Errorf("files counter: want 100, got %d", got)
	}
	if got := h.Spinner.extracted.Load(); got != 100 {
		t.Errorf("extracted counter: want 100, got %d", got)
	}
}

func TestSpinner_RenderShowsCounts(t *testing.T) {
	h := newSpinnerHarness()
	h.IncFiles()
	h.IncFiles()
	h.IncFiles()
	h.IncExtracted()
	h.Spinner.render(0)
	out := h.output()
	if !bytes.Contains([]byte(out), []byte("3 files")) {
		t.Errorf("want '3 files' in render output, got %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("1 extracted")) {
		t.Errorf("want '1 extracted', got %q", out)
	}
}

func TestSpinner_StartStopNotEnabled_ClosesDoneCh(t *testing.T) {
	// enabled=false 时 Start 直接关 doneCh，确保不死等
	buf := &bytes.Buffer{}
	s := &Spinner{out: buf, stopCh: make(chan struct{}), doneCh: make(chan struct{}), enabled: false}
	s.Start()
	select {
	case <-s.doneCh:
		// ok
	case <-time.After(100 * time.Millisecond):
		t.Error("disabled Start should close doneCh immediately")
	}
}

// 1000 个 goroutine 同时 inc，确保 race 检测下计数仍正确
func TestSpinner_CountersUseAtomic(t *testing.T) {
	var s Spinner
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(2)
		go func() { defer wg.Done(); s.IncFiles() }()
		go func() { defer wg.Done(); s.IncExtracted() }()
	}
	wg.Wait()
	if got := s.files.Load(); got != 1000 {
		t.Errorf("files: want 1000, got %d", got)
	}
	if got := s.extracted.Load(); got != 1000 {
		t.Errorf("extracted: want 1000, got %d", got)
	}
}
