// Package query 实现 imfd list 命令的 DSL 评估器。
//
// 3 个核心步骤：
//   1. BuildFilter(flags, userFilter) → (expr string, needles []string)
//      把 CLI flag 翻译成 expr-lang 表达式字符串 + needle slice。
//   2. NewEvaluator(expr, needles) → Evaluator
//      把表达式 compile 一次，cache program。
//   3. (e *Evaluator) Match(record) → bool
//      每个 record 调用 BuildEnv → expr.Run → bool。
//
// 设计原则（per plan-eng-review Q1=C）：
// flag value 通过 env 变量注入（needle1/needle2/...），不字符串拼接。
//
// nil-semantics（per plan-eng-review P4）：
// expr 的 compile-time type checker 看到 nil 会 mismatch；
// 改用「typed zero values」实现：缺字段 → 0 / "" / false。
// 等价语义：iso > 800 在缺字段时 0 > 800 = false，province == '云南' 在缺字段时 "" == "云南" = false。
// EXIF ISO 0 / aperture 0 / shutter 0 在物理意义上不存在，零冲突。
package query

import (
	"github.com/xunull/imfd/internal/media"
)

// AllowedFields 是 env 暴露给 DSL 的字段名清单。
//
// 顺序无关；字段名扁平 snake_case（design doc P3）。
var AllowedFields = []string{
	"file_path",
	"file_name",
	"file_size",
	"type",
	"camera_make",
	"camera_model",
	"lens_make",
	"lens_model",
	"iso",
	"aperture_value",
	"shutter_seconds",
	"focal_length_mm",
	"image_width",
	"image_height",
	"province",
	"city",
	"country",
	"capture_year",
	"capture_hour",
	"audio_codec",
	"audio_bitrate",
	"audio_sample_rate",
	"video_codec",
	"video_width",
	"video_height",
	"device_type",
	"scene_starry_sky",
	"is_edited",
	"is_ai_generated",
}

// envTypes 是 env 的类型骨架，给 expr.Env 做 compile-time type check。
//
// 缺字段在 BuildEnv 里填这些 zero values；这是 nil-safe 的实现方式
// （per plan-eng-review P4：用 typed zero 替代 nil）。
func envTypes() map[string]any {
	return map[string]any{
		"file_path":         "",
		"file_name":         "",
		"file_size":         int64(0),
		"type":              "",
		"camera_make":       "",
		"camera_model":      "",
		"lens_make":         "",
		"lens_model":        "",
		"iso":               0,
		"aperture_value":    0.0,
		"shutter_seconds":   0.0,
		"focal_length_mm":   0.0,
		"image_width":       0,
		"image_height":      0,
		"province":          "",
		"city":              "",
		"country":           "",
		"capture_year":      0,
		"capture_hour":      0,
		"audio_codec":       "",
		"audio_bitrate":     int64(0),
		"audio_sample_rate": 0,
		"video_codec":       "",
		"video_width":       0,
		"video_height":      0,
		"device_type":       "",
		"scene_starry_sky":  false,
		"is_edited":         false,
		"is_ai_generated":   false,
	}
}

// BuildEnv 把一条 MediaRecord 摊平成 expr 友好的 map[string]any。
//
// 缺失字段填类型化的 zero value（nil-safe 实现，见 envTypes 注释）。
func BuildEnv(record *media.MediaRecord, needles []string) map[string]any {
	env := envTypes()

	if record != nil {
		env["file_path"] = record.FilePath
		env["file_name"] = record.FileName
		env["file_size"] = record.FileSize
		env["type"] = record.Type.String()

		if record.Exif != nil {
			env["camera_make"] = record.Exif.CameraMake
			env["camera_model"] = record.Exif.CameraModel
			env["lens_make"] = record.Exif.LensMake
			env["lens_model"] = record.Exif.LensModel
			if v, ok := media.ParseISO(record.Exif.ISO); ok {
				env["iso"] = v
			}
			if v, ok := media.ParseAperture(record.Exif.Aperture); ok {
				env["aperture_value"] = v
			}
			if v, ok := media.ParseShutter(record.Exif.ShutterSpeed); ok {
				env["shutter_seconds"] = v
			}
			if v, ok := media.ParseFocal(record.Exif.FocalLength); ok {
				env["focal_length_mm"] = v
			}
			if record.Exif.ImageWidth > 0 {
				env["image_width"] = record.Exif.ImageWidth
			}
			if record.Exif.ImageHeight > 0 {
				env["image_height"] = record.Exif.ImageHeight
			}
			if record.Exif.HasDateTime {
				env["capture_year"] = record.Exif.DateTimeOriginal.Year()
				env["capture_hour"] = record.Exif.DateTimeOriginal.Hour()
			}

			// derived: device_type / scene_starry_sky / is_edited
			if dt := media.DeviceType(record.Exif.CameraMake); dt != media.DeviceTypeUnknown {
				env["device_type"] = dt
			}
			env["scene_starry_sky"] = media.IsStarrySky(record)
			env["is_edited"] = media.IsEdited(record)
			env["is_ai_generated"] = media.IsAIGenerated(record)
		}

		if record.Location != nil {
			env["province"] = record.Location.Province
			env["city"] = record.Location.City
			env["country"] = record.Location.Country
		}

		if record.Audio != nil {
			env["audio_codec"] = record.Audio.Codec
			if record.Audio.Bitrate > 0 {
				env["audio_bitrate"] = record.Audio.Bitrate
			}
			if record.Audio.SampleRate > 0 {
				env["audio_sample_rate"] = record.Audio.SampleRate
			}
		}

		if record.Video != nil {
			env["video_codec"] = record.Video.Codec
			if record.Video.Width > 0 {
				env["video_width"] = record.Video.Width
			}
			if record.Video.Height > 0 {
				env["video_height"] = record.Video.Height
			}
		}
	}

	// flag value 通过 env 注入（plan Q1=C）：needle1, needle2, ...
	for i, n := range needles {
		env[needleVar(i)] = n
	}

	return env
}

// needleVar 给定 0-based index 返回 needle 变量名 needle1 / needle2 / ...
func needleVar(i int) string {
	return "needle" + itoa(i+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
