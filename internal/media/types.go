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
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
	HasGPS    bool    `json:"has_gps"`
}

// GeoLocation 反查后的地理位置
type GeoLocation struct {
	Country  string `json:"country,omitempty"`
	Province string `json:"province,omitempty"`
	City     string `json:"city,omitempty"`
}

// ExifInfo EXIF 信息
type ExifInfo struct {
	// 相机与镜头
	CameraMake  string `json:"camera_make,omitempty"`
	CameraModel string `json:"camera_model,omitempty"`
	LensMake    string `json:"lens_make,omitempty"`
	LensModel   string `json:"lens_model,omitempty"`

	// 曝光参数
	Aperture             string `json:"aperture,omitempty"`              // 光圈 f 值
	ShutterSpeed         string `json:"shutter_speed,omitempty"`         // 快门速度
	FocalLength          string `json:"focal_length,omitempty"`          // 焦距
	FocalLength35mm      string `json:"focal_length_35mm,omitempty"`     // 等效 35mm 焦距
	ISO                  string `json:"iso,omitempty"`                   // 感光度
	WhiteBalance         string `json:"white_balance,omitempty"`         // 白平衡
	ExposureCompensation string `json:"exposure_compensation,omitempty"` // 曝光补偿
	ExposureTime         string `json:"exposure_time,omitempty"`         // 曝光时间
	ExposureMode         string `json:"exposure_mode,omitempty"`         // 曝光模式
	ExposureProgram      string `json:"exposure_program,omitempty"`      // 曝光程序
	MeteringMode         string `json:"metering_mode,omitempty"`         // 测光模式
	Flash                string `json:"flash,omitempty"`                 // 闪光灯
	ColorSpace           string `json:"color_space,omitempty"`           // 色彩空间

	// 图像尺寸
	ImageWidth  int `json:"image_width,omitempty"`
	ImageHeight int `json:"image_height,omitempty"`

	// 拍摄时间
	DateTimeOriginal time.Time `json:"date_time_original,omitzero"`
	HasDateTime      bool      `json:"has_date_time,omitempty"`

	// GPS
	GPS GPSInfo `json:"gps"`
}

// VideoInfo 视频特有信息
type VideoInfo struct {
	Duration    float64   `json:"duration,omitempty"` // 时长（秒）
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	Codec       string    `json:"codec,omitempty"`
	AudioCodec  string    `json:"audio_codec,omitempty"`
	Bitrate     int64     `json:"bitrate,omitempty"`
	FrameRate   string    `json:"frame_rate,omitempty"`
	CreateTime  time.Time `json:"create_time,omitzero"`
	HasDateTime bool      `json:"has_date_time,omitempty"`
}

// AudioInfo 音频特有信息
//
// 注意：音频文件的"录制时间"不向上传播到 MediaRecord.CaptureTime —— 录音年份
// 和摄影"拍摄时间"是两个语义概念，混在一起会让「拍摄时间段」维度把 1965 年的
// 老歌归类成"凌晨"。RecordedTime 留作 audio-only 字段，未来如需"录制年代"维度
// 从这里取。
type AudioInfo struct {
	Codec           string    `json:"codec,omitempty"`          // 主音轨编解码器（mp3/flac/aac/...）
	Bitrate         int64     `json:"bitrate,omitempty"`        // 比特率（bps）
	SampleRate      int       `json:"sample_rate,omitempty"`    // 采样率（Hz）
	Channels        int       `json:"channels,omitempty"`       // 声道数
	ChannelLayout   string    `json:"channel_layout,omitempty"` // 声道布局（mono/stereo/5.1/...）
	Duration        float64   `json:"duration,omitempty"`       // 时长（秒）
	RecordedTime    time.Time `json:"recorded_time,omitzero"`
	HasRecordedTime bool      `json:"has_recorded_time,omitempty"`
}

// MediaRecord 统一媒体记录
//
// JSON 字段命名走 snake_case，和 stats.Totals 的 JSON 契约一致。
type MediaRecord struct {
	FilePath string    `json:"file_path"`
	FileName string    `json:"file_name"`
	FileSize int64     `json:"file_size"`
	ModTime  time.Time `json:"mod_time"` // 文件系统 mtime（os.FileInfo.ModTime）

	Type MediaType `json:"type"`

	// EXIF 信息（图像时填充）
	Exif *ExifInfo `json:"exif,omitempty"`

	// 视频信息
	Video *VideoInfo `json:"video,omitempty"`

	// 音频信息（仅 TypeAudio 时填充）
	Audio *AudioInfo `json:"audio,omitempty"`

	// 反查后的地理位置
	Location *GeoLocation `json:"location,omitempty"`

	// 拍摄/创建时间（从 EXIF 或视频元数据中提取）
	CaptureTime    time.Time `json:"capture_time,omitzero"`
	HasCaptureTime bool      `json:"has_capture_time,omitempty"`

	// 扩展属性，用于存放尚未结构化的字段（含 *_error 键记录提取错误）
	Attributes map[string]string `json:"attributes,omitempty"`

	// 处理时出现的错误
	Error error `json:"-"` // 不进 JSON（error 接口序列化为 {}，没用）
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
