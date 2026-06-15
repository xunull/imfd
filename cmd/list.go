package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/output"
	"github.com/xunull/imfd/internal/pipeline"
	"github.com/xunull/imfd/internal/query"
)

var (
	flagListType         string
	flagListCameraMakes  []string
	flagListCameraModels []string
	flagListLensModels   []string
	flagListDevice       string
	flagListProvinces    []string
	flagListCities       []string
	flagListScene        string
	flagListISO          string
	flagListYear         string
	flagListCodecs       []string // 同时匹配 audio_codec / video_codec
	flagListAudioCodecs  []string
	flagListVideoCodecs  []string
	flagListFilter       string
	flagListEdited       bool
	flagListOOC          bool
	flagListAI           bool
	flagListNotAI        bool
	flagListPrint0       bool
	flagListNoCache      bool
	flagListWorkers      int
	flagListExtractors   int
	flagListChannelSize  int
	flagListGeoProvider  string
)

// listRunner 注入接缝（同 scan/info）便于测试 fake 路由
var listRunner = runList

// listCmd `imfd list` 顶级命令
var listCmd = &cobra.Command{
	Use:   "list [path...]",
	Short: "按条件过滤媒体文件并列出路径（pipe 友好）",
	Long: `list 是 imfd 的「按条件挑选文件」命令，与 scan（聚合统计）/info（单文件详情）相对。

输出每行一个路径（默认 LF 分隔；-0/--print0 用 NUL 分隔 xargs 友好）。

简单查询用 flag：
  imfd list --type image --province 云南 --device phone ~/Pictures

高阶查询用 --filter DSL：
  imfd list --filter "device_type == 'phone' and province contains '云南'" ~/Pictures

flag 和 --filter 是 AND 关系；同名 flag 多次 = OR。

Exit codes:
  0  success (0 结果也是 0；xargs 友好)
  1  IO 错误（路径不存在 / 权限拒绝）
  2  filter 语法错误（stderr 印 column number）`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listRunner(args, os.Stdout, os.Stderr)
	},
}

func init() {
	listCmd.Flags().StringVarP(&flagListType, "type", "t", "all", "媒体类型: image, video, audio, all")
	listCmd.Flags().StringSliceVar(&flagListCameraMakes, "camera-make", nil, "相机品牌（substring case-insensitive；可重复=OR）")
	listCmd.Flags().StringSliceVar(&flagListCameraModels, "camera-model", nil, "相机型号（substring case-insensitive；可重复=OR）")
	listCmd.Flags().StringSliceVar(&flagListLensModels, "lens", nil, "镜头型号（substring；可重复=OR）")
	listCmd.Flags().StringVar(&flagListDevice, "device", "", "设备类别: phone 或 camera")
	listCmd.Flags().StringSliceVar(&flagListCodecs, "codec", nil, "编解码器：同时匹配 audio_codec / video_codec（可重复=OR），例 --codec flac --codec h264")
	listCmd.Flags().StringSliceVar(&flagListAudioCodecs, "audio-codec", nil, "仅音频编解码器（可重复=OR），例 --audio-codec flac")
	listCmd.Flags().StringSliceVar(&flagListVideoCodecs, "video-codec", nil, "仅视频编解码器（可重复=OR）")
	listCmd.Flags().StringSliceVar(&flagListProvinces, "province", nil, "省份（substring；可重复=OR）")
	listCmd.Flags().StringSliceVar(&flagListCities, "city", nil, "城市（substring；可重复=OR）")
	listCmd.Flags().StringVar(&flagListScene, "scene", "", "场景: v1 仅 starry_sky")
	listCmd.Flags().StringVar(&flagListISO, "iso", "", "ISO: N | >N | <N | >=N | <=N | N-M")
	listCmd.Flags().StringVar(&flagListYear, "year", "", "拍摄年份: N | >=N | N-M")
	listCmd.Flags().StringVarP(&flagListFilter, "filter", "f", "", "expr-lang DSL filter（高阶；和 flag 是 AND）")
	listCmd.Flags().BoolVar(&flagListEdited, "edited", false, "只看编辑过的图像（Lightroom / Photoshop / 后期工具）；与 --ooc 互斥")
	listCmd.Flags().BoolVar(&flagListOOC, "ooc", false, "只看 out-of-camera 直出图像；与 --edited 互斥")
	listCmd.MarkFlagsMutuallyExclusive("edited", "ooc")
	listCmd.Flags().BoolVar(&flagListAI, "ai", false, "只看 AI 生成图像（C2PA / DALL·E / Midjourney / SD 等）；与 --not-ai 互斥")
	listCmd.Flags().BoolVar(&flagListNotAI, "not-ai", false, "排除 AI 生成图像；与 --ai 互斥")
	listCmd.MarkFlagsMutuallyExclusive("ai", "not-ai")
	listCmd.Flags().BoolVarP(&flagListPrint0, "print0", "0", false, "用 NUL 分隔输出（xargs -0 友好）")
	listCmd.Flags().BoolVar(&flagListNoCache, "no-cache", false, "跳过 cache 读写（强制重新提取）")
	listCmd.Flags().IntVarP(&flagListWorkers, "workers", "w", 8, "目录遍历并发数")
	listCmd.Flags().IntVarP(&flagListExtractors, "extractors", "e", 0, "媒体提取并发数（默认 CPU*5）")
	listCmd.Flags().IntVar(&flagListChannelSize, "channel-size", 1024, "内部通道缓冲")
	listCmd.Flags().StringVarP(&flagListGeoProvider, "geo-provider", "g", "offline", "GPS 反查: offline / nominatim")
}

