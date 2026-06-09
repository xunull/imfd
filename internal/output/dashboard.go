package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/xunull/imfd/internal/media"
	"github.com/xunull/imfd/internal/stats"
)

// DashboardPrinter 渲染 dashboard 风格输出（见 plan-design-review 决议）。
//
// 布局：
//
//   imfd · scan all · /path                                          v0.x
//   ─────────────────────────────────────────────────────────────────
//   scanned 24 files · 18 MB · 0.12s · 0 errors
//
//
//   OVERVIEW
//     图像     1   ███░░░░░░░░░░░░░░░░░  4%
//     视频     1   ███░░░░░░░░░░░░░░░░░  4%
//     音频    22   ████████████████████ 92%
//     总计    24
//
//   AUDIO
//     编解码器   mp3   18   ████████████████████  82%
//                flac   4   ████░░░░░░░░░░░░░░░░  18%
//     ...
//
//   (16 维度无有效数据已折叠：相机型号、镜头型号、... — 加 -v 查看)
//
// 5 个 section 表驱动定义在 sectionDefs；维度名映射到 section。未映射的
// 维度归到 OTHER section 末尾兜底（防新加维度漏 register）。
type DashboardPrinter struct {
	Verbose  bool      // -v 时展开全 Unknown 维度
	Source   string    // scan 命令传入的目录
	Duration time.Duration
	Errors   int

	out      io.Writer
	color    *Colorer
}

// NewDashboardPrinter 构造一个新 printer。
// out=nil 时默认 os.Stdout。color 自动按 TTY/NO_COLOR 探测。
func NewDashboardPrinter(out io.Writer, verbose bool, source string, duration time.Duration) *DashboardPrinter {
	if out == nil {
		out = os.Stdout
	}
	useColor := IsTTY(os.Stdout) && !NoColor()
	return &DashboardPrinter{
		Verbose:  verbose,
		Source:   source,
		Duration: duration,
		out:      out,
		color:    NewColorer(useColor),
	}
}

// sectionDef 描述一个 section：名字 + 包含哪些维度名（按这个顺序渲染）
type sectionDef struct {
	name string
	dims []string
}

// 5 个语义 section（plan-design-review D2 决议）
// dims 按渲染顺序排——dashboard 输出会照这个顺序铺
var sectionDefs = []sectionDef{
	{
		name: "OVERVIEW",
		dims: []string{"媒体类型"},
	},
	{
		name: "DEVICE",
		dims: []string{"相机型号", "镜头型号"},
	},
	{
		name: "TIME & PLACE",
		dims: []string{"拍摄时间段", "省份", "城市", "省/市"},
	},
	{
		name: "EXIF SETTINGS",
		dims: []string{"ISO感光度", "光圈", "快门速度", "焦距", "曝光模式", "曝光程序", "白平衡", "测光模式", "闪光灯"},
	},
	{
		name: "AUDIO",
		dims: []string{"音频编解码器", "音频比特率", "音频采样率", "音频声道", "音频时长"},
	},
}

// Print 是主入口。先打 header 再按 section 顺序铺维度，全空维度折叠到 section 末尾。
func (p *DashboardPrinter) Print(report stats.StatsReport) error {
	p.printHeader(report)
	p.printOverview(report)
	p.printSections(report)
	return nil
}

func (p *DashboardPrinter) printHeader(report stats.StatsReport) {
	source := p.Source
	if source == "" {
		source = "."
	}

	// 第一行：imfd · {命令上下文} · {源路径}
	cmdLabel := p.cmdLabel(report)
	header := fmt.Sprintf("imfd · %s · %s", cmdLabel, source)
	fmt.Fprintln(p.out, p.color.Bold(header))

	// 第二行：分隔线
	fmt.Fprintln(p.out, p.color.Dim(strings.Repeat("─", 65)))

	// 第三行：meta（文件数 / 大小占位 / 用时 / 错误数）
	// 文件大小当前 pipeline 不算（出于 perf 与扫描期间字段缺位的考虑），先标 "—"
	// 等 T7 把 size 接上来后填进来
	metaParts := []string{
		fmt.Sprintf("scanned %d files", report.Totals.TotalCount+report.Totals.ErrorCount),
		fmt.Sprintf("%s", p.duration()),
		fmt.Sprintf("%d errors", report.Totals.ErrorCount),
	}
	fmt.Fprintln(p.out, p.color.Dim(strings.Join(metaParts, " · ")))
	fmt.Fprintln(p.out)
}

