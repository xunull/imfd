//go:build integration

// Integration tests against a real ffprobe binary + sample fixtures.
//
// 跑法：
//   go test -tags=integration ./internal/extract/
//
// 需要 ffprobe + testdata/ 下的样本文件。生成命令见 testdata/README.md。
// 默认 go test ./... 不跑这里。

package extract

import (
	"os"
	"path/filepath"
	"testing"
)

const fixtureDir = "testdata"

func skipIfMissing(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(fixtureDir, name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("fixture %s not present (run testdata/gen.sh): %v", p, err)
	}
	return p
}

func TestIntegration_ProbeAudioMP3(t *testing.T) {
	path := skipIfMissing(t, "audio.mp3")
	probe, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	info := BuildAudioInfo(probe)
	if info.Codec != "mp3" {
		t.Errorf("Codec: want mp3, got %q", info.Codec)
	}
	if info.SampleRate == 0 {
		t.Error("SampleRate should be > 0 for real MP3")
	}
	if info.Duration <= 0 {
		t.Error("Duration should be > 0 for real MP3")
	}
}

func TestIntegration_ProbeAudioFLAC(t *testing.T) {
	path := skipIfMissing(t, "audio.flac")
	probe, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	info := BuildAudioInfo(probe)
	if info.Codec != "flac" {
		t.Errorf("Codec: want flac, got %q", info.Codec)
	}
}

func TestIntegration_ProbeAudioWAV(t *testing.T) {
	path := skipIfMissing(t, "audio.wav")
	probe, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	info := BuildAudioInfo(probe)
	if info.Codec == "" {
		t.Error("Codec should not be empty for real WAV")
	}
}

func TestIntegration_ProbeVideoMP4(t *testing.T) {
	path := skipIfMissing(t, "video.mp4")
	probe, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe error: %v", err)
	}
	info := BuildVideoInfo(probe)
	if info.Codec == "" {
		t.Error("Codec should not be empty for real MP4")
	}
	if info.Width == 0 || info.Height == 0 {
		t.Error("dimensions should be > 0 for real MP4")
	}
}

func TestIntegration_BuildRecord_Audio(t *testing.T) {
	path := skipIfMissing(t, "audio.mp3")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := BuildAudioRecord(path, fi)
	if rec.Type.String() != "audio" {
		t.Errorf("Type: want audio, got %s", rec.Type)
	}
	if rec.Audio == nil {
		t.Fatal("Audio info should be populated")
	}
	if rec.HasCaptureTime {
		t.Error("HasCaptureTime should be false for audio (recorded time is audio-only)")
	}
}