// runList 执行 list 流水线。
//   1. BuildFilter → expr string + needles
//   2. NewEvaluator compile（compile error → exit 2 via SyntaxError）
//   3. paths 默认 "."；遍历每个 path 调 RunWithHandler，handler filter+print
//
// 返回的 error 由 main.go 印；exit code 由 main.go 决定（默认 1）。
// SyntaxError 类错误，main.go 通过 errors.Is 识别并 exit 2（如果未来加）。
// v1 简化：直接返回 err，main 印一行 + exit 1，足够。
func runList(paths []string, stdout, stderr io.Writer) error {
	// BuildFilter
	flags := query.ListFlags{
		Type:         flagListType,
		CameraMakes:  flagListCameraMakes,
		CameraModels: flagListCameraModels,
		LensModels:   flagListLensModels,
		DeviceType:   flagListDevice,
		Provinces:    flagListProvinces,
		Cities:       flagListCities,
		Scene:        flagListScene,
		ISO:          flagListISO,
		Year:         flagListYear,
		Codecs:       flagListCodecs,
		AudioCodecs:  flagListAudioCodecs,
		VideoCodecs:  flagListVideoCodecs,
		Edited:       flagListEdited,
		OOC:          flagListOOC,
		AI:           flagListAI,
		NotAI:        flagListNotAI,
	}
	expr, needles := query.BuildFilter(flags, flagListFilter)

	// Compile evaluator
	ev, err := query.NewEvaluator(expr, needles)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, query.SyntaxError) {
			os.Exit(2) // exit 2 = syntax error per plan
		}
		return err
	}

	// 默认 path = "."
	if len(paths) == 0 {
		paths = []string{"."}
	}

	printer := output.NewListPrinter(stdout, flagListPrint0)
	handler := pipeline.HandlerFunc(func(record *media.MediaRecord) error {
		hit, mErr := ev.Match(record)
		if mErr != nil {
			fmt.Fprintf(stderr, "warning: eval error: %v\n", mErr)
			return nil
		}
		if hit {
			return printer.Print(record.FilePath)
		}
		return nil
	})

	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", p, err)
			return err
		}
		if !fi.IsDir() {
			fmt.Fprintf(stderr, "error: %s 不是目录；list 接受目录路径，单文件请用 'imfd info'\n", p)
			return fmt.Errorf("%s is not a directory", p)
		}

		cfg := &config.Config{
			Dir:         p,
			Workers:     flagListWorkers,
			Extractors:  flagListExtractors,
			ChannelSize: flagListChannelSize,
			GeoProvider: flagListGeoProvider,
			MediaTypes:  parseTypeFilter(flagListType),
			NoCache:     flagListNoCache,
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		if err := pipeline.RunWithHandler(cfg, handler); err != nil {
			return err
		}
	}
	return nil
}

// parseTypeFilter 把 --type 字符串映射到 walker 的 MediaTypes 过滤。
// 这层是 walker 层的预过滤（不必扫所有，再 DSL 过滤）：减少不必要的 extract 调用。
func parseTypeFilter(t string) []media.MediaType {
	switch t {
	case "image":
		return []media.MediaType{media.TypeImage}
	case "video":
		return []media.MediaType{media.TypeVideo}
	case "audio":
		return []media.MediaType{media.TypeAudio}
	default:
		return nil // all
	}
}
