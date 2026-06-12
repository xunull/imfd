package config

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/xunull/imfd/internal/media"
)

// OutputFormat 输出格式类型
type OutputFormat int

const (
	FormatTable OutputFormat = iota
	FormatJSON
	FormatBoth
)

// ParseOutputFormat 解析输出格式字符串
func ParseOutputFormat(s string) OutputFormat {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FormatJSON
	case "both":
		return FormatBoth
	default:
		return FormatTable
	}
}

// Config 全局配置
type Config struct {
	Dir          string
	Workers      int
	Extractors   int
	OutputFormat OutputFormat
	ChannelSize  int
	GeoProvider  string // 地理反查提供者: offline, nominatim

	// MediaTypes 限定本次扫描只处理哪些媒体类型；nil 表示全部
	// （走 `imfd scan` 或 `imfd scan all` 时为 nil；走 `scan audio/image/video` 时填对应类型）
	MediaTypes []media.MediaType

	// Verbose 控制 dashboard 是否展开"全 Unknown"维度
	// 默认 false：折叠为一行提示
	Verbose bool

	// LegacyTable 控制是否走老的 go-pretty 表格输出（迁移逃生口）
	// 默认 false：走新 dashboard
	LegacyTable bool

	// NoCache 为 true 时跳过 cache 读写（--no-cache flag）。
	// 默认 false = cache 透明启用。
	NoCache bool

	// CacheDir 指定 cache DB 的目录；空字符串时 pipeline 使用 cache.DefaultDir()。
	CacheDir string
}

// Validate 校验配置合法性，并对未指定的参数填充动态默认值
func (c *Config) Validate() error {
	if c.Dir == "" {
		return fmt.Errorf("目录路径不能为空")
	}
	if c.Workers < 1 {
		return fmt.Errorf("workers 必须大于 0")
	}
	if c.Extractors <= 0 {
		c.Extractors = runtime.NumCPU() * 5
	}
	if c.ChannelSize < 1 {
		return fmt.Errorf("channel-size 必须大于 0")
	}
	return nil
}
