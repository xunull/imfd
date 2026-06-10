package extract

import (
	"os"

	"github.com/xunull/imfd/internal/media"
)

// Extract 统一提取接口
// 根据文件类型选择对应的提取器
func Extract(filePath string) *media.MediaRecord {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return &media.MediaRecord{
			FilePath: filePath,
			Error:    err,
		}
	}

	mediaType := media.ClassifyFile(filePath)
	switch mediaType {
	case media.TypeImage:
		return BuildImageRecord(filePath, fileInfo)
	case media.TypeVideo:
		return BuildVideoRecord(filePath, fileInfo)
	case media.TypeAudio:
		return BuildAudioRecord(filePath, fileInfo)
	default:
		// 非媒体文件仍返回一个最小填充的 record，让 imfd info 能展示 FILE section
		return &media.MediaRecord{
			FilePath: filePath,
			FileName: fileInfo.Name(),
			FileSize: fileInfo.Size(),
			ModTime:  fileInfo.ModTime(),
			Type:     media.TypeUnknown,
		}
	}
}
