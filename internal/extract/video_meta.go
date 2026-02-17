package extract

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// ffprobeOutput ffprobe JSON 输出结构
type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Duration string            `json:"duration"`
	BitRate  string            `json:"bit_rate"`
	Tags     map[string]string `json:"tags"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	RFrameRate string `json:"r_frame_rate"`
	Tags      map[string]string `json:"tags"`
}

// ffprobeAvailable 检查 ffprobe 是否可用
var ffprobeAvailable *bool

func checkFFprobe() bool {
	if ffprobeAvailable != nil {
		return *ffprobeAvailable
	}
	_, err := exec.LookPath("ffprobe")
	result := err == nil
	ffprobeAvailable = &result
	return result
}

// ExtractVideoMeta 使用 ffprobe 提取视频元数据
func ExtractVideoMeta(filePath string) (*media.VideoInfo, error) {
	if !checkFFprobe() {
		return nil, fmt.Errorf("ffprobe 未安装，无法提取视频元数据")
	}

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}

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

	return info, nil
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
