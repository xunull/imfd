package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/extract"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/output"
)

var (
	flagVerifyFormat string
	flagVerifyC2PA   bool
)

// verifyRunner 是命令的可替换执行体（同 scanRunner / infoRunner / listRunner / viewRunner）。
// 测试通过替换它来验证子命令路由与 flag 解析正确。
var verifyRunner = runVerify

// verifyCmd 「侦探」命令：检测一张图像是否被编辑过、是相机直出、还是 AI 生成（v2）。
//
// 与 imfd info（看所有 EXIF）不同：
//   info     —— 展示全部元数据，不下判断
//   verify   —— 聚焦「这张图被处理过吗」，给出 verdict + signals
//
// 与 imfd list --edited（批量过滤）不同：
//   list --edited —— 输出符合条件的文件路径（pipe 友好）
//   verify        —— 单/多文件展开人类可读的判定原因
var verifyCmd = &cobra.Command{
	Use:   "verify <file>...",
	Short: "侦探单个或多个图像文件：AI 生成？后期编辑？",
	Long: `verify 检测一张图像的「身世」，给出两个独立维度的判定：

  AI 生成   ai-generated / not-ai / unknown
            信号：C2PA Content Credentials manifest、EXIF/PNG Software 含 AI 工具名、
            Stable Diffusion / ComfyUI 在 PNG 写的 prompt/参数。
  后期编辑  original / camera-rendered / edited / unknown
            信号：Software 字段、ModifyDate 比 DateTimeOriginal 晚多少。

注意：detection-only —— 只读元数据声明，不验证密码学签名。Software 和 C2PA
manifest 都可被伪造，不要作为法律证据或内容审核唯一依据。

示例：
  # 单文件人类可读报告（AI + 编辑双判定）
  imfd verify ~/photo.jpg

  # 展开 C2PA manifest 详情（生成器、信任级别）
  imfd verify ~/photo.jpg --c2pa

  # JSON 输出（脚本友好）
  imfd verify ~/photo.jpg -f json

  # 批量审计：找出图库里所有 AI 生成图 / 编辑过的照片
  imfd list --ai ~/Photos
  imfd list --edited ~/Photos

非图像文件（mp4 / mp3）会被 skip，不影响后续文件的处理。
当 verify 多文件、任一文件失败时，末尾整体 exit 1；具体错误已经在 stderr 印出。`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return verifyRunner(args, flagVerifyFormat, flagVerifyC2PA, os.Stdout, os.Stderr)
	},
}

func init() {
	verifyCmd.Flags().StringVarP(&flagVerifyFormat, "format", "f", "table",
		"输出格式: table（人类可读）, json（结构化）")
	verifyCmd.Flags().BoolVar(&flagVerifyC2PA, "c2pa", false,
		"展开 C2PA Content Credentials manifest 详情（生成器、信任级别）")
}

// runVerify 是 verifyRunner 的默认实现。
//
// 处理流程：
//  1. 构造 VerifyPrinter（自动 TTY/NO_COLOR 探测）
//  2. 每个文件独立 try-evaluate：
//     - os.Stat 失败 → stderr 报错，记失败数
//     - 调 extract.Extract（与 info / list 走同一路径，自动按扩展名 dispatch image/video/audio）
//     - 把 record 喂给 printer
//  3. 文件间用 separator 隔开（仅 table 模式）
//  4. 末尾：任一文件失败 → 返回简短摘要 error（main 印一行 + exit 1）
func runVerify(paths []string, format string, c2paDetail bool, stdout, stderr io.Writer) error {
	printer := output.NewVerifyPrinter(stdout, format, c2paDetail)

	failed := 0
	prevPrinted := false
	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", path, err)
			failed++
			continue
		}
		if fi.IsDir() {
			fmt.Fprintf(stderr,
				"error: %s 是目录；verify 接受单个或多个文件路径；批量审计请用 'imfd list --edited %s'\n",
				path, path)
			failed++
			continue
		}

		// 复用 extract.Extract：自动按扩展名 dispatch（非图像走 audio / video / unknown 路径）
		record := extract.Extract(path)
		if record.Error != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", path, record.Error)
			failed++
			continue
		}

		// 文件间分隔（仅 table 模式且上一个文件真的渲染了）
		if prevPrinted {
			printer.PrintSeparator()
		}
		if err := printer.Print(record); err != nil {
			fmt.Fprintf(stderr, "error: render %s: %v\n", path, err)
			failed++
			prevPrinted = false
			continue
		}
		prevPrinted = true
	}

	// 末尾标记：图像 vs verdict 分布的快速摘要（仅 table、>=2 文件时印）
	// 单文件 verify 不需要这层，重复信息。
	// 暂不加，保持输出简洁——如果未来需要可以加 --summary flag。

	_ = media.VerdictUnknown // 防 unused import；后续 summary 可能用到

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed (see stderr above)", failed)
	}
	return nil
}
