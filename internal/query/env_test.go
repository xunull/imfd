package query

import (
	"testing"
	"time"

	"github.com/xunull/imfd/internal/media"
)

func TestBuildEnv_NilRecord_AllZeroValues(t *testing.T) {
	env := BuildEnv(nil, nil)
	// nil record → 所有字段填类型化 zero values（不是 nil interface）
	if env["camera_make"] != "" {
		t.Errorf("camera_make zero should be empty string, got %v", env["camera_make"])
	}
	if env["iso"] != 0 {
		t.Errorf("iso zero should be 0, got %v", env["iso"])
	}
	if env["scene_starry_sky"] != false {
		t.Errorf("scene_starry_sky zero should be false, got %v", env["scene_starry_sky"])
	}
}

func TestBuildEnv_ImageRecord(t *testing.T) {
	r := &media.MediaRecord{
		FilePath: "/x/photo.jpg",
		FileName: "photo.jpg",
		FileSize: 1024,
		Type:     media.TypeImage,
		Exif: &media.ExifInfo{
			CameraMake:       "Apple",
			CameraModel:      "iPhone 15",
			ISO:              "800",
			Aperture:         "f/5",
			ShutterSpeed:     "1/250s",
			FocalLength:      "42mm",
			ImageWidth:       6000,
			ImageHeight:      4000,
			HasDateTime:      true,
			DateTimeOriginal: time.Date(2024, 5, 1, 14, 0, 0, 0, time.UTC),
		},
		Location: &media.GeoLocation{Province: "云南省", City: "昆明市"},
	}
	env := BuildEnv(r, nil)
	if env["camera_make"] != "Apple" {
		t.Errorf("camera_make: %v", env["camera_make"])
	}
	if env["iso"] != 800 {
		t.Errorf("iso: %v", env["iso"])
	}
	if env["aperture_value"] != 5.0 {
		t.Errorf("aperture_value: %v", env["aperture_value"])
	}
	if env["province"] != "云南省" {
		t.Errorf("province: %v", env["province"])
	}
	if env["capture_year"] != 2024 {
		t.Errorf("capture_year: %v", env["capture_year"])
	}
	if env["device_type"] != media.DeviceTypePhone {
		t.Errorf("device_type: %v", env["device_type"])
	}
}

func TestBuildEnv_AudioRecord_ExifFieldsZero(t *testing.T) {
	r := &media.MediaRecord{
		FilePath: "/x/song.mp3",
		Type:     media.TypeAudio,
		Audio:    &media.AudioInfo{Codec: "mp3", Bitrate: 192000},
	}
	env := BuildEnv(r, nil)
	if env["audio_codec"] != "mp3" {
		t.Errorf("audio_codec: %v", env["audio_codec"])
	}
	if env["camera_make"] != "" {
		t.Errorf("camera_make should be empty string for audio, got %v", env["camera_make"])
	}
	if env["iso"] != 0 {
		t.Errorf("iso should be 0 for audio, got %v", env["iso"])
	}
}

func TestBuildEnv_NeedlesInjected(t *testing.T) {
	env := BuildEnv(nil, []string{"Sony's Best", "Nikon"})
	if env["needle1"] != "Sony's Best" {
		t.Errorf("needle1: %v", env["needle1"])
	}
	if env["needle2"] != "Nikon" {
		t.Errorf("needle2: %v", env["needle2"])
	}
}

func TestBuildEnv_EmptyExifStringsStayEmpty(t *testing.T) {
	r := &media.MediaRecord{
		Type: media.TypeImage,
		Exif: &media.ExifInfo{CameraMake: "", CameraModel: ""}, // 空字符串
	}
	env := BuildEnv(r, nil)
	if env["camera_make"] != "" {
		t.Errorf("empty camera_make stays empty string, got %v", env["camera_make"])
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 9: "9", 10: "10", 42: "42", 1234: "1234"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}