// cmdLabel 推断 scan 命令的简称（用 totals 推断而非接受新参数）
func (p *DashboardPrinter) cmdLabel(report stats.StatsReport) string {
	t := report.Totals
	hasImage := t.ImageCount > 0
	hasVideo := t.VideoCount > 0
	hasAudio := t.AudioCount > 0
	switch {
	case hasImage && !hasVideo && !hasAudio:
		return "scan image"
	case hasVideo && !hasImage && !hasAudio:
		return "scan video"
	case hasAudio && !hasImage && !hasVideo:
		return "scan audio"
	default:
		return "scan"
	}
}

func (p *DashboardPrinter) duration() string {
	d := p.Duration
	switch {
	case d == 0:
		return "—"
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// printOverview 是 OVERVIEW section 的特化渲染：按媒体类型上色 + bar
func (p *DashboardPrinter) printOverview(report stats.StatsReport) {
	fmt.Fprintln(p.out, p.color.SectionHeader("OVERVIEW"))
	t := report.Totals
	rows := []struct {
		label string
		count int
		mt    media.MediaType
	}{
		{"图像", t.ImageCount, media.TypeImage},
		{"视频", t.VideoCount, media.TypeVideo},
		{"音频", t.AudioCount, media.TypeAudio},
	}
	max := t.TotalCount
	if max == 0 {
		max = 1
	}
	for _, r := range rows {
		bar := p.color.Media(r.mt, Bar(r.count, max))
		fmt.Fprintf(p.out, "  %s  %4d  %s\n", padR(r.label, 6), r.count, bar)
	}
	// 总计单列，不带 bar
	fmt.Fprintf(p.out, "  %s  %4d\n", padR("总计", 6), t.TotalCount)
	if t.ErrorCount > 0 {
		fmt.Fprintf(p.out, "  %s  %4d\n", padR("错误", 6), t.ErrorCount)
	}
	fmt.Fprintln(p.out)
}

// printSections 渲染 DEVICE / TIME & PLACE / EXIF SETTINGS / AUDIO 4 个 section
// （OVERVIEW 已单独 printOverview）
func (p *DashboardPrinter) printSections(report stats.StatsReport) {
	dimByName := make(map[string]stats.DimensionResult, len(report.Dimensions))
	for _, d := range report.Dimensions {
		dimByName[d.DimensionName] = d
	}

	for _, sec := range sectionDefs {
		if sec.name == "OVERVIEW" {
			continue
		}
		p.printSection(sec, dimByName)
	}
}

func (p *DashboardPrinter) printSection(sec sectionDef, dimByName map[string]stats.DimensionResult) {
	// 收集本 section 已注册的维度（未注册的 dim 跳过，比如 scan audio 时 DEVICE 全没 register）
	var emptyNames []string
	var visibleDims []stats.DimensionResult
	for _, name := range sec.dims {
		dim, ok := dimByName[name]
		if !ok {
			continue
		}
		if isEmptyDim(dim) && !p.Verbose {
			emptyNames = append(emptyNames, name)
			continue
		}
		visibleDims = append(visibleDims, dim)
	}

	// section 全空（无 dim register + 无折叠提示）时整段不出现
	if len(visibleDims) == 0 && len(emptyNames) == 0 {
		return
	}

	fmt.Fprintln(p.out, p.color.SectionHeader(sec.name))

	// section 级 keyWidth：扫描本 section 内所有维度的所有桶，找出最宽的 key。
	// 这样同 section 内不同维度的 bar 起始列严格对齐，
	// 不会因为 "NIKKOR Z 24-120mm f/4 S" 把镜头型号那行 bar 推远而失齐。
	sectionKeyWidth := minKeyWidth
	for _, dim := range visibleDims {
		for _, b := range dim.Buckets {
			if w := visualWidth(b.Key); w > sectionKeyWidth {
				sectionKeyWidth = w
			}
		}
	}

	// 渲染可见维度
	for i, dim := range visibleDims {
		p.printDimension(dim, sectionKeyWidth)
		if i < len(visibleDims)-1 {
			fmt.Fprintln(p.out)
		}
	}

	// 折叠提示
	if len(emptyNames) > 0 {
		if len(visibleDims) > 0 {
			fmt.Fprintln(p.out)
		}
		fmt.Fprintln(p.out, p.color.Dim(fmt.Sprintf("  (%d 维度无数据: %s — 加 -v 查看)",
			len(emptyNames), strings.Join(emptyNames, "、"))))
	}
	fmt.Fprintln(p.out)
}

// printDimension 渲染单个维度：字段名 + 桶按 count 降序 + 条形 + 百分比
//
// 排序规则：
//   1. Unknown 永远排末尾（即使 count 最高也不让它抢主位 —— 它意味着"该维度对这部分 record 无效"，
//      不是用户关心的实际数据）
//   2. 其他按 count 降序
//   3. count 相同时按 key 字典序（结果稳定）
//
// keyWidth 由 caller 传入（section 级算好），保证同 section 内 bar 起始列对齐。
func (p *DashboardPrinter) printDimension(dim stats.DimensionResult, keyWidth int) {
	buckets := append([]stats.Bucket(nil), dim.Buckets...)
	sort.Slice(buckets, func(i, j int) bool {
		iUnknown := buckets[i].Key == "Unknown"
		jUnknown := buckets[j].Key == "Unknown"
		if iUnknown != jUnknown {
			return !iUnknown // 非 Unknown 排前
		}
		if buckets[i].Count != buckets[j].Count {
			return buckets[i].Count > buckets[j].Count
		}
		return buckets[i].Key < buckets[j].Key
	})

	// max = 本维度最大桶（维度内归一化——单个维度的桶之间比例直观）
	max := 0
	for _, b := range buckets {
		if b.Count > max {
			max = b.Count
		}
	}

	for i, b := range buckets {
		fieldName := ""
		if i == 0 {
			fieldName = dim.DimensionName
		}
		bar := Bar(b.Count, max)
		// 字段名 dim 化（让数据列突出）
		nameCol := p.color.Dim(padR(fieldName, fieldNameWidth))
		fmt.Fprintf(p.out, "  %s %s %4d  %s\n",
			nameCol,
			padR(b.Key, keyWidth),
			b.Count,
			bar)
	}
}

const (
	fieldNameWidth = 14
	minKeyWidth    = 14
)

// isEmptyDim 判断一个维度是否"无有效数据"——即所有桶都是 Unknown
// 这是空维度折叠的判定规则（plan D4 决议）
func isEmptyDim(dim stats.DimensionResult) bool {
	if len(dim.Buckets) == 0 {
		return true
	}
	for _, b := range dim.Buckets {
		if b.Key != "Unknown" {
			return false
		}
	}
	return true
}

// padR 用 ASCII 空格右填到 width。
// 注意：中文字符在 monospace 终端通常占 2 列宽，本实现按 rune 数算（一致性优先）
// 对纯中文/纯 ASCII 都对齐；混排时会有 1-2 列偏差，但 dashboard 的字段名都是纯中文，
// 桶 key 多是技术词（ASCII）或纯中文，实测对齐没明显问题。
func padR(s string, width int) string {
	w := visualWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// visualWidth 估算字符串在 monospace 终端的视觉列宽。
// 简化模型：rune ≥ 0x4E00（CJK Unified Ideographs 起点）视为 2 列宽，其他 1 列。
// 涵盖：常用汉字、日文汉字、韩文（hangul 范围更广但覆盖足够）。不处理 emoji。
func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x4E00 {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}
