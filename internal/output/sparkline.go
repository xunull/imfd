package output

import (
	"fmt"
	"strings"
)

// inline 条形 sparkline 实现。
//
// 设计决策（来自 plan-design-review D6）：
//   - 定长 20 字符
//   - 以 caller 提供的 max 为满刻度（典型用法：section 内最大桶 → 全部桶共用一个 max）
//   - 默认 Unicode block："█" 实心 / "░" 空心
//   - IMFD_ASCII=1 时退到 "#" / "."
//   - 尾随百分比 " 50%" 右对齐到 4 字符（含空格）
//
// 例：
//   Bar(1, 4, 20) → "█████░░░░░░░░░░░░░░░  25%"
//   Bar(4, 4, 20) → "████████████████████ 100%"
//   Bar(0, 4, 20) → "░░░░░░░░░░░░░░░░░░░░   0%"
//
// max=0（不该发生，但要防御）时返回全空 bar、0%。

const barLen = 20

// Bar 渲染单条 sparkline + 百分比尾标。
// 不上色——颜色由 caller 包装（dashboard 调 Colorer.Media 包整段）。
func Bar(value, max int) string {
	full, empty := "█", "░"
	if UseASCIIBar() {
		full, empty = "#", "."
	}

	if max <= 0 {
		return strings.Repeat(empty, barLen) + "   0%"
	}

	// value 超过 max（不该发生但兜底）按 max 算
	if value > max {
		value = max
	}
	if value < 0 {
		value = 0
	}

	filled := (value * barLen) / max
	// 边界：value > 0 但太小四舍五入到 0 格，强制给 1 格视觉反馈
	if filled == 0 && value > 0 {
		filled = 1
	}

	bar := strings.Repeat(full, filled) + strings.Repeat(empty, barLen-filled)
	pct := (value * 100) / max
	return fmt.Sprintf("%s %3d%%", bar, pct)
}
