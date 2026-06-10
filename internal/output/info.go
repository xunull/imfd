package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/xunull/imfd/internal/media"
)

// FileInfoPrinter 单文件详情输出（imfd info 命令的渲染层）。
//
// 与 DashboardPrinter（scan）区别：
//   - DashboardPrinter 按维度聚合，含 bar/百分比
//   - FileInfoPrinter 按 section 分组的 key-value 平铺，无聚合视觉
//
// 5 个候选 section 表驱动；空 section（含字段全空）自动隐藏。
// 格式：
//   table — section 分组人读
//   json  — 直接 json.MarshalIndent(MediaRecord) (plan A1 决议)
type FileInfoPrinter struct {
	out    io.Writer
	format string
	color  *Colorer
}

// NewFileInfoPrinter 构造一个 printer。
// format ∈ {"table","json"}。color 按 out 是否 TTY + NO_COLOR 自动探测。
//
// out 为 *os.File 时走 TTY 探测；为 *bytes.Buffer / 其他 io.Writer 时不上色
// （单元测试自然 NoColor，断言简单）。
func NewFileInfoPrinter(out io.Writer, format string) *FileInfoPrinter {
	useColor := writerIsTTY(out) && !NoColor()
	return &FileInfoPrinter{
		out:    out,
		format: strings.ToLower(strings.TrimSpace(format)),
		color:  NewColorer(useColor),
	}
}

// writerIsTTY 仅对 *os.File 走 IsTTY；其他 writer（如 bytes.Buffer）返回 false。
func writerIsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return IsTTY(f)
	}
	return false
}

// SectionSeparator 用于多文件 info 之间的视觉分隔。
// 按 stdout 当前状态判定上色；不在 TTY 时返回纯字符串。
func SectionSeparator() string {
	c := NewColorer(IsTTY(os.Stdout) && !NoColor())
	return c.Dim(strings.Repeat("─", 65))
}

// Print 把一个 MediaRecord 渲染到 out。
//
// table 模式：FILE / EXIF / GPS / AUDIO / VIDEO / ERRORS 6 个候选 section
// json 模式：json.MarshalIndent(record, "", "  ")
func (p *FileInfoPrinter) Print(record *media.MediaRecord) error {
	if p.format == "json" {
		return p.printJSON(record)
	}
	return p.printTable(record)
}

func (p *FileInfoPrinter) printJSON(record *media.MediaRecord) error {
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	return enc.Encode(record)
}

// kvRow 单个字段定义：label + 从 record 取值的函数；返回 "" 表示该字段空、不渲染
type kvRow struct {
	label string
	get   func(*media.MediaRecord) string
}

// section 一个 section 的定义：标题 + 字段顺序 + skip 条件
type section struct {
	name string
	rows []kvRow
	// skip 返回 true 时整个 section 不渲染（即使有部分字段非空）
	// 比如 GPS section 在 record.Exif==nil || !HasGPS 时整段跳过
	skip func(*media.MediaRecord) bool
}

func (p *FileInfoPrinter) printTable(record *media.MediaRecord) error {
	secs := buildSections()

	const labelWidth = 12
	first := true
	for _, sec := range secs {
		if sec.skip != nil && sec.skip(record) {
			continue
		}
		// 收集本 section 中非空字段
		var visible []kvRow
		for _, r := range sec.rows {
			if v := r.get(record); v != "" {
				visible = append(visible, r)
			}
		}
		if len(visible) == 0 {
			continue
		}
		if !first {
			fmt.Fprintln(p.out)
		}
		first = false

		fmt.Fprintln(p.out, p.color.SectionHeader(sec.name))
		for _, r := range visible {
			value := r.get(record)
			fmt.Fprintf(p.out, "  %s %s\n", p.color.Dim(padR(r.label, labelWidth)), value)
		}
	}

	return nil
}

