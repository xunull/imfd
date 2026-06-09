package stats

import (
	"fmt"
	"testing"

	"github.com/xunull/imfd/internal/media"
)

func TestGroupCounter(t *testing.T) {
	counter := NewGroupCounter(
		"test_dimension",
		func(record *media.MediaRecord) []string {
			return []string{record.GetCameraModel()}
		},
		DimensionMeta{SortBy: "count", SortOrder: "desc"},
	)

	records := []*media.MediaRecord{
		{Type: media.TypeImage, Exif: &media.ExifInfo{CameraModel: "Canon EOS R5"}},
		{Type: media.TypeImage, Exif: &media.ExifInfo{CameraModel: "Canon EOS R5"}},
		{Type: media.TypeImage, Exif: &media.ExifInfo{CameraModel: "Sony A7IV"}},
		{Type: media.TypeImage},
	}

	for _, r := range records {
		counter.Consume(r)
	}

	result := counter.Result()
	if result.DimensionName != "test_dimension" {
		t.Errorf("expected name 'test_dimension', got %q", result.DimensionName)
	}
	if len(result.Buckets) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(result.Buckets))
	}

	counts := make(map[string]int)
	for _, b := range result.Buckets {
		counts[b.Key] = b.Count
	}

	if counts["Canon EOS R5"] != 2 {
		t.Errorf("expected Canon EOS R5 count 2, got %d", counts["Canon EOS R5"])
	}
	if counts["Sony A7IV"] != 1 {
		t.Errorf("expected Sony A7IV count 1, got %d", counts["Sony A7IV"])
	}
	if counts["Unknown"] != 1 {
		t.Errorf("expected Unknown count 1, got %d", counts["Unknown"])
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	cameraCounter := NewGroupCounter(
		"camera",
		func(record *media.MediaRecord) []string {
			return []string{record.GetCameraModel()}
		},
		DimensionMeta{},
	)

	registry.Register(cameraCounter)

	registry.Consume(&media.MediaRecord{Type: media.TypeImage, Exif: &media.ExifInfo{CameraModel: "Canon"}})
	registry.Consume(&media.MediaRecord{Type: media.TypeVideo})
	registry.Consume(&media.MediaRecord{Type: media.TypeImage, Exif: &media.ExifInfo{CameraModel: "Canon"}})
	registry.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: "mp3"}})
	registry.Consume(&media.MediaRecord{Type: media.TypeAudio, Audio: &media.AudioInfo{Codec: "flac"}})

	report := registry.Report()

	if report.Totals.ImageCount != 2 {
		t.Errorf("expected 2 images, got %d", report.Totals.ImageCount)
	}
	if report.Totals.VideoCount != 1 {
		t.Errorf("expected 1 video, got %d", report.Totals.VideoCount)
	}
	if report.Totals.AudioCount != 2 {
		t.Errorf("expected 2 audios, got %d", report.Totals.AudioCount)
	}
	if report.Totals.TotalCount != 5 {
		t.Errorf("expected 5 total, got %d", report.Totals.TotalCount)
	}
	// 验证 Totals 契约：sum 自洽
	sum := report.Totals.ImageCount + report.Totals.VideoCount + report.Totals.AudioCount
	if sum != report.Totals.TotalCount {
		t.Errorf("Totals 契约破坏: image+video+audio=%d != total=%d", sum, report.Totals.TotalCount)
	}
	if len(report.Dimensions) != 1 {
		t.Errorf("expected 1 dimension, got %d", len(report.Dimensions))
	}
}

func TestRegistryErrorHandling(t *testing.T) {
	registry := NewRegistry()

	errRecord := &media.MediaRecord{
		Error: fmt.Errorf("test error"),
	}
	registry.Consume(errRecord)

	report := registry.Report()
	if report.Totals.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", report.Totals.ErrorCount)
	}
	if report.Totals.TotalCount != 0 {
		t.Errorf("expected 0 total for error records, got %d", report.Totals.TotalCount)
	}
}

func TestGroupCounterReset(t *testing.T) {
	counter := NewGroupCounter(
		"test",
		func(record *media.MediaRecord) []string {
			return []string{"key"}
		},
		DimensionMeta{},
	)

	counter.Consume(&media.MediaRecord{Type: media.TypeImage})
	counter.Reset()

	result := counter.Result()
	if len(result.Buckets) != 0 {
		t.Errorf("expected 0 buckets after reset, got %d", len(result.Buckets))
	}
}
