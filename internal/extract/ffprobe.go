package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// ProbeResult ffprobe -show_format -show_streams 的 JSON 输出
type ProbeResult struct {
	Format  ProbeFormat   `json:"format"`
	Streams []ProbeStream `json:"streams"`
}

type ProbeFormat struct {
	Duration string            `json:"duration"`
	BitRate  string            `json:"bit_rate"`
	Tags     map[string]string `json:"tags"`
}

type ProbeStream struct {
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	RFrameRate    string            `json:"r_frame_rate"`
	SampleRate    string            `json:"sample_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	BitRate       string            `json:"bit_rate"`
	Tags          map[string]string `json:"tags"`
}

var (
	ffprobeOnce      sync.Once
	ffprobeAvailable bool
)

// ProbeTimeout 是每个文件 ffprobe 调用的超时时间。
// 超时后该文件视为提取失败（记录错误继续处理下一个），不会卡死整个流水线。
// 可通过 IMFD_PROBE_TIMEOUT 环境变量覆盖（单位：秒），测试也可直接赋值。
var ProbeTimeout = 30 * time.Second

// checkFFprobe 检查 ffprobe 是否安装；并发安全，单进程内只检测一次
func checkFFprobe() bool {
	ffprobeOnce.Do(func() {
		_, err := exec.LookPath("ffprobe")
		ffprobeAvailable = err == nil
	})
	return ffprobeAvailable
}

// Probe 对单个媒体文件运行 ffprobe，返回结构化的 ProbeResult。
// 超过 ProbeTimeout 时 kill 子进程并返回超时错误，不会永久阻塞。
func Probe(filePath string) (*ProbeResult, error) {
	if !checkFFprobe() {
		return nil, fmt.Errorf("ffprobe 未安装，无法提取媒体元数据")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("ffprobe 超时 (%.0fs)，跳过此文件: %s", ProbeTimeout.Seconds(), filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	return ParseProbeResult(output)
}

// ParseProbeResult 把 ffprobe 的 JSON 字节流解析为 ProbeResult；
// 提取出来便于单测——不需要真实 ffprobe 进程即可验证字段映射
func ParseProbeResult(raw []byte) (*ProbeResult, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("ffprobe 输出为空")
	}
	var probe ProbeResult
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	return &probe, nil
}

// FirstStreamOfType 返回第一个匹配类型的流；找不到返回 nil
func (p *ProbeResult) FirstStreamOfType(codecType string) *ProbeStream {
	for i := range p.Streams {
		if p.Streams[i].CodecType == codecType {
			return &p.Streams[i]
		}
	}
	return nil
}
