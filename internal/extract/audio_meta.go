package extract

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// ExtractAudioMeta 使用 ffprobe 提取音频元数据
func ExtractAudioMeta(filePath string) (*media.AudioInfo, error) {
	probe, err := Probe(filePath)
	if err != nil {
		return nil, err
	}
	return BuildAudioInfo(probe), nil
}

// BuildAudioInfo 从 ProbeResult 中提取音频相关字段。
// 抽成独立函数以便单测（不依赖真实 ffprobe 进程）。
//
// 关于"录制时间"：本函数只把 ffprobe 的 format.tags.creation_time/date 解析进
// AudioInfo.RecordedTime，**不**向 MediaRecord.CaptureTime 传播。原因见
// AudioInfo 文档注释——音频"录制年份"和摄影"拍摄时间"是不同语义。
func BuildAudioInfo(probe *ProbeResult) *media.AudioInfo {
	info := &media.AudioInfo{}

	if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.Duration = dur
	}

	// 取第一条 audio stream；多语言/混音文件常见多条音轨
	if s := probe.FirstStreamOfType("audio"); s != nil {
		info.Codec = s.CodecName
		info.Channels = s.Channels
		info.ChannelLayout = s.ChannelLayout
		if rate, err := strconv.Atoi(s.SampleRate); err == nil {
			info.SampleRate = rate
		}
		// 优先用 stream-level bit_rate；纯音频文件常缺，回退到 format.bit_rate
		if br, err := strconv.ParseInt(s.BitRate, 10, 64); err == nil && br > 0 {
			info.Bitrate = br
		}
	}
	if info.Bitrate == 0 {
		if br, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
			info.Bitrate = br
		}
	}

	// 录制时间：优先 creation_time，回退 date / year
	if t, ok := parseAudioRecordedTime(probe.Format.Tags); ok {
		info.RecordedTime = t
		info.HasRecordedTime = true
	}

	return info
}

// parseAudioRecordedTime 从 tags map 中按优先级解析录制时间
// 优先级：creation_time > date > year
// 年份-only（"2018"）解析为该年 1 月 1 日
func parseAudioRecordedTime(tags map[string]string) (time.Time, bool) {
	if tags == nil {
		return time.Time{}, false
	}
	keys := []string{"creation_time", "date", "year"}
	for _, k := range keys {
		v, ok := tags[k]
		if !ok {
			// ID3v2 大写键也常见
			v, ok = tags[strings.ToUpper(k)]
		}
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		if t, err := parseAudioDateTime(v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseAudioDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02 15:04:05",
		"2006:01:02 15:04:05",
		"2006-01-02",
		"2006",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析音频日期: %s", s)
}

// BuildAudioRecord 从音频文件构建 MediaRecord
func BuildAudioRecord(filePath string, fileInfo os.FileInfo) *media.MediaRecord {
	record := &media.MediaRecord{
		FilePath:   filePath,
		FileName:   fileInfo.Name(),
		FileSize:   fileInfo.Size(),
		Type:       media.TypeAudio,
		Attributes: make(map[string]string),
	}

	audioInfo, err := ExtractAudioMeta(filePath)
	if err != nil {
		record.Attributes["audio_error"] = err.Error()
		return record
	}

	record.Audio = audioInfo

	// 注意：故意不向 record.CaptureTime 传播 RecordedTime；
	// 见 media.AudioInfo 文档。

	return record
}