// buildSections 定义所有 section 和字段顺序。
// 表驱动：新加字段只改这里，不动渲染逻辑。
func buildSections() []section {
	return []section{
		{
			name: "FILE",
			rows: []kvRow{
				{"路径", func(r *media.MediaRecord) string { return r.FilePath }},
				{"大小", func(r *media.MediaRecord) string {
					if r.FileSize <= 0 {
						return ""
					}
					return formatFileSize(r.FileSize)
				}},
				{"修改时间", func(r *media.MediaRecord) string {
					if r.ModTime.IsZero() {
						return ""
					}
					return r.ModTime.Format("2006-01-02 15:04:05")
				}},
				{"类型", typeLabel},
			},
		},
		{
			name: "EXIF",
			skip: func(r *media.MediaRecord) bool { return r.Exif == nil },
			rows: []kvRow{
				{"相机", func(r *media.MediaRecord) string {
					if r.Exif.CameraMake != "" && r.Exif.CameraModel != "" {
						return r.Exif.CameraMake + " " + r.Exif.CameraModel
					}
					return firstNonEmpty(r.Exif.CameraMake, r.Exif.CameraModel)
				}},
				{"镜头", func(r *media.MediaRecord) string {
					if r.Exif.LensMake != "" && r.Exif.LensModel != "" {
						return r.Exif.LensMake + " " + r.Exif.LensModel
					}
					return firstNonEmpty(r.Exif.LensMake, r.Exif.LensModel)
				}},
				{"拍摄时间", func(r *media.MediaRecord) string {
					if r.Exif.HasDateTime {
						return r.Exif.DateTimeOriginal.Format("2006-01-02 15:04:05")
					}
					return ""
				}},
				{"ISO", func(r *media.MediaRecord) string { return r.Exif.ISO }},
				{"光圈", func(r *media.MediaRecord) string { return r.Exif.Aperture }},
				{"快门", func(r *media.MediaRecord) string { return r.Exif.ShutterSpeed }},
				{"焦距", func(r *media.MediaRecord) string {
					if r.Exif.FocalLength35mm != "" && r.Exif.FocalLength != "" {
						return r.Exif.FocalLength + "（等效 " + r.Exif.FocalLength35mm + "）"
					}
					return firstNonEmpty(r.Exif.FocalLength, r.Exif.FocalLength35mm)
				}},
				{"尺寸", func(r *media.MediaRecord) string {
					if r.Exif.ImageWidth > 0 && r.Exif.ImageHeight > 0 {
						return fmt.Sprintf("%dx%d", r.Exif.ImageWidth, r.Exif.ImageHeight)
					}
					return ""
				}},
				{"曝光模式", func(r *media.MediaRecord) string { return r.Exif.ExposureMode }},
				{"曝光程序", func(r *media.MediaRecord) string { return r.Exif.ExposureProgram }},
				{"曝光补偿", func(r *media.MediaRecord) string { return r.Exif.ExposureCompensation }},
				{"白平衡", func(r *media.MediaRecord) string { return r.Exif.WhiteBalance }},
				{"测光模式", func(r *media.MediaRecord) string { return r.Exif.MeteringMode }},
				{"闪光灯", func(r *media.MediaRecord) string { return r.Exif.Flash }},
				{"色彩空间", func(r *media.MediaRecord) string { return r.Exif.ColorSpace }},
			},
		},
		{
			name: "GPS",
			skip: func(r *media.MediaRecord) bool { return r.Exif == nil || !r.Exif.GPS.HasGPS },
			rows: []kvRow{
				{"纬度", func(r *media.MediaRecord) string { return fmt.Sprintf("%.6f", r.Exif.GPS.Latitude) }},
				{"经度", func(r *media.MediaRecord) string { return fmt.Sprintf("%.6f", r.Exif.GPS.Longitude) }},
				{"海拔", func(r *media.MediaRecord) string {
					if r.Exif.GPS.Altitude != 0 {
						return fmt.Sprintf("%.1f m", r.Exif.GPS.Altitude)
					}
					return ""
				}},
				{"地点", func(r *media.MediaRecord) string {
					if r.Location == nil {
						return ""
					}
					parts := []string{}
					for _, s := range []string{r.Location.Country, r.Location.Province, r.Location.City} {
						if s != "" {
							parts = append(parts, s)
						}
					}
					return strings.Join(parts, " / ")
				}},
			},
		},
		{
			name: "AUDIO",
			skip: func(r *media.MediaRecord) bool { return r.Audio == nil },
			rows: []kvRow{
				{"编解码器", func(r *media.MediaRecord) string { return r.Audio.Codec }},
				{"比特率", func(r *media.MediaRecord) string {
					if r.Audio.Bitrate > 0 {
						return formatBitrate(r.Audio.Bitrate)
					}
					return ""
				}},
				{"采样率", func(r *media.MediaRecord) string {
					if r.Audio.SampleRate > 0 {
						return fmt.Sprintf("%d Hz", r.Audio.SampleRate)
					}
					return ""
				}},
				{"声道", func(r *media.MediaRecord) string {
					if r.Audio.ChannelLayout != "" {
						return r.Audio.ChannelLayout
					}
					if r.Audio.Channels > 0 {
						return fmt.Sprintf("%d", r.Audio.Channels)
					}
					return ""
				}},
				{"时长", func(r *media.MediaRecord) string {
					if r.Audio.Duration > 0 {
						return formatDuration(r.Audio.Duration)
					}
					return ""
				}},
				{"录制时间", func(r *media.MediaRecord) string {
					if r.Audio.HasRecordedTime {
						return r.Audio.RecordedTime.Format("2006-01-02 15:04:05")
					}
					return ""
				}},
			},
		},
		{
			name: "VIDEO",
			skip: func(r *media.MediaRecord) bool { return r.Video == nil },
			rows: []kvRow{
				{"编解码器", func(r *media.MediaRecord) string { return r.Video.Codec }},
				{"音轨编码", func(r *media.MediaRecord) string { return r.Video.AudioCodec }},
				{"分辨率", func(r *media.MediaRecord) string {
					if r.Video.Width > 0 && r.Video.Height > 0 {
						return fmt.Sprintf("%dx%d", r.Video.Width, r.Video.Height)
					}
					return ""
				}},
				{"比特率", func(r *media.MediaRecord) string {
					if r.Video.Bitrate > 0 {
						return formatBitrate(r.Video.Bitrate)
					}
					return ""
				}},
				{"帧率", func(r *media.MediaRecord) string { return r.Video.FrameRate }},
				{"时长", func(r *media.MediaRecord) string {
					if r.Video.Duration > 0 {
						return formatDuration(r.Video.Duration)
					}
					return ""
				}},
				{"创建时间", func(r *media.MediaRecord) string {
					if r.Video.HasDateTime {
						return r.Video.CreateTime.Format("2006-01-02 15:04:05")
					}
					return ""
				}},
			},
		},
		{
			name: "ERRORS",
			skip: func(r *media.MediaRecord) bool { return !hasErrors(r) },
			rows: []kvRow{
				// 单条 row，dynamic 列举 Attributes 里所有 *_error 键
				{"提取错误", func(r *media.MediaRecord) string {
					if r.Attributes == nil {
						return ""
					}
					keys := make([]string, 0)
					for k := range r.Attributes {
						if strings.HasSuffix(k, "_error") {
							keys = append(keys, k)
						}
					}
					sort.Strings(keys)
					if len(keys) == 0 {
						return ""
					}
					var sb strings.Builder
					for i, k := range keys {
						if i > 0 {
							sb.WriteString("\n               ")
						}
						sb.WriteString(k)
						sb.WriteString(": ")
						sb.WriteString(r.Attributes[k])
					}
					return sb.String()
				}},
				{"系统错误", func(r *media.MediaRecord) string {
					if r.Error != nil {
						return r.Error.Error()
					}
					return ""
				}},
			},
		},
	}
}

func typeLabel(r *media.MediaRecord) string {
	if r.Type == media.TypeUnknown {
		return "unknown (not a recognized media file)"
	}
	return r.Type.String()
}

func hasErrors(r *media.MediaRecord) bool {
	if r.Error != nil {
		return true
	}
	for k := range r.Attributes {
		if strings.HasSuffix(k, "_error") {
			return true
		}
	}
	return false
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// formatFileSize 人类可读：1.2 GB / 856.4 KB / 432 B
// 1024 进制；KB/MB/GB 用 2 位小数，B 用整数。
func formatFileSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	return fmt.Sprintf("%.2f %s", float64(n)/float64(div), suffix)
}

// formatBitrate 人类可读比特率：320 kbps / 5.2 Mbps
func formatBitrate(bps int64) string {
	if bps >= 1_000_000 {
		return fmt.Sprintf("%.2f Mbps", float64(bps)/1_000_000)
	}
	return fmt.Sprintf("%d kbps", bps/1000)
}

// formatDuration 秒数 → 人类可读：12.34s / 1m23s / 1h23m45s
func formatDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", seconds)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}
