package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xunull/imfd/internal/media"
)

func newTablePrinterForTest() (*bytes.Buffer, *FileInfoPrinter) {
	buf := &bytes.Buffer{}
	p := NewFileInfoPrinter(buf, "table")
	// 强制关色让断言简单（bytes.Buffer 走不到 TTY 路径，已经默认 disable，
	// 但显式保险）
	p.color = NewColorer(false)
	return buf, p
}

func imageRecordFixture() *media.MediaRecord {
	return &media.MediaRecord{
		FilePath: "/test/photo.jpg",
		FileName: "photo.jpg",
		FileSize: 2_400_000,
		ModTime:  time.Date(2024, 8, 12, 15, 30, 42, 0, time.UTC),
		Type:     media.TypeImage,
		Exif: &media.ExifInfo{
			CameraMake:       "Canon",
			CameraModel:      "EOS 1300D",
			LensModel:        "EF-S18-55mm f/3.5-5.6 IS II",
			ISO:              "800",
			Aperture:         "f/5",
			ShutterSpeed:     "1/60s",
			FocalLength:      "42mm",
			ImageWidth:       6000,
			ImageHeight:      4000,
			ExposureMode:     "Auto",
			ExposureProgram:  "Portrait mode",
			WhiteBalance:     "Auto",
			MeteringMode:     "Pattern",
			Flash:            "Fired",
			DateTimeOriginal: time.Date(2024, 3, 15, 14, 22, 11, 0, time.UTC),
			HasDateTime:      true,
			GPS: media.GPSInfo{
				Latitude:  31.2304,
				Longitude: 121.4737,
				HasGPS:    true,
			},
		},
		Location: &media.GeoLocation{
			Country:  "中国",
			Province: "上海市",
			City:     "黄浦区",
		},
	}
}

func TestPrint_ImageRecord_AllSections(t *testing.T) {
	buf, p := newTablePrinterForTest()
	if err := p.Print(imageRecordFixture()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"FILE", "/test/photo.jpg", "2.29 MB", "2024-08-12 15:30:42", "image",
		"EXIF", "Canon EOS 1300D", "EF-S18-55mm", "800", "f/5", "1/60s", "6000x4000",
		"GPS", "31.230400", "121.473700", "上海市 / 黄浦区",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}

	// AUDIO/VIDEO section 不该出现（image record）
	for _, unwanted := range []string{"AUDIO", "VIDEO", "ERRORS"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected section %q in image output:\n%s", unwanted, out)
		}
	}
}

func TestPrint_AudioRecord(t *testing.T) {
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/song.mp3",
		FileName: "song.mp3",
		FileSize: 5_000_000,
		Type:     media.TypeAudio,
		Audio: &media.AudioInfo{
			Codec:         "mp3",
			Bitrate:       192_000,
			SampleRate:    44100,
			Channels:      2,
			ChannelLayout: "stereo",
			Duration:      245.6,
		},
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{"FILE", "AUDIO", "mp3", "192 kbps", "44100 Hz", "stereo", "4m05s"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in audio output:\n%s", want, out)
		}
	}

	for _, unwanted := range []string{"EXIF", "GPS", "VIDEO"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected section %q in audio output", unwanted)
		}
	}
}

func TestPrint_VideoRecord(t *testing.T) {
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/clip.mp4",
		FileName: "clip.mp4",
		FileSize: 10_000_000,
		Type:     media.TypeVideo,
		Video: &media.VideoInfo{
			Codec:      "h264",
			AudioCodec: "aac",
			Width:      1920,
			Height:     1080,
			Bitrate:    5_200_000,
			FrameRate:  "30/1",
			Duration:   60.0,
		},
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{"FILE", "VIDEO", "h264", "aac", "1920x1080", "5.20 Mbps", "30/1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in video output:\n%s", want, out)
		}
	}

	for _, unwanted := range []string{"EXIF", "AUDIO 文件 section", "GPS"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected section %q in video output", unwanted)
		}
	}
}

func TestPrint_NonMediaRecord_ShowsTypeUnknown(t *testing.T) {
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/readme.txt",
		FileName: "readme.txt",
		FileSize: 100,
		Type:     media.TypeUnknown,
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "FILE") {
		t.Error("non-media should still print FILE section")
	}
	if !strings.Contains(out, "unknown") {
		t.Errorf("non-media should show 'unknown' type label, got:\n%s", out)
	}
}

