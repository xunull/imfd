package extract

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/xunull/imfd/internal/c2pa"
	"github.com/xunull/imfd/internal/media"
)

// headReadSize 是从图像文件头读取的字节数。
// EXIF (App1) 几乎总在文件最前；C2PA (App11) 推荐放靠前。64 KB 覆盖绝大多数。
// 只读头部而非整文件：避免 50 MB DNG × 并发 worker 的内存峰值（plan-eng-review A2）。
// 超过 64 KB 的 C2PA manifest 会 degrade（DetectJPEG 拿截断字节，可能仅 Present 无 Generator，
// 或退回 keyword 信号）。
const headReadSize = 64 * 1024

// ExtractImageExif 从图像文件头提取 EXIF + C2PA + PNG 文本信号。
//
// 关键设计（plan-eng-review）：
//   - 只读前 headReadSize 字节，goexif 和 c2pa 共享同一份 bytes（一次 IO）
//   - goexif 解析失败不再是 error —— PNG screenshot / 很多 AI 图没有标准 EXIF，
//     但仍可能有 C2PA manifest 或 PNG 文本信号。文件可读就返回非 nil info。
//   - 只有文件 IO 失败才返回 error（此时 record.Exif 保持 nil）。
func ExtractImageExif(filePath string) (*media.ExifInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	head, err := io.ReadAll(io.LimitReader(f, headReadSize))
	if err != nil {
		return nil, fmt.Errorf("读取文件头失败: %w", err)
	}

	info := &media.ExifInfo{}

	// EXIF 解析（best-effort；失败不影响 C2PA/PNG 信号提取）
	if x, derr := exif.Decode(bytes.NewReader(head)); derr == nil {
		populateExif(info, x)
	}

	// C2PA manifest + PNG 文本信号检测（detection-only）
	res := c2pa.Detect(head)
	if res.Manifest != nil {
		info.C2PA = &media.C2PAInfo{
			Present:   res.Manifest.Present,
			Generator: res.Manifest.Generator,
		}
	}
	for _, e := range res.PNGText {
		info.PNGText = append(info.PNGText, media.PNGTextEntry{Key: e.Key, Value: e.Value})
	}

	return info, nil
}

// populateExif 把 goexif 解析结果填进 ExifInfo（从 ExtractImageExif 拆出便于阅读）。
func populateExif(info *media.ExifInfo, x *exif.Exif) {
	info.CameraMake = getTagString(x, exif.Make)
	info.CameraModel = getTagString(x, exif.Model)
	info.LensModel = getTagString(x, exif.LensModel)

	info.ISO = getTagString(x, exif.ISOSpeedRatings)

	if aperture := getTagRational(x, exif.FNumber); aperture != "" {
		info.Aperture = "f/" + aperture
	}

	info.ShutterSpeed = formatShutterSpeed(x)
	info.ExposureTime = getTagString(x, exif.ExposureTime)

	if fl := getTagRational(x, exif.FocalLength); fl != "" {
		info.FocalLength = fl + "mm"
	}

	info.FocalLength35mm = getTagString(x, exif.FocalLengthIn35mmFilm)
	if info.FocalLength35mm != "" {
		info.FocalLength35mm += "mm"
	}

	info.WhiteBalance = decodeWhiteBalance(getTagInt(x, exif.WhiteBalance))
	info.ExposureCompensation = getTagRational(x, exif.ExposureBiasValue)
	info.ExposureMode = decodeExposureMode(getTagInt(x, exif.ExposureMode))
	info.ExposureProgram = decodeExposureProgram(getTagInt(x, exif.ExposureProgram))
	info.MeteringMode = decodeMeteringMode(getTagInt(x, exif.MeteringMode))
	info.Flash = decodeFlash(getTagInt(x, exif.Flash))
	info.ColorSpace = getTagString(x, exif.ColorSpace)

	// Software 字段：编辑器（Lightroom / Photoshop）会写自己的名字。
	// 相机直出通常为空；少数相机内置软件（"Sony Imaging Edge"）也会写——后续 IsEdited 区分。
	info.Software = getTagString(x, exif.Software)

	if w := getTagInt(x, exif.PixelXDimension); w > 0 {
		info.ImageWidth = w
	}
	if h := getTagInt(x, exif.PixelYDimension); h > 0 {
		info.ImageHeight = h
	}

	// DateTimeOriginal (0x9003)：拍摄时间。
	// 注意：x.DateTime() 在 DateTimeOriginal 缺失时会 fallback 到 DateTime (= ModifyDate)，
	// 这会让 IsEdited 的「ModifyDate > DateTimeOriginal + 60s」变成 self-compare。
	// 此处严格只读 DateTimeOriginal tag，确保语义。
	if dt, ok := getTagDateTime(x, exif.DateTimeOriginal); ok {
		info.DateTimeOriginal = dt
		info.HasDateTime = true
	}

	// DateTime (0x0132) = ModifyDate：文件最后修改时间（编辑器写入）。
	// 与 DateTimeOriginal 严格独立读取，给 IsEdited 比较用。
	if dt, ok := getTagDateTime(x, exif.DateTime); ok {
		info.ModifyDate = dt
		info.HasModifyDate = true
	}

	if lat, lon, err := x.LatLong(); err == nil {
		info.GPS = media.GPSInfo{
			Latitude:  lat,
			Longitude: lon,
			HasGPS:    true,
		}
	}
}

