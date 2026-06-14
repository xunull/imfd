package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetVerifyFlags 重置 verify flag 状态。
func resetVerifyFlags(t *testing.T) {
	t.Helper()
	flagVerifyFormat = "table"
}

// makeTempFile 在 t.TempDir 里建一个文件，返回路径。
func makeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunVerify_FileNotFound(t *testing.T) {
	resetVerifyFlags(t)
	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"/nonexistent/path/zzz.jpg"}, "table", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr should mention error, got: %q", stderr.String())
	}
}

func TestRunVerify_DirectoryRejected(t *testing.T) {
	resetVerifyFlags(t)
	var stdout, stderr bytes.Buffer
	err := runVerify([]string{t.TempDir()}, "table", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for directory path")
	}
	if !strings.Contains(stderr.String(), "list --edited") {
		t.Errorf("stderr should hint at 'imfd list --edited', got: %q", stderr.String())
	}
}

func TestRunVerify_NonImageSkipsButZeroExit(t *testing.T) {
	// 非图像文件：mp4 / mp3 / txt — verify 应 skip 不报错（exit 0）
	resetVerifyFlags(t)
	path := makeTempFile(t, "test.mp4", []byte("fake mp4"))

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{path}, "table", &stdout, &stderr)
	if err != nil {
		t.Fatalf("non-image should NOT error, got: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SKIP") {
		t.Errorf("stdout should show SKIP, got: %q", stdout.String())
	}
}

func TestRunVerify_FakeImageTable(t *testing.T) {
	// 假 jpg：goexif 解析失败，但 extract.Extract 仍返回基础 record（Type=Image）
	// verify 应正常打印（verdict=unknown，因为没 EXIF）
	resetVerifyFlags(t)
	path := makeTempFile(t, "fake.jpg", []byte("not-real-jpg"))

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{path}, "table", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}
	// table 输出包含 FILE 章节
	if !strings.Contains(stdout.String(), "FILE") {
		t.Errorf("stdout should contain FILE section, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "VERDICT") {
		t.Errorf("stdout should contain VERDICT section, got: %q", stdout.String())
	}
	// 假 jpg 没有 EXIF，verdict 应是 unknown
	if !strings.Contains(stdout.String(), "unknown") {
		t.Errorf("stdout verdict should be 'unknown', got: %q", stdout.String())
	}
}

func TestRunVerify_FakeImageJSON(t *testing.T) {
	resetVerifyFlags(t)
	path := makeTempFile(t, "fake.jpg", []byte("not-real-jpg"))

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{path}, "json", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runVerify: %v\nstderr: %s", err, stderr.String())
	}

	// 验证 JSON 可解析
	var report struct {
		FilePath string   `json:"file_path"`
		Verdict  string   `json:"verdict"`
		IsEdited bool     `json:"is_edited"`
		Signals  []string `json:"signals"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON output: %v\nbytes: %s", err, stdout.String())
	}
	if report.FilePath != path {
		t.Errorf("file_path: got %q, want %q", report.FilePath, path)
	}
	if report.Verdict != "unknown" {
		t.Errorf("verdict: got %q, want 'unknown' (fake jpg has no EXIF)", report.Verdict)
	}
	if report.IsEdited {
		t.Error("is_edited should be false for fake jpg with no Software field")
	}
	if len(report.Signals) == 0 {
		t.Error("signals should never be empty")
	}
}

func TestRunVerify_MultipleFilesContinueOnError(t *testing.T) {
	// 第 1 个文件不存在 → error 记账
	// 第 2 个文件存在 → 应继续处理
	// 末尾返回 "1 file(s) failed"
	resetVerifyFlags(t)
	goodPath := makeTempFile(t, "good.jpg", []byte("fake"))

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"/nope/zzz.jpg", goodPath}, "table", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error summary (1 failed)")
	}
	if !strings.Contains(err.Error(), "1 file") {
		t.Errorf("error should mention failure count, got: %v", err)
	}
	// 好的文件仍然被渲染了
	if !strings.Contains(stdout.String(), "good.jpg") {
		t.Errorf("stdout should still render the good file, got: %q", stdout.String())
	}
}

func TestRunVerify_MultipleFilesSeparator(t *testing.T) {
	resetVerifyFlags(t)
	a := makeTempFile(t, "a.jpg", []byte("fake"))
	b := makeTempFile(t, "b.jpg", []byte("fake"))

	var stdout, stderr bytes.Buffer
	if err := runVerify([]string{a, b}, "table", &stdout, &stderr); err != nil {
		t.Fatalf("runVerify: %v", err)
	}
	out := stdout.String()
	// 两个 file path 都出现
	if !strings.Contains(out, "a.jpg") || !strings.Contains(out, "b.jpg") {
		t.Errorf("both files should appear in output, got: %q", out)
	}
	// 至少 2 个 FILE section
	if c := strings.Count(out, "FILE"); c < 2 {
		t.Errorf("expected 2 FILE sections, got %d", c)
	}
}

// TestVerifyCommandRegistered 验证 verify 命令在 root 注册了
func TestVerifyCommandRegistered(t *testing.T) {
	resetVerifyFlags(t)
	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetArgs([]string{"verify", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("verify --help should not error: %v", err)
	}
}
