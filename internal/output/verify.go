package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xunull/imfd/internal/media"
)

// VerifyPrinter 渲染 imfd verify 的输出。
//
// 格式：
//   table — 人类可读，section 风格仿 info（color helpers + ASCII，无 emoji）
//   json  — 包含 verdict / is_edited / signals 的结构化输出
//
// 不复用 FileInfoPrinter，因为 verify 输出聚焦「编辑判定」+ signals，
// 不需要 EXIF 全字段；JSON 结构也不同（加 verdict / signals）。
//
// c2paReport 是 JSON 里的 C2PA 子结构。
type c2paReport struct {
	Present   bool   `json:"present"`
	Generator string `json:"generator,omitempty"`
	Trust     string `json:"trust"` // 永远 "detection-only"
}

// VerifyReport 是 JSON 输出的固定结构，下游脚本可稳定消费。
//
// 两个独立维度：
//   - verdict / is_edited：编辑检测（original/camera-rendered/edited/unknown）
//   - ai_verdict / is_ai_generated：AI 生成检测（ai-generated/not-ai/unknown）
type VerifyReport struct {
	FilePath         string      `json:"file_path"`
	FileName         string      `json:"file_name"`
	FileSize         int64       `json:"file_size"`
	Type             string      `json:"type"`
	IsEdited         bool        `json:"is_edited"`
	Verdict          string      `json:"verdict"`     // 编辑检测
	IsAIGenerated    bool        `json:"is_ai_generated"`
	AIVerdict        string      `json:"ai_verdict"`  // AI 生成检测
	C2PA             *c2paReport `json:"c2pa,omitempty"`
	CameraMake       string      `json:"camera_make,omitempty"`
	CameraModel      string      `json:"camera_model,omitempty"`
	LensModel        string      `json:"lens_model,omitempty"`
	Software         string      `json:"software,omitempty"`
	DateTimeOriginal string      `json:"date_time_original,omitempty"`
	ModifyDate       string      `json:"modify_date,omitempty"`
	AISignals        []string    `json:"ai_signals"`
	Signals          []string    `json:"signals"` // 编辑信号
	Skipped          string      `json:"skipped,omitempty"`
}

// VerifyPrinter 是 verify 命令的输出渲染器。
type VerifyPrinter struct {
	out    io.Writer
	format string
	detail bool // --c2pa：展开 C2PA MANIFEST section（即便没 manifest 也显示 Present: no）
	color  *Colorer
}

// NewVerifyPrinter 构造 printer。format ∈ {"table","json"}。
// detail=true（--c2pa）时强制展开 C2PA MANIFEST section。
// out 为 *os.File 时走 TTY 探测；其他 writer 不上色（单元测试断言简单）。
func NewVerifyPrinter(out io.Writer, format string, detail bool) *VerifyPrinter {
	useColor := writerIsTTY(out) && !NoColor()
	return &VerifyPrinter{
		out:    out,
		format: strings.ToLower(strings.TrimSpace(format)),
		detail: detail,
		color:  NewColorer(useColor),
	}
}

// Print 把一个 record 的 verify 结果渲染到 out。
// 非图像 record 走 skip 路径（短输出，不渲染所有字段）。
func (p *VerifyPrinter) Print(record *media.MediaRecord) error {
	if p.format == "json" {
		return p.printJSON(record)
	}
	return p.printTable(record)
}

// PrintSeparator 多文件批量时插入分隔线。
func (p *VerifyPrinter) PrintSeparator() {
	if p.format == "json" {
		return // JSON 一行一对象，不需分隔
	}
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, p.color.Dim(strings.Repeat("─", 65)))
	fmt.Fprintln(p.out)
}

