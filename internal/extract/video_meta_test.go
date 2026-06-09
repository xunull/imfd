package extract

import (
	"testing"
	"time"
)

func TestBuildVideoInfo_HappyPath(t *testing.T) {
	probe := &ProbeResult{
		Format: ProbeFormat{
			Duration: "120.5",
			BitRate:  "5000000",
			Tags:     map[string]string{"creation_time": "2023-04-15T10:20:30.000000Z"},
		},
		Streams: []ProbeStream{
			{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080, RFrameRate: "30000/1001"},
			{CodecType: "audio", CodecName: "aac"},
		},
	}

	info := BuildVideoInfo(probe)

	if info.Duration != 120.5 {
		t.Errorf("Duration: want 120.5, got %v", info.Duration)
	}
	if info.Bitrate != 5000000 {
		t.Errorf("Bitrate: want 5000000, got %d", info.Bitrate)
	}
	if info.Codec != "h264" {
		t.Errorf("Codec: want h264, got %q", info.Codec)
	}
	if info.Width != 1920 || info.Height != 1080 {
		t.Errorf("dimensions: want 1920x1080, got %dx%d", info.Width, info.Height)
	}
	if info.FrameRate != "30000/1001" {
		t.Errorf("FrameRate: want 30000/1001, got %q", info.FrameRate)
	}
	if info.AudioCodec != "aac" {
		t.Errorf("AudioCodec: want aac, got %q", info.AudioCodec)
	}
	if !info.HasDateTime {
		t.Error("HasDateTime: want true, got false")
	}
	want := time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)
	if !info.CreateTime.Equal(want) {
		t.Errorf("CreateTime: want %v, got %v", want, info.CreateTime)
	}
}

func TestBuildVideoInfo_NoCreationTime(t *testing.T) {
	probe := &ProbeResult{
		Format: ProbeFormat{Duration: "10.0", BitRate: "1000000"},
		Streams: []ProbeStream{
			{CodecType: "video", CodecName: "vp9"},
		},
	}
	info := BuildVideoInfo(probe)
	if info.HasDateTime {
		t.Error("HasDateTime: want false when no creation_time tag, got true")
	}
}

func TestBuildVideoInfo_InvalidDuration(t *testing.T) {
	probe := &ProbeResult{
		Format: ProbeFormat{Duration: "N/A", BitRate: "garbage"},
		Streams: []ProbeStream{{CodecType: "video", CodecName: "h264"}},
	}
	info := BuildVideoInfo(probe)
	if info.Duration != 0 {
		t.Errorf("Duration: want 0 on parse fail, got %v", info.Duration)
	}
	if info.Bitrate != 0 {
		t.Errorf("Bitrate: want 0 on parse fail, got %d", info.Bitrate)
	}
}

func TestBuildVideoInfo_OnlyAudioStream(t *testing.T) {
	// 极端边界：传入一个只有 audio stream 的 probe（这其实是个音频文件，但 BuildVideoInfo 不应该 panic）
	probe := &ProbeResult{
		Format:  ProbeFormat{Duration: "30.0"},
		Streams: []ProbeStream{{CodecType: "audio", CodecName: "mp3"}},
	}
	info := BuildVideoInfo(probe)
	if info.Codec != "" {
		t.Errorf("Codec: want empty when no video stream, got %q", info.Codec)
	}
	if info.AudioCodec != "mp3" {
		t.Errorf("AudioCodec: want mp3, got %q", info.AudioCodec)
	}
}

func TestParseVideoDateTime_MultipleLayouts(t *testing.T) {
	cases := []struct {
		input string
		want  time.Time
	}{
		{"2023-04-15T10:20:30Z", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023-04-15T10:20:30.000000Z", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023-04-15 10:20:30", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023:04:15 10:20:30", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := parseVideoDateTime(c.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(c.want) {
				t.Errorf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestParseVideoDateTime_Invalid(t *testing.T) {
	if _, err := parseVideoDateTime("not a date"); err == nil {
		t.Error("expected error for invalid input")
	}
}
