package cmd

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xunull/imfd/internal/config"
	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/pipeline"
	"github.com/xunull/imfd/internal/query"
)

var (
	flagViewType         string
	flagViewCameraMakes  []string
	flagViewCameraModels []string
	flagViewLensModels   []string
	flagViewDevice       string
	flagViewCodecs       []string
	flagViewAudioCodecs  []string
	flagViewVideoCodecs  []string
	flagViewProvinces    []string
	flagViewCities       []string
	flagViewScene        string
	flagViewISO          string
	flagViewYear         string
	flagViewFilter       string
	flagViewRename       string
	flagViewNoOpen       bool
	flagViewNoCache      bool
	flagViewWorkers      int
	flagViewExtractors   int
	flagViewChannelSize  int
	flagViewGeoProvider  string
)

// ErrMacOSOnly is returned when view is run on a non-macOS platform.
// RunE detects it and calls os.Exit(2) so runView itself is fully testable.
var ErrMacOSOnly = errors.New("imfd view 目前仅支持 macOS（Windows/Linux 请使用 imfd list 配合文件管理器）")

// currentOS is a var (not const) so tests can override it.
var currentOS = runtime.GOOS

// openDir is injectable for tests; real impl calls macOS `open`.
var openDir = func(dir string) error {
	return exec.Command("open", dir).Run()
}

// viewRunner is the injection seam (same pattern as scanRunner / listRunner).
var viewRunner = runView

var viewCmd = &cobra.Command{
	Use:   "view [path...]",
	Short: "按条件筛选媒体文件并在 Finder 中打开虚拟视图（仅 macOS）",
	Long: `view 在系统临时目录下创建符号链接虚拟目录，通过 Finder 查看筛选结果。

原始文件不会被移动或修改。同一查询条件生成同一虚拟目录（重复运行刷新内容）。
关闭 Finder 窗口或重启后虚拟目录自动消失。

示例：
  imfd view --province 云南 ~/Pictures
  imfd view --device phone --year 2024 ~/Photos --rename "{date}_{city}.{ext}"
  imfd view --filter "iso > 1600" ~/Photos --no-open

仅支持 macOS。Windows / Linux 用户请使用 imfd list 输出路径配合文件管理器。`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := viewRunner(args, os.Stdout, os.Stderr)
		if errors.Is(err, ErrMacOSOnly) {
			os.Exit(2)
		}
		return err
	},
}

