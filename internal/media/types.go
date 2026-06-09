package media

import "time"

// MediaType 媒体类型
type MediaType int

const (
	TypeUnknown MediaType = iota
	TypeImage
	TypeVideo
	TypeAudio
)

func (t MediaType) String() string {
	switch t {
	case TypeImage:
		return "image"
	case TypeVideo:
		return "video"
	case TypeAudio:
		return "audio"
	default:
		return "unknown"
	}
}

// GPSInfo GPS 位置信息
type GPSInfo struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	HasGPS    bool
}

// GeoLocation 反查后的地理位置
type GeoLocation struct {
	Country  string
	Province string
	City     string
}

// ExifInfo EXIF 信息
type ExifInfo struct {
	// 相机与镜头
	CameraMake  string
	CameraModel string
	LensMake    string
	LensModel   string

	// 曝光参数
	Aperture           string // 光圈 f 值
	ShutterSpeed       string // 快门速度
	FocalLength        string // 焦距
	FocalLength35mm    string // 等效 35mm 焦距
	ISO                string // 感光度
	WhiteBalance       string // 白平衡
	ExposureCompensation string // 曝光补偿
	ExposureTime       string // 曝光时间
	ExposureMode       string // 曝光模式
	ExposureProgram    string // 曝光程序
	MeteringMode       string // 测光模式
	Flash              string // 闪光灯
	ColorSpace         string // 色彩空间

	// 图像尺寸
	ImageWidth  int
	ImageHeight int

	// 拍摄时间
	DateTimeOriginal time.Time
	HasDateTime      bool

	// GPS
	GPS GPSInfo
}

// VideoInfo 视频特有信息
type VideoInfo struct {
	Duration    float64 // 时长（秒）
	Width       int
	Height      int
	Codec       string
	AudioCodec  string
	Bitrate     int64
	FrameRate   string
	CreateTime  time.Time
	HasDateTime bool
}

// AudioInfo 音频特有信息
//
// 注意：音频文件的"录制时间"不向上传播到 MediaRecord.CaptureTime —— 录音年份
// 和摄影"拍摄时间"是两个语义概念，混在一起会让「拍摄时间段」维度把 1965 年的
// 老歌归类成"凌晨"。RecordedTime 留作 audio-only 字段，未来如需"录制年代"维度
// 从这里取。
type AudioInfo struct {
	Codec            string  // 主音轨编解码器（mp3/flac/aac/...）
	Bitrate          int64   // 比特率（bps）
	SampleRate       int     // 采样率（Hz）
	Channels         int     // 声道数
	ChannelLayout    string  // 声道布局（mono/stereo/5.1/...）
	Duration         float64 // 时长（秒）
	RecordedTime     time.Time
	HasRecordedTime  bool
}

// MediaRecord 统一媒体记录
type MediaRecord struct {
	FilePath string
	FileName string
	FileSize int64
	Type     MediaType

	// EXIF 信息（图像时填充）
	Exif *ExifInfo

	// 视频信息
	Video *VideoInfo

	// 音频信息（仅 TypeAudio 时填充）
	Audio *AudioInfo

	// 反查后的地理位置
	Location *GeoLocation

	// 拍摄/创建时间（从 EXIF 或视频元数据中提取）
	CaptureTime    time.Time
	HasCaptureTime bool

	// 扩展属性，用于存放尚未结构化的字段
	Attributes map[string]string

	// 处理时出现的错误
	Error error
}

// GetCameraModel 获取相机型号
func (r *MediaRecord) GetCameraModel() string {
	if r.Exif != nil && r.Exif.CameraModel != "" {
		return r.Exif.CameraModel
	}
	return "Unknown"
}

// GetLensModel 获取镜头型号
func (r *MediaRecord) GetLensModel() string {
	if r.Exif != nil && r.Exif.LensModel != "" {
		return r.Exif.LensModel
	}
	return "Unknown"
}

// GetProvince 获取省份
func (r *MediaRecord) GetProvince() string {
	if r.Location != nil && r.Location.Province != "" {
		return r.Location.Province
	}
	return "Unknown"
}

// GetCity 获取城市
func (r *MediaRecord) GetCity() string {
	if r.Location != nil && r.Location.City != "" {
		return r.Location.City
	}
	return "Unknown"
}

// GetProvinceCity 获取省市组合
func (r *MediaRecord) GetProvinceCity() string {
	province := r.GetProvince()
	city := r.GetCity()
	if province == "Unknown" && city == "Unknown" {
		return "Unknown"
	}
	if province == city {
		return province
	}
	return province + "/" + city
}

// HasGPS 是否有 GPS 信息
func (r *MediaRecord) HasGPS() bool {
	if r.Exif != nil {
		return r.Exif.GPS.HasGPS
	}
	return false
}
