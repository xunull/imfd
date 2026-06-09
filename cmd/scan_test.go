package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/media"
)

// withFakeRunner 替换 scanRunner 为捕获 cfg 的 fake；返回 restore 函数和读取最近 cfg 的 getter
func withFakeRunner(t *testing.T) (func() *config.Config, func()) {
	t.Helper()
	orig := scanRunner
	var captured *config.Config
	scanRunner = func(cfg *config.Config) error {
		captured = cfg
		return nil
	}
	return func() *config.Config { return captured }, func() { scanRunner = orig }
}

// scopedTempDir 建立一个临时目录给 -d / 位置参数使用
// 真正的 scan 不会跑（fake runner 截了），目录只要存在
func scopedTempDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	// 加一个普通文件防止空目录边界
	if err := os.WriteFile(filepath.Join(d, "marker.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

// resetFlags 重置全局 flag 状态，避免子测试间互相污染
func resetFlags(t *testing.T) {
	t.Helper()
	flagDir = "."
	flagWorkers = 8
	flagExtractors = 0
	flagOutputFormat = "table"
	flagChannelSize = 1024
	flagGeoProvider = "offline"
	flagVerbose = false
	flagLegacyTable = false
}

func TestScanRouting_BareScan_NilMediaTypes(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	if cfg.MediaTypes != nil {
		t.Errorf("bare scan: want MediaTypes=nil, got %v", cfg.MediaTypes)
	}
	if cfg.Dir != dir {
		t.Errorf("Dir: want %q, got %q", dir, cfg.Dir)
	}
}

func TestScanRouting_ScanAll_NilMediaTypes(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "all", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	if cfg.MediaTypes != nil {
		t.Errorf("scan all: want MediaTypes=nil, got %v", cfg.MediaTypes)
	}
}

func TestScanRouting_ScanAudio_AudioOnly(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "audio", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	want := []media.MediaType{media.TypeAudio}
	if !slices.Equal(cfg.MediaTypes, want) {
		t.Errorf("scan audio: want %v, got %v", want, cfg.MediaTypes)
	}
}

func TestScanRouting_ScanImage_ImageOnly(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "image", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	want := []media.MediaType{media.TypeImage}
	if !slices.Equal(cfg.MediaTypes, want) {
		t.Errorf("scan image: want %v, got %v", want, cfg.MediaTypes)
	}
}

func TestScanRouting_ScanVideo_VideoOnly(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "video", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	want := []media.MediaType{media.TypeVideo}
	if !slices.Equal(cfg.MediaTypes, want) {
		t.Errorf("scan video: want %v, got %v", want, cfg.MediaTypes)
	}
}

func TestScanRouting_PersistentFlagInheritance(t *testing.T) {
	// -w 在父命令上是 PersistentFlag，scan audio 子命令应该能收到
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "-w", "16", "audio", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	if cfg.Workers != 16 {
		t.Errorf("Workers: want 16 (PersistentFlag 继承), got %d", cfg.Workers)
	}
	if !slices.Equal(cfg.MediaTypes, []media.MediaType{media.TypeAudio}) {
		t.Errorf("MediaTypes: want [TypeAudio], got %v", cfg.MediaTypes)
	}
}

func TestScanRouting_VerboseFlagSetsConfig(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "-v", "audio", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	if !cfg.Verbose {
		t.Error("-v should set cfg.Verbose=true")
	}
	if cfg.LegacyTable {
		t.Error("absent --legacy-table should leave cfg.LegacyTable=false")
	}
}

func TestScanRouting_LegacyTableFlagSetsConfig(t *testing.T) {
	resetFlags(t)
	getCfg, restore := withFakeRunner(t)
	defer restore()

	dir := scopedTempDir(t)
	rootCmd.SetArgs([]string{"scan", "--legacy-table", "image", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	cfg := getCfg()
	if cfg == nil {
		t.Fatal("scanRunner was not called")
	}
	if !cfg.LegacyTable {
		t.Error("--legacy-table should set cfg.LegacyTable=true")
	}
}

func TestScanRouting_InvalidDirReturnsError(t *testing.T) {
	resetFlags(t)
	_, restore := withFakeRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"scan", "audio", "/nonexistent-imfd-test-path-zzz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("want error for missing dir, got nil")
	}
}
