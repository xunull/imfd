package media

// scene_starry_sky derived 字段。
//
// v1 唯一 scene 启发式（其他 scene per plan-eng-review LIST-DIM-1 延后）。
//
// 规则（per design doc P3）：
//   true ⇔ iso > 1600
//          AND shutter_seconds > 10
//          AND capture_hour ∈ [22,4]（22:00-04:59）
//
// nil-safe：record 缺任何依赖字段（Exif=nil、ISO 解析不出、HasCaptureTime=false）→ false。
//
// 拍摄小时使用 wall-clock（无 TZ 转换；EXIF DateTimeOriginal 通常存的就是
// 拍摄地本地时间）。

// IsStarrySky 判定一条 record 是否启发式标记为星空照。
func IsStarrySky(r *MediaRecord) bool {
	if r == nil || r.Exif == nil {
		return false
	}

	iso, ok := ParseISO(r.Exif.ISO)
	if !ok || iso <= 1600 {
		return false
	}

	shutter, ok := ParseShutter(r.Exif.ShutterSpeed)
	if !ok || shutter <= 10 {
		return false
	}

	if !r.Exif.HasDateTime {
		return false
	}
	hour := r.Exif.DateTimeOriginal.Hour()
	if !(hour >= 22 || hour <= 4) {
		return false
	}

	return true
}
