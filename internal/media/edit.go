package media

import (
	"strings"
	"time"
)

// 编辑检测的常量与规则。
//
// 设计目标（design doc Premises P1-P5 + Eng Review A1-A3）：
//   - 信号源 = EXIF Software 字段 + ModifyDate vs DateTimeOriginal 时间差
//   - nil-safe：缺字段不 panic、返回 false
//   - 60 秒容忍窗口防 RAW→JPEG 转换误判
//   - 4 级 verdict（original / camera-rendered / edited / unknown）；
//     IsEdited 布尔接口给 list/view filter，Verdict 字符串给 verify 命令展示

// Verdict 字符串常量。
const (
	VerdictOriginal       = "original"
	VerdictCameraRendered = "camera-rendered"
	VerdictEdited         = "edited"
	VerdictUnknown        = "unknown"
)

// editorKeywords 是 Software 字段命中即视为「编辑过」的关键字白名单。
// 全部小写比对；Software 字段也 lower 后做 substring 匹配。
//
// 这个列表故意保守：宁可漏检（false negative）也不要把相机直出误判成编辑。
// 用户在 office-hours assignment 已验证主流分布；遇到新工具加进来即可。
var editorKeywords = []string{
	"lightroom",
	"photoshop",
	"capture one",
	"luminar",
	"affinity",
	"pixelmator",
	"preview",       // macOS Preview.app 编辑后会写
	"photos",        // macOS / iOS Photos.app in-app 调整
	"darktable",
	"rawtherapee",
	"on1",
	"dxo",
	"snapseed",
	"vsco",
	"gimp",
}

// cameraSoftwareKeywords 是相机内置软件写 Software 字段时常见的关键字。
// 命中这些表示是相机自己渲染（如富士菲林模拟、Sony Imaging Edge），verdict = camera-rendered。
// 必须与 editorKeywords 严格分开避免假阳：「Sony Imaging Edge」不应算 edited。
var cameraSoftwareKeywords = []string{
	"imaging edge",     // Sony
	"digital photo professional", // Canon DPP
	"raw file converter", // Fujifilm
	"raw converter",
	"hdr+",             // Google Pixel
	"camera",           // 通用：相机 firmware 写 "EOS Camera Firmware" 等
	"firmware",
	"ver.",             // "ver.1.0" 这种相机 firmware 标识
}

// rawConversionToleranceSeconds 是 ModifyDate 与 DateTimeOriginal 之间允许的最大偏差秒数，
// 用于忽略相机内 RAW→JPEG 转换的合理时差。超过这个窗口才认为是真正的后期编辑。
const rawConversionToleranceSeconds = 60

// IsEdited 判定一条 record 是否经过后期编辑。
//
// 信号源：
//   1. Software 字段命中 editorKeywords 任一关键字 → true
//   2. ModifyDate 比 DateTimeOriginal 晚 > 60s → true
//
// 任一不命中返回 false。nil-safe：record / record.Exif 为 nil → false。
// 相机品牌写的 Software（camera-rendered）不算 edited。
func IsEdited(r *MediaRecord) bool {
	if r == nil || r.Exif == nil {
		return false
	}

	// 信号 1：Software 字段
	switch classifySoftware(r.Exif.Software) {
	case softwareEditor:
		return true
	case softwareCamera:
		// 明确是相机渲染，跳过 ModifyDate 比较（相机生成 JPEG 时 ModifyDate
		// 经常合理地晚于 DateTimeOriginal，不应误判）
		return false
	}

	// 信号 2：ModifyDate 比 DateTimeOriginal 晚 > 60s
	if r.Exif.HasModifyDate && r.Exif.HasDateTime {
		diff := r.Exif.ModifyDate.Sub(r.Exif.DateTimeOriginal)
		if diff > time.Duration(rawConversionToleranceSeconds)*time.Second {
			return true
		}
	}

	return false
}

