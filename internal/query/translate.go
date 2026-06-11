package query

import (
	"fmt"
	"strconv"
	"strings"
)

// ListFlags 是 cmd/list.go 收集的所有 filter 相关 flag 值。
// 翻译层把它转成 expr 字符串 + needle slice。
//
// 设计规则（per plan-eng-review）：
//   - 多同名 flag (CameraMakes []string) = OR
//   - 多不同 flag = AND
//   - + UserFilter = AND
//   - 字符串 flag 经 needle env 注入（避免字符串拼接的 escape 风险）
type ListFlags struct {
	Type         string   // image / video / audio / all
	CameraMakes  []string // --camera-make 可重复 → OR
	CameraModels []string // --camera-model 可重复 → OR
	LensModels   []string // --lens 可重复 → OR
	DeviceType   string   // phone / camera
	Provinces    []string // --province 可重复 → OR
	Cities       []string // --city 可重复 → OR
	Scene        string   // v1 仅 starry_sky
	ISO          string   // "N" / ">N" / "<N" / ">=N" / "<=N" / "N-M"
	Year         string   // 同上
	// codec 三个 flag：
	// --codec 同时匹配 audio_codec/video_codec（"flac" / "h264" 不需要管它是音是视频）
	// --audio-codec / --video-codec 精确二选一
	Codecs      []string
	AudioCodecs []string
	VideoCodecs []string
}

// BuildFilter 把 ListFlags + UserFilter 翻译成单一 expr 字符串 + needles。
//
// 输出例：
//   flags: --province 云南 --device phone
//   user filter: "iso > 800"
//   →
//   expr  = "(lower(province) contains lower(needle1)) and (device_type == \"phone\") and (iso > 800)"
//   needles = []string{"云南"}
//
// 当所有 flag 和 userFilter 都空时返回 ("true", nil)：匹配所有 record。
func BuildFilter(f ListFlags, userFilter string) (string, []string) {
	var parts []string
	var needles []string

	// helper：把字符串值加进 needles，返回它的变量名 (needle1, needle2, ...)
	addNeedle := func(v string) string {
		needles = append(needles, v)
		return needleVar(len(needles) - 1)
	}

	// type
	if f.Type != "" && f.Type != "all" {
		parts = append(parts, fmt.Sprintf("(type == %q)", f.Type))
	}

	// camera_make（多 = OR；substring case-insensitive；双边 lower()）
	if len(f.CameraMakes) > 0 {
		parts = append(parts, orSubstring("camera_make", f.CameraMakes, addNeedle))
	}
	if len(f.CameraModels) > 0 {
		parts = append(parts, orSubstring("camera_model", f.CameraModels, addNeedle))
	}
	if len(f.LensModels) > 0 {
		parts = append(parts, orSubstring("lens_model", f.LensModels, addNeedle))
	}
	if len(f.Provinces) > 0 {
		parts = append(parts, orSubstring("province", f.Provinces, addNeedle))
	}
	if len(f.Cities) > 0 {
		parts = append(parts, orSubstring("city", f.Cities, addNeedle))
	}

	// codec：--codec 在 audio_codec / video_codec 任一命中
	if len(f.Codecs) > 0 {
		parts = append(parts, orSubstringMultiField([]string{"audio_codec", "video_codec"}, f.Codecs, addNeedle))
	}
	if len(f.AudioCodecs) > 0 {
		parts = append(parts, orSubstring("audio_codec", f.AudioCodecs, addNeedle))
	}
	if len(f.VideoCodecs) > 0 {
		parts = append(parts, orSubstring("video_codec", f.VideoCodecs, addNeedle))
	}

	// device_type / scene_starry_sky：精确匹配
	if f.DeviceType != "" {
		parts = append(parts, fmt.Sprintf("(device_type == %q)", f.DeviceType))
	}
	if f.Scene == "starry_sky" {
		parts = append(parts, "(scene_starry_sky == true)")
	}

	// numeric range syntax
	if f.ISO != "" {
		if frag, ok := buildRangeExpr("iso", f.ISO); ok {
			parts = append(parts, frag)
		}
	}
	if f.Year != "" {
		if frag, ok := buildRangeExpr("capture_year", f.Year); ok {
			parts = append(parts, frag)
		}
	}

	// userFilter 原样加入
	if strings.TrimSpace(userFilter) != "" {
		parts = append(parts, "("+userFilter+")")
	}

	if len(parts) == 0 {
		return "true", nil
	}
	return strings.Join(parts, " and "), needles
}

