package config

import (
	"fmt"
	"runtime"
	"strings"
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