// Verdict 返回 4 级编辑判定字符串（verify 命令展示用）。
func Verdict(r *MediaRecord) string {
	if r == nil || r.Exif == nil {
		return VerdictUnknown
	}

	switch classifySoftware(r.Exif.Software) {
	case softwareEditor:
		return VerdictEdited
	case softwareCamera:
		return VerdictCameraRendered
	}

	// Software 为空 / 不归类时，看 ModifyDate
	if r.Exif.HasModifyDate && r.Exif.HasDateTime {
		diff := r.Exif.ModifyDate.Sub(r.Exif.DateTimeOriginal)
		if diff > time.Duration(rawConversionToleranceSeconds)*time.Second {
			return VerdictEdited
		}
		// 有完整时间字段且未超窗口 → original
		return VerdictOriginal
	}

	// Software 为空 + 没有完整时间字段 → 信号不足
	if r.Exif.Software == "" && !r.Exif.HasModifyDate {
		// 这种情况典型于相机直出 JPEG（没写 Software，没写 ModifyDate）
		// 视为 original 而非 unknown，避免大量原片被标 unknown
		if r.Exif.HasDateTime {
			return VerdictOriginal
		}
	}

	return VerdictUnknown
}

// EditSignals 返回 verify 命令展示用的「为什么这么判」的信号列表。
// 每条信号是「✓ <原因>」或「✗ <原因>」，前者代表命中、后者代表未命中但参与判定。
func EditSignals(r *MediaRecord) []string {
	if r == nil || r.Exif == nil {
		return []string{"✗ EXIF 数据缺失"}
	}

	var signals []string
	switch classifySoftware(r.Exif.Software) {
	case softwareEditor:
		signals = append(signals, "✓ Software 字段命中编辑器关键字: "+r.Exif.Software)
	case softwareCamera:
		signals = append(signals, "✓ Software 字段为相机内置软件: "+r.Exif.Software)
	case softwareUnknown:
		if r.Exif.Software != "" {
			signals = append(signals, "? Software 字段未归类: "+r.Exif.Software)
		} else {
			signals = append(signals, "✗ Software 字段为空（典型相机直出）")
		}
	}

	if r.Exif.HasModifyDate && r.Exif.HasDateTime {
		diff := r.Exif.ModifyDate.Sub(r.Exif.DateTimeOriginal)
		switch {
		case diff > time.Duration(rawConversionToleranceSeconds)*time.Second:
			signals = append(signals, "✓ ModifyDate 比 DateTimeOriginal 晚 "+formatDuration(diff))
		case diff < 0:
			signals = append(signals, "✗ ModifyDate 早于 DateTimeOriginal（异常）")
		default:
			signals = append(signals, "✗ ModifyDate 与 DateTimeOriginal 相差 ≤60s（相机 RAW→JPEG 容忍窗口）")
		}
	} else if !r.Exif.HasModifyDate {
		signals = append(signals, "✗ ModifyDate 字段缺失")
	}

	signals = append(signals, "✗ XMP edit history 未深度解析 (v1)")
	return signals
}

// softwareClass 是 Software 字段的分类。
type softwareClass int

const (
	softwareUnknown softwareClass = iota
	softwareEditor
	softwareCamera
)

// classifySoftware 把 Software 字段归类为 editor / camera / unknown。
// 优先匹配 cameraSoftwareKeywords —— 「Sony Imaging Edge」不应被 editor 关键字误捕。
func classifySoftware(s string) softwareClass {
	if s == "" {
		return softwareUnknown
	}
	low := strings.ToLower(s)

	for _, k := range cameraSoftwareKeywords {
		if strings.Contains(low, k) {
			return softwareCamera
		}
	}
	for _, k := range editorKeywords {
		if strings.Contains(low, k) {
			return softwareEditor
		}
	}
	return softwareUnknown
}

// formatDuration 把 time.Duration 格式化成人类可读字符串（用于 EditSignals 输出）。
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Truncate(time.Second).String()
	}
	if d < time.Hour {
		return d.Truncate(time.Second).String()
	}
	if d < 24*time.Hour {
		return d.Truncate(time.Minute).String()
	}
	days := int(d.Hours() / 24)
	return formatInt(days) + " days"
}

// formatInt 简单 int → string（避免引入 strconv 在 hot path）。
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
