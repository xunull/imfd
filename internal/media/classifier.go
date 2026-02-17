package media

import (
	"path/filepath"
	"strings"
)

// 支持的图像扩展名
var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".webp": true,
	".heic": true,
	".heif": true,
	".raw":  true,
	".cr2":  true,
	".cr3":  true,
	".nef":  true,
	".arw":  true,
	".dng":  true,
	".orf":  true,
	".rw2":  true,
	".pef":  true,
	".sr2":  true,
	".raf":  true,
}

// 支持的视频扩展名
var videoExtensions = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".wmv":  true,
	".flv":  true,
	".m4v":  true,
	".mpg":  true,
	".mpeg": true,
	".3gp":  true,
	".webm": true,
	".mts":  true,
	".m2ts": true,
	".ts":   true,
}

// ClassifyFile 根据扩展名判断文件类型
func ClassifyFile(filename string) MediaType {
	ext := strings.ToLower(filepath.Ext(filename))
	if imageExtensions[ext] {
		return TypeImage
	}
	if videoExtensions[ext] {
		return TypeVideo
	}
	return TypeUnknown
}

// IsMediaFile 判断是否为媒体文件
func IsMediaFile(filename string) bool {
	return ClassifyFile(filename) != TypeUnknown
}