func TestPrint_RecordWithErrors_ShowsErrorsSection(t *testing.T) {
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/corrupt.jpg",
		Type:     media.TypeImage,
		Attributes: map[string]string{
			"exif_error":  "EXIF 解析失败: EOF",
			"video_error": "ffprobe timeout",
		},
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ERRORS") {
		t.Errorf("want ERRORS section when *_error attrs present, got:\n%s", out)
	}
	if !strings.Contains(out, "EXIF 解析失败") {
		t.Errorf("ERRORS section should include exif_error message")
	}
	if !strings.Contains(out, "ffprobe timeout") {
		t.Errorf("ERRORS section should include video_error message")
	}
}

func TestPrint_RecordWithSystemError(t *testing.T) {
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/x.jpg",
		Type:     media.TypeImage,
		Error:    errors.New("permission denied"),
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ERRORS") {
		t.Errorf("want ERRORS section when r.Error != nil")
	}
	if !strings.Contains(out, "permission denied") {
		t.Errorf("ERRORS should include system error message")
	}
}

func TestPrint_NilSubstructs_NoPanic(t *testing.T) {
	// Exif/Video/Audio/Location/Attributes 全 nil 不该 panic
	buf, p := newTablePrinterForTest()
	record := &media.MediaRecord{
		FilePath: "/test/x.unknown",
		Type:     media.TypeUnknown,
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "FILE") {
		t.Error("want FILE section even with all nil substructs")
	}
}

func TestPrint_JSON_OmitsEmptySubstructs(t *testing.T) {
	buf := &bytes.Buffer{}
	p := NewFileInfoPrinter(buf, "json")
	record := &media.MediaRecord{
		FilePath: "/test/audio.mp3",
		FileName: "audio.mp3",
		FileSize: 1000,
		Type:     media.TypeAudio,
		Audio:    &media.AudioInfo{Codec: "mp3"},
	}
	if err := p.Print(record); err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output should be valid JSON, got error %v\n%s", err, buf.String())
	}
	if _, ok := decoded["file_path"]; !ok {
		t.Error("JSON should have snake_case file_path key")
	}
	if _, ok := decoded["audio"]; !ok {
		t.Error("JSON should have audio key when Audio != nil")
	}
	// omitempty 字段：image record 没 exif，应该不出现
	if _, ok := decoded["exif"]; ok {
		t.Error("JSON should omit exif key when Exif == nil")
	}
	if _, ok := decoded["video"]; ok {
		t.Error("JSON should omit video key when Video == nil")
	}
}

func TestPrint_TableNoColorByDefault(t *testing.T) {
	// bytes.Buffer 不是 *os.File → colorer 应该自动 disabled
	buf := &bytes.Buffer{}
	p := NewFileInfoPrinter(buf, "table")
	if err := p.Print(imageRecordFixture()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("bytes.Buffer 输出不该有 ANSI escape, got:\n%q", buf.String())
	}
}

func TestFormatFileSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.00 KB"},
		{2_400_000, "2.29 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{int64(1.5 * 1024 * 1024 * 1024 * 1024), "1.50 TB"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := formatFileSize(c.n); got != c.want {
				t.Errorf("formatFileSize(%d) = %q, want %q", c.n, got, c.want)
			}
		})
	}
}

func TestFormatBitrate(t *testing.T) {
	cases := []struct {
		bps  int64
		want string
	}{
		{128_000, "128 kbps"},
		{320_000, "320 kbps"},
		{5_200_000, "5.20 Mbps"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := formatBitrate(c.bps); got != c.want {
				t.Errorf("formatBitrate(%d) = %q, want %q", c.bps, got, c.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		secs float64
		want string
	}{
		{12.34, "12.34s"},
		{60, "1m00s"},
		{245.6, "4m05s"},
		{3600, "1h00m00s"},
		{3725, "1h02m05s"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := formatDuration(c.secs); got != c.want {
				t.Errorf("formatDuration(%v) = %q, want %q", c.secs, got, c.want)
			}
		})
	}
}
