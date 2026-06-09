package dimensions

import (
	"fmt"

	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// NewAudioCodecDimension 按音频文件的编解码器分组统计。
// 注意：只统计音频文件本身的 codec，不合并视频里的音轨 codec。
func NewAudioCodecDimension() stats.DimensionCounter {
	return stats.NewFieldDimension("音频编解码器", "按音频编解码器分组统计（仅音频文件，不含视频音轨）",
		func(r *media.MediaRecord) string {
			if r.Audio == nil {
				return ""
			}
			return r.Audio.Codec
		},
		media.TypeAudio)
}

// NewAudioBitrateDimension 按音频比特率分桶统计。
func NewAudioBitrateDimension() stats.DimensionCounter {
	return stats.NewFieldDimension("音频比特率", "按音频比特率分桶统计（96k 以下/128k/192k/256k/320k/lossless）",
		func(r *media.MediaRecord) string {
			if r.Audio == nil || r.Audio.Bitrate == 0 {
				return ""
			}
			return bitrateBucket(r.Audio.Bitrate)
		},
		media.TypeAudio)
}

// NewAudioSampleRateDimension 按采样率分组统计。
func NewAudioSampleRateDimension() stats.DimensionCounter {
	return stats.NewFieldDimension("音频采样率", "按音频采样率分组统计",
		func(r *media.MediaRecord) string {
			if r.Audio == nil || r.Audio.SampleRate == 0 {
				return ""
			}
			return sampleRateLabel(r.Audio.SampleRate)
		},
		media.TypeAudio)
}

// NewAudioChannelsDimension 按声道布局分组统计。
// 优先用 ChannelLayout（mono/stereo/5.1）；缺失时回落到通道数。
func NewAudioChannelsDimension() stats.DimensionCounter {
	return stats.NewFieldDimension("音频声道", "按音频声道布局分组统计",
		func(r *media.MediaRecord) string {
			if r.Audio == nil {
				return ""
			}
			if r.Audio.ChannelLayout != "" {
				return r.Audio.ChannelLayout
			}
			if r.Audio.Channels > 0 {
				return fmt.Sprintf("%dch", r.Audio.Channels)
			}
			return ""
		},
		media.TypeAudio)
}

// NewAudioDurationBucketDimension 按音频时长分桶统计。
func NewAudioDurationBucketDimension() stats.DimensionCounter {
	return stats.NewFieldDimension("音频时长", "按音频时长分桶统计（<1m/1-5m/5-30m/30m-2h/>2h）",
		func(r *media.MediaRecord) string {
			if r.Audio == nil || r.Audio.Duration <= 0 {
				return ""
			}
			return durationBucket(r.Audio.Duration)
		},
		media.TypeAudio)
}

// bitrateBucket 把 bps 比特率归类到人类友好的桶
func bitrateBucket(bps int64) string {
	switch {
	case bps < 96_000:
		return "<96kbps"
	case bps < 128_000:
		return "96kbps"
	case bps < 192_000:
		return "128kbps"
	case bps < 256_000:
		return "192kbps"
	case bps < 320_000:
		return "256kbps"
	case bps < 500_000:
		return "320kbps"
	default:
		return "lossless / 500kbps+"
	}
}

// sampleRateLabel 用 kHz 表示采样率
func sampleRateLabel(hz int) string {
	if hz%1000 == 0 {
		return fmt.Sprintf("%dkHz", hz/1000)
	}
	return fmt.Sprintf("%.1fkHz", float64(hz)/1000.0)
}

// durationBucket 把秒数归类为「<1m / 1-5m / 5-30m / 30m-2h / >2h」
func durationBucket(sec float64) string {
	switch {
	case sec < 60:
		return "<1分钟"
	case sec < 5*60:
		return "1-5分钟"
	case sec < 30*60:
		return "5-30分钟"
	case sec < 2*60*60:
		return "30分钟-2小时"
	default:
		return ">2小时"
	}
}