func init() {
	viewCmd.Flags().StringVarP(&flagViewType, "type", "t", "all", "媒体类型: image, video, audio, all")
	viewCmd.Flags().StringSliceVar(&flagViewCameraMakes, "camera-make", nil, "相机品牌（可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewCameraModels, "camera-model", nil, "相机型号（可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewLensModels, "lens", nil, "镜头型号（可重复=OR）")
	viewCmd.Flags().StringVar(&flagViewDevice, "device", "", "设备类别: phone 或 camera")
	viewCmd.Flags().StringSliceVar(&flagViewCodecs, "codec", nil, "编解码器（同时匹配 audio/video；可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewAudioCodecs, "audio-codec", nil, "音频编解码器（可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewVideoCodecs, "video-codec", nil, "视频编解码器（可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewProvinces, "province", nil, "省份（可重复=OR）")
	viewCmd.Flags().StringSliceVar(&flagViewCities, "city", nil, "城市（可重复=OR）")
	viewCmd.Flags().StringVar(&flagViewScene, "scene", "", "场景: starry_sky")
	viewCmd.Flags().StringVar(&flagViewISO, "iso", "", "ISO: N | >N | <N | N-M")
	viewCmd.Flags().StringVar(&flagViewYear, "year", "", "拍摄年份: N | >=N | N-M")
	viewCmd.Flags().StringVarP(&flagViewFilter, "filter", "f", "", "expr-lang DSL（和 flag 是 AND）")
	viewCmd.Flags().StringVar(&flagViewRename, "rename", "", `symlink 重命名模板，例: "{date}_{city}.{ext}"（默认保留原文件名）`)
	viewCmd.Flags().BoolVar(&flagViewNoOpen, "no-open", false, "只建 symlink，不打开 Finder（输出目录路径到 stdout）")
	viewCmd.Flags().BoolVar(&flagViewNoCache, "no-cache", false, "跳过 cache 读写（强制重新提取）")
	viewCmd.Flags().IntVarP(&flagViewWorkers, "workers", "w", 8, "目录遍历并发数")
	viewCmd.Flags().IntVarP(&flagViewExtractors, "extractors", "e", 0, "媒体提取并发数（默认 CPU*5）")
	viewCmd.Flags().IntVar(&flagViewChannelSize, "channel-size", 1024, "内部通道缓冲")
	viewCmd.Flags().StringVarP(&flagViewGeoProvider, "geo-provider", "g", "offline", "GPS 反查: offline / nominatim")
}

// runView is the testable core of the view command.
func runView(paths []string, stdout, stderr io.Writer) error {
	if currentOS != "darwin" {
		fmt.Fprintln(stderr, "error: "+ErrMacOSOnly.Error())
		return ErrMacOSOnly
	}

	flags := query.ListFlags{
		Type:         flagViewType,
		CameraMakes:  flagViewCameraMakes,
		CameraModels: flagViewCameraModels,
		LensModels:   flagViewLensModels,
		DeviceType:   flagViewDevice,
		Provinces:    flagViewProvinces,
		Cities:       flagViewCities,
		Scene:        flagViewScene,
		ISO:          flagViewISO,
		Year:         flagViewYear,
		Codecs:       flagViewCodecs,
		AudioCodecs:  flagViewAudioCodecs,
		VideoCodecs:  flagViewVideoCodecs,
	}
	filterExpr, needles := query.BuildFilter(flags, flagViewFilter)

	ev, err := query.NewEvaluator(filterExpr, needles)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, query.SyntaxError) {
			os.Exit(2)
		}
		return err
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Compute absolute paths for deterministic hash.
	absPaths := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, _ := filepath.Abs(p)
		absPaths = append(absPaths, abs)
	}
	vDir := viewDirPath(filterExpr, absPaths)

	if err := os.MkdirAll(vDir, 0o755); err != nil {
		return fmt.Errorf("创建虚拟视图目录失败: %w", err)
	}
	cleanOldSymlinks(vDir)

	sh := &symlinkHandler{viewDir: vDir, rename: flagViewRename}

	for _, p := range paths {
		fi, statErr := os.Stat(p)
		if statErr != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", p, statErr)
			return statErr
		}
		if !fi.IsDir() {
			fmt.Fprintf(stderr, "error: %s 不是目录；单文件请用 'imfd info'\n", p)
			return fmt.Errorf("%s is not a directory", p)
		}

		cfg := &config.Config{
			Dir:         p,
			Workers:     flagViewWorkers,
			Extractors:  flagViewExtractors,
			ChannelSize: flagViewChannelSize,
			GeoProvider: flagViewGeoProvider,
			MediaTypes:  parseTypeFilter(flagViewType),
			NoCache:     flagViewNoCache,
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		handler := pipeline.HandlerFunc(func(record *media.MediaRecord) error {
			hit, mErr := ev.Match(record)
			if mErr != nil {
				fmt.Fprintf(stderr, "warning: eval error: %v\n", mErr)
				return nil
			}
			if hit {
				return sh.link(record)
			}
			return nil
		})

		if err := pipeline.RunWithHandler(cfg, handler); err != nil {
			return err
		}
	}

	if sh.count == 0 {
		fmt.Fprintln(stderr, "0 files matched，未打开 Finder")
		return nil
	}

	fmt.Fprintf(stderr, "%d files → %s\n", sh.count, vDir)
	fmt.Fprintln(stdout, vDir)

	if !flagViewNoOpen {
		if err := openDir(vDir); err != nil {
			// Non-fatal: user can still navigate to the path printed on stdout.
			fmt.Fprintf(stderr, "warning: 打开 Finder 失败: %v\n", err)
		}
	}
	return nil
}

