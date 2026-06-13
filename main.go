package main

import (
	"fmt"
	"os"

	"github.com/xunull/imfd/cmd"
)

// 版本信息：goreleaser 构建时通过 -ldflags 注入；本地 go build 保持 dev 默认。
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
