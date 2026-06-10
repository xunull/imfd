package extract

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// ExtractVideoMeta 使用 ffprobe 提取视频元数据
func ExtractVideoMeta(filePath string) (*media.VideoInfo, error) {
	probe, err := Probe(filePath)
	if err != nil {
		return nil, err
	}
	return BuildVideoInfo(probe), nil
}

// BuildVideoInfo 从 ProbeResult 中提取视频相关字段。
// 抽成独立函数以便单测（不依赖真实 ffprobe 进程）。
func BuildVideoInfo(probe *ProbeResult) *media.VideoInfo {
	info := &media.VideoInfo{}

	if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.Duration = dur
	}
	if br, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
		info.Bitrate = br
	}

	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			info.Codec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height
			info.FrameRate = stream.RFrameRate
		case "audio":
			info.AudioCodec = stream.CodecName
		}
	}

	if ct, ok := probe.Format.Tags["creation_time"]; ok {
		if t, err := parseVideoDateTime(ct); err == nil {
			info.CreateTime = t
			info.HasDateTime = true
		}
	}

	return info
}

func parseVideoDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02 15:04:05",
		"2006:01:02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析视频日期: %s", s)
}

// BuildVideoRecord 从视频文件构建 MediaRecord
func BuildVideoRecord(filePath string, fileInfo os.FileInfo) *media.MediaRecord {
	record := &media.MediaRecord{
		FilePath:   filePath,
		FileName:   fileInfo.Name(),
		FileSize:   fileInfo.Size(),
		ModTime:    fileInfo.ModTime(),
		Type:       media.TypeVideo,
		Attributes: make(map[string]string),
	}

	videoInfo, err := ExtractVideoMeta(filePath)
	if err != nil {
		record.Attributes["video_error"] = err.Error()
		return record
	}

	record.Video = videoInfo

	if videoInfo.HasDateTime {
		record.CaptureTime = videoInfo.CreateTime
		record.HasCaptureTime = true
	}

	return record
}