func getTagString(x *exif.Exif, tag exif.FieldName) string {
	t, err := x.Get(tag)
	if err != nil {
		return ""
	}
	if t.Format() == tiff.StringVal {
		s, err := t.StringVal()
		if err != nil {
			return ""
		}
		return s
	}
	return t.String()
}

func getTagRational(x *exif.Exif, tag exif.FieldName) string {
	t, err := x.Get(tag)
	if err != nil {
		return ""
	}
	numer, denom, err := t.Rat2(0)
	if err != nil {
		return t.String()
	}
	if denom == 0 {
		return fmt.Sprintf("%d", numer)
	}
	val := float64(numer) / float64(denom)
	if val == math.Floor(val) {
		return fmt.Sprintf("%.0f", val)
	}
	return fmt.Sprintf("%.1f", val)
}

func getTagInt(x *exif.Exif, tag exif.FieldName) int {
	t, err := x.Get(tag)
	if err != nil {
		return -1
	}
	v, err := t.Int(0)
	if err != nil {
		return -1
	}
	return v
}

// getTagDateTime 读指定 EXIF tag 并解析为 time.Time。
// 不像 goexif 的 x.DateTime()，本函数严格只读指定 tag，不做 fallback。
// 用于区分 DateTimeOriginal（拍摄时间）和 DateTime（ModifyDate / 编辑时间）。
func getTagDateTime(x *exif.Exif, tag exif.FieldName) (time.Time, bool) {
	t, err := x.Get(tag)
	if err != nil {
		return time.Time{}, false
	}
	s, err := t.StringVal()
	if err != nil {
		return time.Time{}, false
	}
	dt, err := parseExifDateTime(s)
	if err != nil {
		return time.Time{}, false
	}
	return dt, true
}

func formatShutterSpeed(x *exif.Exif) string {
	t, err := x.Get(exif.ExposureTime)
	if err != nil {
		return ""
	}
	numer, denom, err := t.Rat2(0)
	if err != nil {
		return t.String()
	}
	if denom == 0 {
		return fmt.Sprintf("%ds", numer)
	}
	if numer == 1 {
		return fmt.Sprintf("1/%ds", denom)
	}
	val := float64(numer) / float64(denom)
	if val >= 1 {
		return fmt.Sprintf("%.1fs", val)
	}
	reciprocal := int64(math.Round(float64(denom) / float64(numer)))
	return fmt.Sprintf("1/%ds", reciprocal)
}

func decodeWhiteBalance(val int) string {
	switch val {
	case 0:
		return "Auto"
	case 1:
		return "Manual"
	default:
		return ""
	}
}

func decodeExposureMode(val int) string {
	switch val {
	case 0:
		return "Auto"
	case 1:
		return "Manual"
	case 2:
		return "Auto bracket"
	default:
		return ""
	}
}

func decodeExposureProgram(val int) string {
	switch val {
	case 0:
		return "Not defined"
	case 1:
		return "Manual"
	case 2:
		return "Normal program"
	case 3:
		return "Aperture priority"
	case 4:
		return "Shutter priority"
	case 5:
		return "Creative program"
	case 6:
		return "Action program"
	case 7:
		return "Portrait mode"
	case 8:
		return "Landscape mode"
	default:
		return ""
	}
}

func decodeMeteringMode(val int) string {
	switch val {
	case 0:
		return "Unknown"
	case 1:
		return "Average"
	case 2:
		return "CenterWeightedAverage"
	case 3:
		return "Spot"
	case 4:
		return "MultiSpot"
	case 5:
		return "Pattern"
	case 6:
		return "Partial"
	case 255:
		return "Other"
	default:
		return ""
	}
}

func decodeFlash(val int) string {
	if val < 0 {
		return ""
	}
	fired := val & 1
	if fired == 1 {
		return "Fired"
	}
	return "Not fired"
}

// BuildImageRecord 从图像文件构建 MediaRecord
func BuildImageRecord(filePath string, fileInfo os.FileInfo) *media.MediaRecord {
	record := &media.MediaRecord{
		FilePath:   filePath,
		FileName:   fileInfo.Name(),
		FileSize:   fileInfo.Size(),
		ModTime:    fileInfo.ModTime(),
		Type:       media.TypeImage,
		Attributes: make(map[string]string),
	}

	exifInfo, err := ExtractImageExif(filePath)
	if err != nil {
		record.Attributes["exif_error"] = err.Error()
		return record
	}

	record.Exif = exifInfo

	if exifInfo.HasDateTime {
		record.CaptureTime = exifInfo.DateTimeOriginal
		record.HasCaptureTime = true
	}

	return record
}

// parseExifDateTime 解析 EXIF 日期时间格式
func parseExifDateTime(s string) (time.Time, error) {
	layouts := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析日期: %s", s)
}
