package extract

import (
	"testing"
	"time"
)

func TestBuildAudioInfo_HappyPath_MP3(t *testing.T) {
	probe := &ProbeResult{
		Format: ProbeFormat{
			Duration: "245.6",
			BitRate:  "192000",
			Tags:     map[string]string{"creation_time": "2023-04-15T10:20:30.000000Z"},
		},
		Streams: []ProbeStream{
			{CodecType: "audio", CodecName: "mp3", SampleRate: "44100", Channels: 2, ChannelLayout: "stereo"},
		},
	}

	info := BuildAudioInfo(probe)

	if info.Codec != "mp3" {
		t.Errorf("Codec: want mp3, got %q", info.Codec)
	}
	if info.SampleRate != 44100 {
		t.Errorf("SampleRate: want 44100, got %d", info.SampleRate)
	}
	if info.Channels != 2 {
		t.Errorf("Channels: want 2, got %d", info.Channels)
	}
	if info.ChannelLayout != "stereo" {
		t.Errorf("ChannelLayout: want stereo, got %q", info.ChannelLayout)
	}
	if info.Duration != 245.6 {
		t.Errorf("Duration: want 245.6, got %v", info.Duration)
	}
	if info.Bitrate != 192000 {
		t.Errorf("Bitrate: want 192000 (format-level fallback), got %d", info.Bitrate)
	}
	if !info.HasRecordedTime {
		t.Error("HasRecordedTime: want true")
	}
}

func TestBuildAudioInfo_StreamBitratePreferredOverFormat(t *testing.T) {
	// stream-level bit_rate 优先于 format-level
	probe := &ProbeResult{
		Format: ProbeFormat{Duration: "10", BitRate: "320000"},
		Streams: []ProbeStream{
			{CodecType: "audio", CodecName: "flac", BitRate: "850000"},
		},
	}
	info := BuildAudioInfo(probe)
	if info.Bitrate != 850000 {
		t.Errorf("Bitrate: want 850000 (stream-level), got %d", info.Bitrate)
	}
}

func TestBuildAudioInfo_NoAudioStream(t *testing.T) {
	// 文件扩名 .mp3 但内容损坏，ffprobe 解析不出 audio stream
	probe := &ProbeResult{
		Format:  ProbeFormat{Duration: "0"},
		Streams: []ProbeStream{},
	}
	info := BuildAudioInfo(probe)
	if info.Codec != "" {
		t.Errorf("Codec: want empty when no audio stream, got %q", info.Codec)
	}
	if info.SampleRate != 0 {
		t.Errorf("SampleRate: want 0, got %d", info.SampleRate)
	}
}

func TestBuildAudioInfo_MultipleAudioStreams_TakesFirst(t *testing.T) {
	// 多音轨：取第一条
	probe := &ProbeResult{
		Format: ProbeFormat{Duration: "100"},
		Streams: []ProbeStream{
			{CodecType: "audio", CodecName: "aac", SampleRate: "48000", Channels: 2, Tags: map[string]string{"language": "eng"}},
			{CodecType: "audio", CodecName: "ac3", SampleRate: "48000", Channels: 6, Tags: map[string]string{"language": "chi"}},
		},
	}
	info := BuildAudioInfo(probe)
	if info.Codec != "aac" {
		t.Errorf("Codec: want aac (first stream), got %q", info.Codec)
	}
	if info.Channels != 2 {
		t.Errorf("Channels: want 2, got %d", info.Channels)
	}
}

func TestBuildAudioInfo_NoTags(t *testing.T) {
	probe := &ProbeResult{
		Format:  ProbeFormat{Duration: "30"},
		Streams: []ProbeStream{{CodecType: "audio", CodecName: "wav", SampleRate: "44100"}},
	}
	info := BuildAudioInfo(probe)
	if info.HasRecordedTime {
		t.Error("HasRecordedTime: want false when no tags, got true")
	}
}

func TestParseAudioRecordedTime_PriorityOrder(t *testing.T) {
	// creation_time 优先于 date 优先于 year
	tags := map[string]string{
		"creation_time": "2023-04-15T10:20:30Z",
		"date":          "2018-01-01",
		"year":          "2000",
	}
	got, ok := parseAudioRecordedTime(tags)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("priority: want %v (creation_time), got %v", want, got)
	}
}

func TestParseAudioRecordedTime_FallbackToDate(t *testing.T) {
	tags := map[string]string{"date": "2018-05-12"}
	got, ok := parseAudioRecordedTime(tags)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := time.Date(2018, 5, 12, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestParseAudioRecordedTime_YearOnly(t *testing.T) {
	tags := map[string]string{"date": "2018"}
	got, ok := parseAudioRecordedTime(tags)
	if !ok {
		t.Fatal("expected ok=true for year-only")
	}
	want := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestParseAudioRecordedTime_UpperCaseKeys(t *testing.T) {
	// ID3v2 frame 大写键
	tags := map[string]string{"DATE": "2020"}
	_, ok := parseAudioRecordedTime(tags)
	if !ok {
		t.Error("expected to find DATE (uppercase) tag")
	}
}

func TestParseAudioRecordedTime_NilMap(t *testing.T) {
	if _, ok := parseAudioRecordedTime(nil); ok {
		t.Error("expected ok=false for nil tags map")
	}
}

func TestParseAudioRecordedTime_EmptyAndWhitespace(t *testing.T) {
	tags := map[string]string{"creation_time": "", "date": "   ", "year": ""}
	if _, ok := parseAudioRecordedTime(tags); ok {
		t.Error("expected ok=false when all tags empty/whitespace")
	}
}

func TestParseAudioRecordedTime_InvalidFormat(t *testing.T) {
	tags := map[string]string{"date": "not-a-date"}
	if _, ok := parseAudioRecordedTime(tags); ok {
		t.Error("expected ok=false for unparseable date")
	}
}

func TestParseAudioDateTime_AllLayouts(t *testing.T) {
	cases := []struct {
		input string
		want  time.Time
	}{
		{"2023-04-15T10:20:30Z", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023-04-15T10:20:30.000000Z", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023-04-15 10:20:30", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023:04:15 10:20:30", time.Date(2023, 4, 15, 10, 20, 30, 0, time.UTC)},
		{"2023-04-15", time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC)},
		{"2023", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := parseAudioDateTime(c.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(c.want) {
				t.Errorf("want %v, got %v", c.want, got)
			}
		})
	}
}
