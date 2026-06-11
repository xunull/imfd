package pipeline

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/media"
)

// TestRunWithHandler_CustomHandlerCollectsRecords 是 RecordHandler 接缝的 smoke test。
// CRITICAL：plan-eng-review A1=A 要求 RecordHandler 通用化重构必须保证此路径工作。
func TestRunWithHandler_CustomHandlerCollectsRecords(t *testing.T) {
	dir := t.TempDir()
	// 写个最小媒体文件名（即使内容不解析也会触发 walker → extract → handler）
	if err := os.WriteFile(filepath.Join(dir, "fake.jpg"), []byte("not-real-jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var collected []*media.MediaRecord
	handler := HandlerFunc(func(r *media.MediaRecord) error {
		mu.Lock()
		collected = append(collected, r)
		mu.Unlock()
		return nil
	})

	cfg := &config.Config{
		Dir:         dir,
		Workers:     2,
		Extractors:  2,
		ChannelSize: 16,
		GeoProvider: "offline",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	if err := RunWithHandler(cfg, handler); err != nil {
		t.Fatalf("RunWithHandler error: %v", err)
	}

	if len(collected) != 1 {
		t.Errorf("expected handler called 1 time for fake.jpg, got %d", len(collected))
	}
	if len(collected) > 0 && collected[0].FileName != "fake.jpg" {
		t.Errorf("expected fake.jpg, got %s", collected[0].FileName)
	}
}

// TestRun_BackwardCompatible 是 scan 路径的 smoke test：传 nil handler 保留旧 aggregate 行为。
func TestRun_BackwardCompatible(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Dir:          dir,
		Workers:      2,
		Extractors:   2,
		ChannelSize:  16,
		GeoProvider:  "offline",
		OutputFormat: config.FormatJSON, // 走 JSON 路径避免 dashboard stdout 写入测试日志
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	// 空目录走完整 scan 路径，应不报错
	if err := Run(cfg); err != nil {
		t.Fatalf("Run on empty dir should not error, got %v", err)
	}
}
