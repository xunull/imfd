package extract

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/xunull/imfd/internal/media"
)

// ExtractImageExif 从图像文件中提取 EXIF 信息
func ExtractImageExif(filePath string) (*media.ExifInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("EXIF 解析失败: %w", err)
	}

	info := &media.ExifInfo{}

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

	if w := getTagInt(x, exif.PixelXDimension); w > 0 {
		info.ImageWidth = w
	}
	if h := getTagInt(x, exif.PixelYDimension); h > 0 {
		info.ImageHeight = h
	}

	if dt, err := x.DateTime(); err == nil {
		info.DateTimeOriginal = dt
		info.HasDateTime = true
	}

	lat, lon, err := x.LatLong()
	if err == nil {
		info.GPS = media.GPSInfo{
			Latitude:  lat,
			Longitude: lon,
			HasGPS:    true,
		}
	}

	return info, nil
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
