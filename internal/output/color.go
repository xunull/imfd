package output

import "github.com/xunull/imfd/internal/media"

// ANSI 颜色 helper。规则：
//
// - 调用方负责把 colors enabled 状态传入（通常通过 newDashboard 构造时探测一次并缓存）
// - 三种媒体类型各有一色（视觉锚点）：image=蓝 / video=紫 / audio=绿
// - section header 用粗体青色
// - 字段名（dim 名）用 dim gray，让数据列突出
//
// 配色挑选避开了 16-color 黄/红（warning/error 语义），未来加错误高亮时不冲突。

const (
	ansiReset    = "\x1b[0m"
	ansiBold     = "\x1b[1m"
	ansiDim      = "\x1b[2m"
	ansiBlue     = "\x1b[34m"
	ansiMagenta  = "\x1b[35m"
	ansiGreen    = "\x1b[32m"
	ansiBoldCyan = "\x1b[1;36m"
)

// Colorer 是 dashboard 内部用的小封装，构造一次后所有调用都是常数时间字符串拼接。
// enabled=false 时所有方法都返回原文本，不加 ANSI。
type Colorer struct {
	enabled bool
}

// NewColorer 根据 TTY 状态 + NO_COLOR 决定是否启用色。
// 用 os.Stdout 探测是因为 dashboard 写的是 stdout。
func NewColorer(enabled bool) *Colorer {
	return &Colorer{enabled: enabled}
}

func (c *Colorer) wrap(prefix, s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return prefix + s + ansiReset
}

// Bold 强调用，section header / 关键数字
func (c *Colorer) Bold(s string) string { return c.wrap(ansiBold, s) }

// Dim 削弱用，次要标签 / 折叠提示行
func (c *Colorer) Dim(s string) string { return c.wrap(ansiDim, s) }

// SectionHeader 是 dashboard 五个 section 标题的统一样式
func (c *Colorer) SectionHeader(s string) string { return c.wrap(ansiBoldCyan, s) }

// Media 根据媒体类型上色：image 蓝、video 紫、audio 绿
// 未知类型返回原色（保守降级）
func (c *Colorer) Media(t media.MediaType, s string) string {
	switch t {
	case media.TypeImage:
		return c.wrap(ansiBlue, s)
	case media.TypeVideo:
		return c.wrap(ansiMagenta, s)
	case media.TypeAudio:
		return c.wrap(ansiGreen, s)
	}
	return s
}
