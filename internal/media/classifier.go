package media

import (
	"path/filepath"
	"slices"
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

// 支持的音频扩展名
var audioExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".aac":  true,
	".m4a":  true,
	".ogg":  true,
	".oga":  true,
	".opus": true,
	".wav":  true,
	".wma":  true,
	".ape":  true,
	".wv":   true,
	".alac": true,
	".dsd":  true,
	".dsf":  true,
	".dff":  true,
	".aiff": true,
	".aif":  true,
	".amr":  true,
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
	if audioExtensions[ext] {
		return TypeAudio
	}
	return TypeUnknown
}

// IsMediaFile 判断是否为媒体文件
func IsMediaFile(filename string) bool {
	return ClassifyFile(filename) != TypeUnknown
}

// IsMatchedFile 判断文件是否匹配给定的媒体类型集合。
//
// allowed=nil 时等价于 IsMediaFile，保持向后兼容（既有 walker 调用方传 nil 即可）。
// 非 nil 时只接受 allowed 中列出的类型。
//
// allowed 用 slice 而不是 map 表达：典型场景下 len(allowed)==1，O(n) 扫描比 map 快、且零分配。
func IsMatchedFile(filename string, allowed []MediaType) bool {
	t := ClassifyFile(filename)
	if t == TypeUnknown {
		return false
	}
	if allowed == nil {
		return true
	}
	return slices.Contains(allowed, t)
}
