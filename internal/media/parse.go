package media

import (
	"strconv"
	"strings"
)

// EXIF 数值字段解析器。
//
// imfd 现有 EXIF 提取层已把字段 format 为 string（"f/5"、"1/250s"、"42mm"、"800"），
// 这层把它们 parse 回 typed value，供 list 命令的 DSL evaluator 用。
//
// 设计原则（per plan-eng-review）：strict parse。strconv.Atoi/ParseFloat 失败即返回
// (zero, false)。不做 lenient 解析（"ISO 800" → false，不是 800）；依赖 EXIF 提取
// 层已 normalize。"auto" / "" 等都返回 false。
//
// 所有函数 nil-safe 输入：空字符串返回 (zero, false)。

// ParseISO 解析 EXIF ISO 字符串。
//   "800" → (800, true)
//   "" / "auto" / "ISO 800" → (0, false)
func ParseISO(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ParseAperture 解析光圈 f 值字符串。
//   "f/5" / "F/5" → (5.0, true)
//   "f/2.8" → (2.8, true)
//   "5" / "5.0" → (5.0, true)（已是 bare digit 也接受）
//   "" → (0, false)
func ParseAperture(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// 去掉 "f/" 或 "F/" 前缀
	lower := strings.ToLower(s)
	lower = strings.TrimPrefix(lower, "f/")
	f, err := strconv.ParseFloat(lower, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ParseShutter 解析快门速度字符串为秒数。
//   "1/250s" → (0.004, true)
//   "1/4000" → (0.00025, true)
//   "30s" → (30.0, true)
//   "30" → (30.0, true)
//   "0.5s" → (0.5, true)
//   "" / "auto" → (0, false)
func ParseShutter(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// 去掉尾部 "s"（秒单位标记）
	s = strings.TrimSuffix(s, "s")
	s = strings.TrimSuffix(s, "S")

	// 处理分数形式 "1/250"
	if i := strings.Index(s, "/"); i > 0 {
		num, err1 := strconv.ParseFloat(s[:i], 64)
		den, err2 := strconv.ParseFloat(s[i+1:], 64)
		if err1 != nil || err2 != nil || den == 0 {
			return 0, false
		}
		return num / den, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ParseFocal 解析焦距字符串为毫米数。
//   "42mm" → (42.0, true)
//   "85.5mm" → (85.5, true)
//   "50" → (50.0, true)
//   "" → (0, false)
func ParseFocal(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.TrimSuffix(strings.ToLower(s), "mm")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
