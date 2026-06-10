package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/extract"
	"github.com/xunull/imfd/internal/geo"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/output"
)

var (
	flagInfoFormat      string
	flagInfoGeoProvider string
)

// infoRunner 是单文件 info 的可替换执行体。用于测试注入 fake runner
// 验证参数路由（同 scanRunner 的接缝模式）。
var infoRunner = runInfo

// infoCmd 顶级单文件查询命令。和 scan（聚合统计）平级、语义不同：
// info 看一个文件的全部元数据，scan 看一堆文件的聚合分布。
//
// 用法：
//   imfd info photo.jpg                 — 单文件
//   imfd info *.jpg                     — shell glob 多文件
//   find . -name '*.heic' | xargs imfd info
//
// 多文件错误模型（plan Q1 决议）：某个文件失败不中断后面文件，
// 错误印到 stderr，末尾 exit 1（如有任一失败）。
var infoCmd = &cobra.Command{
	Use:   "info <file>...",
	Short: "查看一个或多个媒体文件的完整元数据",
	Long: `info 是 imfd 的「看单文件」命令，与 scan 的「看一堆文件」相对。

输出按 section 分组（FILE / EXIF / GPS / AUDIO / VIDEO / ERRORS），
缺失的 section 自动隐藏。支持 shell glob：

  imfd info *.jpg
  find . -name '*.heic' | xargs imfd info

某个文件失败不会中断后面文件；末尾返回非零退出码如果有任一失败。`,
	Args: cobra.MinimumNArgs(1),
	// 我们已在 runInfo 里把错误 print 到 stderr；让 cobra 别再印 Usage + Error 重复
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return infoRunner(args, flagInfoFormat, flagInfoGeoProvider, os.Stdout, os.Stderr)
	},
}

func init() {
	infoCmd.Flags().StringVarP(&flagInfoFormat, "format", "f", "table",
		"输出格式: table（人类可读 section 分组）, json（marshal MediaRecord）")
	infoCmd.Flags().StringVarP(&flagInfoGeoProvider, "geo-provider", "g", "offline",
		"GPS 反查方式: offline(离线), nominatim(OpenStreetMap在线)")
}

// runInfo 是 infoRunner 的默认实现。
//
// 一次性构造 resolver（offline 加载中国城市表 ~30ms，跨多文件复用），
// 循环里 if record.HasGPS() 才调 Resolve。
//
// 错误处理（plan Q1）：
//   - 每个文件独立 try：os.Stat / IsDir / extract 失败都打印 stderr，记失败数
//   - 文件间用空行 + 分隔线分开
//   - 末尾：返回简短摘要 err（"N file(s) failed"），让 main.go 印一行总结
//     而不是重复每文件的详细错误（已在 stderr 印过）
func runInfo(paths []string, format, geoProvider string, stdout, stderr io.Writer) error {
	// 构造 resolver。错误立即返回（这种错是 config 性的，应该 fail-fast）。
	gp, err := geo.ParseGeoProvider(geoProvider)
	if err != nil {
		return err
	}
	resolver, err := geo.NewResolver(gp)
	if err != nil {
		return fmt.Errorf("创建地理反查器失败: %w", err)
	}

	printer := output.NewFileInfoPrinter(stdout, format)

	failed := 0
	prevPrinted := false
	for _, path := range paths {
		// 只有上一个文件真的打到 stdout 才插分隔线；失败的文件不占 stdout，
		// 否则会出现「分隔线 → 分隔线」的幽灵空段
		if prevPrinted {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, output.SectionSeparator())
			fmt.Fprintln(stdout)
		}
		if err := infoOne(path, resolver, printer, stderr); err != nil {
			failed++
			prevPrinted = false
		} else {
			prevPrinted = true
		}
	}
	if failed > 0 {
		// main.go 会 fmt.Fprintln(os.Stderr, err) 这个简短摘要；
		// 每文件详情已经在 infoOne 里 stderr 印过，不重复
		return fmt.Errorf("%d file(s) failed (see stderr above)", failed)
	}
	return nil
}

// infoOne 处理单个文件路径：stat → resolve type → extract → print。
// 错误打到 stderr 同时返回，让 caller 决定是否累计 / 中断。
func infoOne(path string, resolver geo.GeoResolver, printer *output.FileInfoPrinter, stderr io.Writer) error {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s: %v\n", path, err)
		return err
	}
	if fi.IsDir() {
		err := fmt.Errorf("%s 是目录；要看汇总统计请用 'imfd scan %s'，要看每个文件请用 'imfd info %s/*'", path, path, path)
		fmt.Fprintf(stderr, "error: %v\n", err)
		return err
	}

	record := extract.Extract(path)
	if record.HasGPS() && resolver != nil {
		if loc, e := resolver.Resolve(record.Exif.GPS.Latitude, record.Exif.GPS.Longitude); e == nil {
			record.Location = loc
		}
	}

	return printer.Print(record)
}

// 编译期断言：确保 media 包被导入（无副作用，编译时类型检查）
var _ = media.TypeUnknown
