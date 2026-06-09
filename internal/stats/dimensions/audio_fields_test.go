package dimensions

import (
	"testing"

	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// bucketCounts 把 dimension.Result().Buckets 折回 map 便于断言
func bucketCounts(buckets []stats.Bucket) map[string]int {
	m := make(map[string]int, len(buckets))
	for _, b := range buckets {
		m[b.Key] = b.Count
	}
	return m
}

func TestAudioCodecDimension(t *testing.T) {
	dim := NewAudioCodecDimension()
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: "mp3"}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: "mp3"}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: "flac"}})
	dim.Consume(&media.MediaRecord{Type: media.TypeImage}) // Audio nil → Unknown
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: ""}})

	got := bucketCounts(dim.Result().Buckets)
	want := map[string]int{"mp3": 2, "flac": 1, "Unknown": 2}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("bucket %q: want %d, got %d (full=%v)", k, v, got[k], got)
		}
	}
}

func TestAudioBitrateDimension_BucketBoundaries(t *testing.T) {
	dim := NewAudioBitrateDimension()
	cases := []struct {
		bps  int64
		want string
	}{
		{64_000, "<96kbps"},
		{96_000, "96kbps"},
		{128_000, "128kbps"},
		{192_000, "192kbps"},
		{256_000, "256kbps"},
		{320_000, "320kbps"},
		{900_000, "lossless / 500kbps+"},
	}
	for _, c := range cases {
		dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Bitrate: c.bps}})
	}
	// 零比特率 + image → Unknown ×2
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Bitrate: 0}})
	dim.Consume(&media.MediaRecord{Type: media.TypeImage})

	got := bucketCounts(dim.Result().Buckets)
	for _, c := range cases {
		if got[c.want] != 1 {
			t.Errorf("bitrate %d: want bucket %q count=1, got=%d (full=%v)", c.bps, c.want, got[c.want], got)
		}
	}
	if got["Unknown"] != 2 {
		t.Errorf("Unknown: want 2, got %d", got["Unknown"])
	}
}

func TestAudioSampleRateDimension(t *testing.T) {
	dim := NewAudioSampleRateDimension()
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{SampleRate: 44100}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{SampleRate: 48000}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{SampleRate: 96000}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{SampleRate: 0}}) // Unknown

	got := bucketCounts(dim.Result().Buckets)
	checks := map[string]int{"44.1kHz": 1, "48kHz": 1, "96kHz": 1, "Unknown": 1}
	for k, v := range checks {
		if got[k] != v {
			t.Errorf("%s: want %d, got %d (full=%v)", k, v, got[k], got)
		}
	}
}

func TestAudioChannelsDimension_LayoutPriority(t *testing.T) {
	dim := NewAudioChannelsDimension()
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{ChannelLayout: "stereo", Channels: 2}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{ChannelLayout: "5.1", Channels: 6}})
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Channels: 1}}) // 无 layout，落回通道数
	dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{}})            // 全空 → Unknown

	got := bucketCounts(dim.Result().Buckets)
	checks := map[string]int{"stereo": 1, "5.1": 1, "1ch": 1, "Unknown": 1}
	for k, v := range checks {
		if got[k] != v {
			t.Errorf("%s: want %d, got %d (full=%v)", k, v, got[k], got)
		}
	}
}

func TestAudioDurationBucketDimension(t *testing.T) {
	dim := NewAudioDurationBucketDimension()
	cases := []struct {
		sec  float64
		want string
	}{
		{30, "<1分钟"},
		{120, "1-5分钟"},
		{600, "5-30分钟"},
		{3600, "30分钟-2小时"},
		{10000, ">2小时"},
		{0, "Unknown"},
	}
	for _, c := range cases {
		dim.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Duration: c.sec}})
	}
	got := bucketCounts(dim.Result().Buckets)
	for _, c := range cases {
		if got[c.want] < 1 {
			t.Errorf("duration %v: want bucket %q count>=1, got=%d (full=%v)", c.sec, c.want, got[c.want], got)
		}
	}
}

func TestAudioDimensions_NilAudioNeverPanics(t *testing.T) {
	// 所有 audio dim 对非 audio record（Audio==nil）应当返回 Unknown 桶且不 panic
	imgRec := &media.MediaRecord{Type: media.TypeImage, Exif: &media.ExifInfo{}}
	dims := []stats.DimensionCounter{
		NewAudioCodecDimension(),
		NewAudioBitrateDimension(),
		NewAudioSampleRateDimension(),
		NewAudioChannelsDimension(),
		NewAudioDurationBucketDimension(),
	}
	for _, dim := range dims {
		dim.Consume(imgRec) // 不 panic
		got := bucketCounts(dim.Result().Buckets)
		if got["Unknown"] != 1 {
			t.Errorf("dim %s: want Unknown bucket count=1 for image record, got=%v", dim.Name(), got)
		}
	}
}