func (p *VerifyPrinter) printJSON(record *media.MediaRecord) error {
	report := buildReport(record)
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func (p *VerifyPrinter) printTable(record *media.MediaRecord) error {
	if record.Type != media.TypeImage {
		fmt.Fprintf(p.out, "%s  %s (%s)\n",
			p.color.Dim("SKIP:"),
			record.FilePath,
			p.color.Dim("not an image, verify currently supports image only"))
		return nil
	}

	report := buildReport(record)

	// File header
	fmt.Fprintln(p.out, p.color.SectionHeader("FILE"))
	p.row("Path", report.FilePath)
	p.row("Size", fmt.Sprintf("%s bytes", fmtInt64(report.FileSize)))
	p.row("Type", report.Type)

	// Capture info
	if record.Exif != nil {
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, p.color.SectionHeader("CAPTURE"))
		if report.DateTimeOriginal != "" {
			p.row("Captured", report.DateTimeOriginal)
		}
		if report.ModifyDate != "" {
			p.row("Modified", report.ModifyDate)
		}
		if report.CameraMake != "" || report.CameraModel != "" {
			p.row("Camera", strings.TrimSpace(report.CameraMake+" "+report.CameraModel))
		}
		if report.LensModel != "" {
			p.row("Lens", report.LensModel)
		}
		if report.Software != "" {
			p.row("Software", report.Software)
		}
	}

	// Verdict（两个独立维度：AI 生成 + 编辑）
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, p.color.SectionHeader("VERDICT"))
	p.row("AI", p.colorAIVerdict(report.AIVerdict))
	p.row("Edit", p.colorVerdict(report.Verdict))

	// C2PA MANIFEST（--c2pa 强制展开；否则仅在有 manifest 时显示）
	if p.detail || report.C2PA != nil {
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, p.color.SectionHeader("C2PA MANIFEST"))
		if report.C2PA != nil {
			p.row("Present", "yes")
			if report.C2PA.Generator != "" {
				p.row("Generator", report.C2PA.Generator)
			}
			p.row("Trust", p.color.Dim("detection-only (signature NOT verified)"))
		} else {
			p.row("Present", "no")
		}
	}

	// AI signals
	if len(report.AISignals) > 0 {
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, p.color.SectionHeader("AI SIGNALS"))
		for _, s := range report.AISignals {
			fmt.Fprintf(p.out, "  %s\n", s)
		}
	}

	// Edit signals
	if len(report.Signals) > 0 {
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, p.color.SectionHeader("EDIT SIGNALS"))
		for _, s := range report.Signals {
			fmt.Fprintf(p.out, "  %s\n", s)
		}
	}

	return nil
}

func (p *VerifyPrinter) row(label, value string) {
	fmt.Fprintf(p.out, "  %s %s\n", p.color.Dim(padRight(label+":", 11)), value)
}

func (p *VerifyPrinter) colorVerdict(v string) string {
	switch v {
	case media.VerdictOriginal:
		return p.color.Green(v)
	case media.VerdictCameraRendered:
		return p.color.Cyan(v)
	case media.VerdictEdited:
		return p.color.Yellow(v)
	default:
		return p.color.Dim(v)
	}
}

func (p *VerifyPrinter) colorAIVerdict(v string) string {
	switch v {
	case media.AIVerdictGenerated:
		return p.color.Yellow(v) // 注意：AI 生成
	case media.AIVerdictNotAI:
		return p.color.Green(v)
	default:
		return p.color.Dim(v)
	}
}

// buildReport 把 record 转成 VerifyReport 结构（JSON / table 共用）。
func buildReport(record *media.MediaRecord) VerifyReport {
	r := VerifyReport{
		FilePath:      record.FilePath,
		FileName:      record.FileName,
		FileSize:      record.FileSize,
		Type:          record.Type.String(),
		Verdict:       media.Verdict(record),
		IsEdited:      media.IsEdited(record),
		AIVerdict:     media.AIVerdict(record),
		IsAIGenerated: media.IsAIGenerated(record),
		AISignals:     media.AISignals(record),
		Signals:       media.EditSignals(record),
	}

	if record.Type != media.TypeImage {
		r.Skipped = "not an image"
	}

	if record.Exif != nil {
		r.CameraMake = record.Exif.CameraMake
		r.CameraModel = record.Exif.CameraModel
		r.LensModel = record.Exif.LensModel
		r.Software = record.Exif.Software
		if record.Exif.HasDateTime {
			r.DateTimeOriginal = record.Exif.DateTimeOriginal.Format("2006-01-02 15:04:05")
		}
		if record.Exif.HasModifyDate {
			r.ModifyDate = record.Exif.ModifyDate.Format("2006-01-02 15:04:05")
		}
		if record.Exif.C2PA != nil {
			r.C2PA = &c2paReport{
				Present:   record.Exif.C2PA.Present,
				Generator: record.Exif.C2PA.Generator,
				Trust:     "detection-only",
			}
		}
	}

	return r
}

// padRight pads string to width (no truncation).
func padRight(s string, width int) string {
	for len(s) < width {
		s += " "
	}
	return s
}

// fmtInt64 千分位逗号分隔（仿 cmd/cache.go 的 fmtCount，避免循环 import）。
func fmtInt64(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