// -- helpers -----------------------------------------------------------------

// viewDirPath returns a deterministic /tmp/imfd-view-XXXXXXXX path based on
// the filter expression and sorted absolute source paths.
func viewDirPath(filterExpr string, absPaths []string) string {
	h := fnv.New32a()
	fmt.Fprintln(h, filterExpr)
	sorted := slices.Clone(absPaths)
	slices.Sort(sorted)
	for _, p := range sorted {
		fmt.Fprintln(h, p)
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("imfd-view-%08x", h.Sum32()))
}

// cleanOldSymlinks removes symlinks from a previous run of the same view.
// Regular files are left untouched (user might have placed them there manually).
func cleanOldSymlinks(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// symlinkHandler creates one symlink per matched record.
// Called from pipeline stage 3 (single goroutine) — no mutex needed.
type symlinkHandler struct {
	viewDir string
	rename  string // template; "" = keep original filename
	count   int
}

func (h *symlinkHandler) link(r *media.MediaRecord) error {
	if r.Error != nil {
		return nil
	}
	// symlink target must be absolute: relative paths are resolved from the
	// symlink's directory (/tmp/imfd-view-xxx/), not the working directory.
	src, err := filepath.Abs(r.FilePath)
	if err != nil {
		src = r.FilePath
	}
	name := filepath.Base(src)
	if h.rename != "" {
		name = applyViewTemplate(h.rename, r)
	}
	dst := uniqueSymlinkPath(h.viewDir, name)
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("创建 symlink 失败 (%s): %w", src, err)
	}
	h.count++
	return nil
}

// uniqueSymlinkPath returns a collision-free path under dir for the given name.
// If dir/name exists, tries dir/base_1.ext, dir/base_2.ext, ...
func uniqueSymlinkPath(dir, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(dir, name)
	for i := 1; ; i++ {
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
	}
}

// applyViewTemplate replaces {var} placeholders with metadata values.
// Missing metadata falls back to "Unknown" for strings, mtime for dates.
func applyViewTemplate(tmpl string, r *media.MediaRecord) string {
	t := r.ModTime
	if r.HasCaptureTime {
		t = r.CaptureTime
	}
	year  := fmt.Sprintf("%04d", t.Year())
	month := fmt.Sprintf("%02d", int(t.Month()))
	day   := fmt.Sprintf("%02d", t.Day())

	make_, model, iso_ := "Unknown", "Unknown", "0"
	if r.Exif != nil {
		if r.Exif.CameraMake != "" {
			make_ = sanitizeFilename(r.Exif.CameraMake)
		}
		if r.Exif.CameraModel != "" {
			model = sanitizeFilename(r.Exif.CameraModel)
		}
		if r.Exif.ISO != "" {
			iso_ = sanitizeFilename(r.Exif.ISO)
		}
	}

	city_, prov_ := "Unknown", "Unknown"
	if r.Location != nil {
		if r.Location.City != "" {
			city_ = sanitizeFilename(r.Location.City)
		}
		if r.Location.Province != "" {
			prov_ = sanitizeFilename(r.Location.Province)
		}
	}

	ext      := strings.ToLower(strings.TrimPrefix(filepath.Ext(r.FilePath), "."))
	filename := strings.TrimSuffix(filepath.Base(r.FilePath), filepath.Ext(r.FilePath))

	return strings.NewReplacer(
		"{year}", year,
		"{month}", month,
		"{day}", day,
		"{date}", year+"-"+month+"-"+day,
		"{camera_make}", make_,
		"{camera_model}", model,
		"{city}", city_,
		"{province}", prov_,
		"{type}", r.Type.String(),
		"{iso}", iso_,
		"{ext}", ext,
		"{filename}", filename,
	).Replace(tmpl)
}

// sanitizeFilename strips characters that are invalid in macOS filenames.
func sanitizeFilename(s string) string {
	return strings.NewReplacer(
		"/", "_",
		":", "_",
		string([]byte{0}), "",
	).Replace(strings.TrimSpace(s))
}