// orSubstringMultiField 生成跨多字段的 substring OR：
// fields=[audio_codec, video_codec], vals=[flac] →
//   ( lower(audio_codec) contains lower(needle1) or lower(video_codec) contains lower(needle1) )
// 用于 --codec flac 这种"不关心是音频还是视频"的场景
func orSubstringMultiField(fields []string, vals []string, addNeedle func(string) string) string {
	if len(fields) == 0 || len(vals) == 0 {
		return ""
	}
	var ors []string
	for _, v := range vals {
		nv := addNeedle(v)
		for _, field := range fields {
			ors = append(ors, fmt.Sprintf("(lower(%s) contains lower(%s))", field, nv))
		}
	}
	return "(" + strings.Join(ors, " or ") + ")"
}

// orSubstring 生成 (lower(field) contains lower(needleN1)) or (lower(field) contains lower(needleN2))...
//
// 为什么双边 lower()：EXIF make 大小写各种各样 (NIKON CORPORATION / Nikon / nikon)，
// needle 也可能各种大小写。两边都 lower 后 contains 才行为可预测。
func orSubstring(field string, vals []string, addNeedle func(string) string) string {
	if len(vals) == 0 {
		return ""
	}
	var ors []string
	for _, v := range vals {
		nv := addNeedle(v)
		ors = append(ors, fmt.Sprintf("(lower(%s) contains lower(%s))", field, nv))
	}
	return "(" + strings.Join(ors, " or ") + ")"
}

// buildRangeExpr 解析 numeric range 4 种语法：
//   "800"         → (field == 800)
//   ">800"        → (field > 800)
//   "<800"        → (field < 800)
//   ">=800"       → (field >= 800)
//   "<=800"       → (field <= 800)
//   "800-1600"    → (field >= 800 and field <= 1600)
//
// 解析失败返回 ("", false)。
func buildRangeExpr(field, spec string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false
	}

	// >=  /  <=
	if strings.HasPrefix(spec, ">=") {
		if n, ok := tryInt(spec[2:]); ok {
			return fmt.Sprintf("(%s >= %d)", field, n), true
		}
		return "", false
	}
	if strings.HasPrefix(spec, "<=") {
		if n, ok := tryInt(spec[2:]); ok {
			return fmt.Sprintf("(%s <= %d)", field, n), true
		}
		return "", false
	}
	// >  /  <
	if strings.HasPrefix(spec, ">") {
		if n, ok := tryInt(spec[1:]); ok {
			return fmt.Sprintf("(%s > %d)", field, n), true
		}
		return "", false
	}
	if strings.HasPrefix(spec, "<") {
		if n, ok := tryInt(spec[1:]); ok {
			return fmt.Sprintf("(%s < %d)", field, n), true
		}
		return "", false
	}
	// "A-B"
	if i := strings.Index(spec, "-"); i > 0 {
		lo, ok1 := tryInt(spec[:i])
		hi, ok2 := tryInt(spec[i+1:])
		if ok1 && ok2 {
			return fmt.Sprintf("(%s >= %d and %s <= %d)", field, lo, field, hi), true
		}
		return "", false
	}
	// "N" bare
	if n, ok := tryInt(spec); ok {
		return fmt.Sprintf("(%s == %d)", field, n), true
	}
	return "", false
}

func tryInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
