package media

import "testing"

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected MediaType
	}{
		{"jpg", "photo.jpg", TypeImage},
		{"JPG_upper", "PHOTO.JPG", TypeImage},
		{"jpeg", "test.jpeg", TypeImage},
		{"png", "icon.png", TypeImage},
		{"heic", "IMG_001.HEIC", TypeImage},
		{"cr2_raw", "raw.CR2", TypeImage},
		{"nef_raw", "raw.nef", TypeImage},
		{"dng_raw", "raw.dng", TypeImage},
		{"mp4", "video.mp4", TypeVideo},
		{"MOV_upper", "clip.MOV", TypeVideo},
		{"avi", "movie.avi", TypeVideo},
		{"mkv", "movie.mkv", TypeVideo},
		{"mp3", "song.mp3", TypeAudio},
		{"MP3_upper", "SONG.MP3", TypeAudio},
		{"flac", "lossless.flac", TypeAudio},
		{"m4a", "podcast.m4a", TypeAudio},
		{"aac", "stream.aac", TypeAudio},
		{"ogg_vorbis", "music.ogg", TypeAudio},
		{"opus", "voice.opus", TypeAudio},
		{"wav", "recording.WAV", TypeAudio},
		{"aiff", "demo.aiff", TypeAudio},
		{"dsd_dsf", "hi-res.dsf", TypeAudio},
		{"txt", "readme.txt", TypeUnknown},
		{"go", "main.go", TypeUnknown},
		{"no_ext", "noext", TypeUnknown},
		{"empty", "", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyFile(tt.filename)
			if result != tt.expected {
				t.Errorf("ClassifyFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsMediaFile(t *testing.T) {
	if !IsMediaFile("photo.jpg") {
		t.Error("Expected photo.jpg to be a media file")
	}
	if !IsMediaFile("video.mp4") {
		t.Error("Expected video.mp4 to be a media file")
	}
	if !IsMediaFile("song.mp3") {
		t.Error("Expected song.mp3 to be a media file")
	}
	if !IsMediaFile("music.flac") {
		t.Error("Expected music.flac to be a media file")
	}
	if IsMediaFile("readme.txt") {
		t.Error("Expected readme.txt to not be a media file")
	}
}

func TestIsMatchedFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		allowed  []MediaType
		want     bool
	}{
		// nil allowed = 全媒体（向后兼容 IsMediaFile 行为）
		{"nil_jpg", "p.jpg", nil, true},
		{"nil_mp4", "p.mp4", nil, true},
		{"nil_mp3", "p.mp3", nil, true},
		{"nil_txt", "p.txt", nil, false},

		// audio-only
		{"audio_only_mp3", "song.mp3", []MediaType{TypeAudio}, true},
		{"audio_only_flac", "song.flac", []MediaType{TypeAudio}, true},
		{"audio_only_jpg", "photo.jpg", []MediaType{TypeAudio}, false},
		{"audio_only_mp4", "movie.mp4", []MediaType{TypeAudio}, false},
		{"audio_only_txt", "readme.txt", []MediaType{TypeAudio}, false},

		// image-only
		{"image_only_jpg", "photo.jpg", []MediaType{TypeImage}, true},
		{"image_only_heic", "img.HEIC", []MediaType{TypeImage}, true},
		{"image_only_mp3", "song.mp3", []MediaType{TypeImage}, false},

		// video-only
		{"video_only_mp4", "clip.mp4", []MediaType{TypeVideo}, true},
		{"video_only_mp3", "song.mp3", []MediaType{TypeVideo}, false},

		// 多类型组合
		{"audio_image_mp3", "song.mp3", []MediaType{TypeAudio, TypeImage}, true},
		{"audio_image_jpg", "photo.jpg", []MediaType{TypeAudio, TypeImage}, true},
		{"audio_image_mp4", "movie.mp4", []MediaType{TypeAudio, TypeImage}, false},

		// 空 slice（不是 nil！）= 不匹配任何（显式收紧）
		{"empty_slice_mp3", "song.mp3", []MediaType{}, false},
		{"empty_slice_jpg", "photo.jpg", []MediaType{}, false},

		// 非媒体文件无论 allowed 怎么写都拒
		{"txt_with_audio_allowed", "readme.txt", []MediaType{TypeAudio}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMatchedFile(tt.filename, tt.allowed)
			if got != tt.want {
				t.Errorf("IsMatchedFile(%q, %v) = %v, want %v", tt.filename, tt.allowed, got, tt.want)
			}
		})
	}
}
