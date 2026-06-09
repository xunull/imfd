package extract

import (
	"strings"
	"testing"
)

func TestParseProbeResult_HappyPath(t *testing.T) {
	raw := []byte(`{
		"format": {
			"duration": "12.345",
			"bit_rate": "192000",
			"tags": {"creation_time": "2023-04-15T10:20:30.000000Z", "title": "Test"}
		},
		"streams": [
			{"codec_type": "video", "codec_name": "h264", "width": 1920, "height": 1080, "r_frame_rate": "30000/1001"},
			{"codec_type": "audio", "codec_name": "aac", "sample_rate": "48000", "channels": 2, "channel_layout": "stereo", "bit_rate": "128000"}
		]
	}`)

	probe, err := ParseProbeResult(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if probe.Format.Duration != "12.345" {
		t.Errorf("Format.Duration: want 12.345, got %q", probe.Format.Duration)
	}
	if probe.Format.BitRate != "192000" {
		t.Errorf("Format.BitRate: want 192000, got %q", probe.Format.BitRate)
	}
	if probe.Format.Tags["creation_time"] != "2023-04-15T10:20:30.000000Z" {
		t.Errorf("creation_time tag missing or wrong")
	}
	if len(probe.Streams) != 2 {
		t.Fatalf("want 2 streams, got %d", len(probe.Streams))
	}
}

func TestParseProbeResult_EmptyInput(t *testing.T) {
	if _, err := ParseProbeResult([]byte{}); err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestParseProbeResult_InvalidJSON(t *testing.T) {
	_, err := ParseProbeResult([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "解析") {
		t.Errorf("error should mention 解析, got %v", err)
	}
}

func TestParseProbeResult_NoStreams(t *testing.T) {
	raw := []byte(`{"format": {"duration": "1.0"}, "streams": []}`)
	probe, err := ParseProbeResult(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(probe.Streams) != 0 {
		t.Errorf("want 0 streams, got %d", len(probe.Streams))
	}
}

func TestParseProbeResult_MultipleAudioStreams(t *testing.T) {
	// 多语言电影常见：多条音轨
	raw := []byte(`{
		"format": {},
		"streams": [
			{"codec_type": "video", "codec_name": "h264"},
			{"codec_type": "audio", "codec_name": "aac", "tags": {"language": "eng"}},
			{"codec_type": "audio", "codec_name": "ac3", "tags": {"language": "chi"}}
		]
	}`)
	probe, err := ParseProbeResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first := probe.FirstStreamOfType("audio")
	if first == nil {
		t.Fatal("FirstStreamOfType audio returned nil")
	}
	if first.CodecName != "aac" {
		t.Errorf("first audio stream codec: want aac, got %q", first.CodecName)
	}
}

func TestFirstStreamOfType_NotFound(t *testing.T) {
	probe := &ProbeResult{Streams: []ProbeStream{{CodecType: "video", CodecName: "h264"}}}
	if got := probe.FirstStreamOfType("audio"); got != nil {
		t.Errorf("expected nil for missing type, got %+v", got)
	}
}

func TestCheckFFprobe_IdempotentAndRaceSafe(t *testing.T) {
	// sync.Once 保证：多次并发调用都拿到同一个结果
	results := make(chan bool, 50)
	for range 50 {
		go func() { results <- checkFFprobe() }()
	}
	first := <-results
	for range 49 {
		if got := <-results; got != first {
			t.Errorf("checkFFprobe inconsistent under concurrency: want %v, got %v", first, got)
		}
	}
}
