package media

import (
	"testing"
	"time"
)

func mkRecord(iso, shutter string, hour int, hasTime bool) *MediaRecord {
	return &MediaRecord{
		Exif: &ExifInfo{
			ISO:              iso,
			ShutterSpeed:     shutter,
			HasDateTime:      hasTime,
			DateTimeOriginal: time.Date(2024, 1, 1, hour, 0, 0, 0, time.UTC),
		},
	}
}

func TestIsStarrySky(t *testing.T) {
	cases := []struct {
		name string
		r    *MediaRecord
		want bool
	}{
		{"happy 23 点 ISO 2000 30s", mkRecord("2000", "30s", 23, true), true},
		{"happy 凌晨 3 点", mkRecord("3200", "20s", 3, true), true},
		{"happy 边界 hour=22", mkRecord("2000", "15s", 22, true), true},
		{"happy 边界 hour=4", mkRecord("2000", "15s", 4, true), true},
		{"fail hour=21 不在 [22,4]", mkRecord("2000", "15s", 21, true), false},
		{"fail hour=5 不在 [22,4]", mkRecord("2000", "15s", 5, true), false},
		{"fail hour=12 中午", mkRecord("2000", "15s", 12, true), false},
		{"fail ISO 800 太低", mkRecord("800", "15s", 23, true), false},
		{"fail ISO 1600 不严格大于", mkRecord("1600", "15s", 23, true), false},
		{"fail 快门 5s 太快", mkRecord("2000", "5s", 23, true), false},
		{"fail 快门 1/250 远不够", mkRecord("2000", "1/250s", 23, true), false},
		{"fail HasDateTime=false", mkRecord("2000", "15s", 23, false), false},
		{"fail ISO 解析不出", mkRecord("auto", "15s", 23, true), false},
		{"fail Shutter 解析不出", mkRecord("2000", "auto", 23, true), false},
		{"nil record", nil, false},
		{"nil Exif", &MediaRecord{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IsStarrySky(c.r)
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
