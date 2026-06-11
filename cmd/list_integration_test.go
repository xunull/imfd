//go:build integration

// Integration tests for imfd list against real fixtures + real walker/extract.
//
// 跑法:
//   go test -tags=integration ./cmd/
//
// 依赖 internal/extract/testdata/gen.sh 生成的样本（mp3/flac/wav/mp4）。
// 默认 go test 不跑这里。

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureDir = "../internal/extract/testdata"

func skipIfNoFixtures(t *testing.T) {
	t.Helper()
	for _, f := range []string{"audio.mp3", "audio.flac", "audio.wav", "video.mp4"} {
		if _, err := os.Stat(filepath.Join(fixtureDir, f)); err != nil {
			t.Skipf("fixture %s missing (run testdata/gen.sh): %v", f, err)
		}
	}
}

func setupFixtureDir(t *testing.T) string {
	t.Helper()
	skipIfNoFixtures(t)
	dir := t.TempDir()
	for _, f := range []string{"audio.mp3", "audio.flac", "audio.wav", "video.mp4"} {
		src := filepath.Join(fixtureDir, f)
		dst := filepath.Join(dir, f)
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestIntegration_ListAudioType(t *testing.T) {
	resetListFlags(t)
	dir := setupFixtureDir(t)
	flagListType = "audio"

	var stdout, stderr bytes.Buffer
	if err := runList([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("runList: %v\nstderr: %s", err, stderr.String())
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("want 3 audio files, got %d: %v", len(lines), lines)
	}
}

func TestIntegration_ListFilterByCodec(t *testing.T) {
	resetListFlags(t)
	dir := setupFixtureDir(t)
	flagListType = "audio"
	flagListFilter = `audio_codec == "mp3"`

	var stdout, stderr bytes.Buffer
	if err := runList([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("runList: %v\nstderr: %s", err, stderr.String())
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 1 || !strings.HasSuffix(lines[0], "audio.mp3") {
		t.Errorf("want 1 mp3, got %v", lines)
	}
}

func TestIntegration_NullSeparator(t *testing.T) {
	resetListFlags(t)
	dir := setupFixtureDir(t)
	flagListType = "audio"
	flagListPrint0 = true

	var stdout, stderr bytes.Buffer
	if err := runList([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("runList: %v", err)
	}
	out := stdout.String()
	if strings.Contains(out, "\n") {
		t.Errorf("NUL mode output should not contain \\n: %q", out)
	}
	if !strings.Contains(out, "\x00") {
		t.Error("NUL mode output should contain \\0")
	}
}
