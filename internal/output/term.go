package output

import (
	"os"

	xterm "golang.org/x/term"
)

// 终端感知层。所有 dashboard / spinner / color 都先走这里查询：
// - 输出走到 pipe / 文件时不上色、不画 spinner
// - 用户设了 NO_COLOR=1 时禁色（https://no-color.org/ 约定）
// - 用户设了 IMFD_ASCII=1 时 sparkline 走 ASCII fallback
// 这些查询是无副作用的、可并发安全的：实现上每次现查环境变量与 fd 状态。

// IsTTY 检查给定文件描述符是否是终端。stdout/stderr pipe 到文件时返回 false。
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	return xterm.IsTerminal(int(f.Fd()))
}

// NoColor 用户明确要求不上色。
// 触发条件：环境变量 NO_COLOR 非空，或 IMFD_NO_COLOR 非空。
// 即使在 TTY 下，用户也可以强制禁用。
func NoColor() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("IMFD_NO_COLOR") != ""
}

// UseASCIIBar 用户要求 sparkline 用纯 ASCII（'#' '.'）而不是 Unicode block。
// 触发条件：IMFD_ASCII=1。古老 SSH / Windows cmd 等场景的兜底。
func UseASCIIBar() bool {
	return os.Getenv("IMFD_ASCII") == "1"
}

// TermWidth 返回终端宽度（列数）。无法探测时返回 80（保守默认）。
// dashboard 暂不用这个动态调整 bar 长度（D6 选了定长 20），保留为将来扩展。
func TermWidth() int {
	if w, _, err := xterm.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}
