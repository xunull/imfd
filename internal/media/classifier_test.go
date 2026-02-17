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
	if IsMediaFile("readme.txt") {
		t.Error("Expected readme.txt to not be a media file")
	}
}
