package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 重置 info flag 状态防止子测试间互相污染
func resetInfoFlags(t *testing.T) {
	t.Helper()
	flagInfoFormat = "table"
	flagInfoGeoProvider = "offline"
}

// withFakeInfoRunner 替换 infoRunner 为 capture fake，返回 inspector + restore
type infoCallInfo struct {
	args        []string
	format      string
	geoProvider string
	stdout      io.Writer
	stderr      io.Writer
}

func withFakeInfoRunner(t *testing.T) (*infoCallInfo, func()) {
	t.Helper()
	orig := infoRunner
	captured := &infoCallInfo{}
	infoRunner = func(args []string, format, geoProvider string, stdout, stderr io.Writer) error {
		captured.args = args
		captured.format = format
		captured.geoProvider = geoProvider
		captured.stdout = stdout
		captured.stderr = stderr
		return nil
	}
	return captured, func() { infoRunner = orig }
}

func TestInfoRouting_SingleFile(t *testing.T) {
	resetInfoFlags(t)
	cap, restore := withFakeInfoRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"info", "a.jpg"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(cap.args) != 1 || cap.args[0] != "a.jpg" {
		t.Errorf("args: want [a.jpg], got %v", cap.args)
	}
}

func TestInfoRouting_MultiFile(t *testing.T) {
	resetInfoFlags(t)
	cap, restore := withFakeInfoRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"info", "a.jpg", "b.mp3", "c.mp4"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	want := []string{"a.jpg", "b.mp3", "c.mp4"}
	if len(cap.args) != 3 {
		t.Fatalf("want 3 args, got %v", cap.args)
	}
	for i, v := range want {
		if cap.args[i] != v {
			t.Errorf("args[%d]: want %q, got %q", i, v, cap.args[i])
		}
	}
}

func TestInfoRouting_FormatFlag(t *testing.T) {
	resetInfoFlags(t)
	cap, restore := withFakeInfoRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"info", "-f", "json", "a.jpg"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if cap.format != "json" {
		t.Errorf("format: want json, got %q", cap.format)
	}
}

func TestInfoRouting_GeoProviderFlag(t *testing.T) {
	resetInfoFlags(t)
	cap, restore := withFakeInfoRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"info", "-g", "nominatim", "a.jpg"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if cap.geoProvider != "nominatim" {
		t.Errorf("geoProvider: want nominatim, got %q", cap.geoProvider)
	}
}

func TestInfoRouting_NoArgsErrors(t *testing.T) {
	resetInfoFlags(t)
	_, restore := withFakeInfoRunner(t)
	defer restore()

	// 抑制 cobra 把 usage 印到 stderr 污染测试输出
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"info"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("info without args should error (MinimumNArgs(1))")
	}
}

// 真实 runInfo 的集成测试（不走 fake，走 disk）
//
// 这一组测试 runInfo 本身的逻辑：os.Stat / IsDir 检测 / 多文件错误模型
// 不创建真媒体文件——extract.Extract 对非媒体文件返回 TypeUnknown 即可
func TestRunInfo_NonExistentFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runInfo([]string{"/not/exist.jpg"}, "table", "offline", &stdout, &stderr)
	if err == nil {
		t.Error("want error for non-existent file")
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr should contain 'error:', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "not/exist.jpg") {
		t.Errorf("stderr should mention the failing path, got %q", stderr.String())
	}
}

func TestRunInfo_DirectoryArgErrorsWithScanHint(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := runInfo([]string{dir}, "table", "offline", &stdout, &stderr)
	if err == nil {
		t.Error("want error when arg is a directory")
	}
	if !strings.Contains(stderr.String(), "scan") {
		t.Errorf("dir error should hint at 'scan' command, got %q", stderr.String())
	}
}

func TestRunInfo_MultiFile_FailedInMiddle_Continues(t *testing.T) {
	// 用 t.TempDir 建两个真实的（非媒体）文件 + 中间夹一个不存在
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	c := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(a, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runInfo([]string{a, "/not/exist.jpg", c}, "table", "offline", &stdout, &stderr)

	// 应该返回 err（1 file failed）
	if err == nil {
		t.Error("want error when any file fails")
	}
	if !strings.Contains(err.Error(), "1 file") {
		t.Errorf("summary error should mention '1 file', got %v", err)
	}

	// a 和 c 都该出现在 stdout
	out := stdout.String()
	if !strings.Contains(out, a) {
		t.Error("stdout should contain output for first file 'a'")
	}
	if !strings.Contains(out, c) {
		t.Error("stdout should contain output for third file 'c' (continued after middle failure)")
	}

	// 中间的错误该在 stderr
	if !strings.Contains(stderr.String(), "not/exist.jpg") {
		t.Errorf("stderr should mention the failing middle file, got %q", stderr.String())
	}
}

func TestRunInfo_AllSuccess_NoError(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(a, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runInfo([]string{a}, "table", "offline", &stdout, &stderr)
	if err != nil {
		t.Errorf("want nil err for all success, got %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty on all success, got %q", stderr.String())
	}
}

func TestRunInfo_InvalidGeoProviderErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runInfo([]string{"a.txt"}, "table", "not-a-provider", &stdout, &stderr)
	if err == nil {
		t.Error("want error for invalid geo provider")
	}
}
