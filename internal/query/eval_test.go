package query

import (
	"errors"
	"testing"
	"time"

	"github.com/xunull/imfd/internal/media"
)

func makeImageRecord(make_, model, iso, prov string, hour int) *media.MediaRecord {
	return &media.MediaRecord{
		FilePath: "/x/p.jpg",
		Type:     media.TypeImage,
		Exif: &media.ExifInfo{
			CameraMake:       make_,
			CameraModel:      model,
			ISO:              iso,
			HasDateTime:      hour > 0,
			DateTimeOriginal: time.Date(2024, 1, 1, hour, 0, 0, 0, time.UTC),
		},
		Location: &media.GeoLocation{Province: prov},
	}
}

func TestNewEvaluator_SyntaxError(t *testing.T) {
	_, err := NewEvaluator("iso >>> 800", nil)
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !errors.Is(err, SyntaxError) {
		t.Errorf("err should wrap SyntaxError, got %v", err)
	}
}

func TestNewEvaluator_EmptyFilterMatchesAll(t *testing.T) {
	ev, err := NewEvaluator("", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := ev.Match(makeImageRecord("Sony", "A7", "800", "云南省", 14))
	if !got {
		t.Error("empty filter should match all records, got false")
	}
}

func TestEvaluator_Match_HappyPath(t *testing.T) {
	ev, err := NewEvaluator(`type == "image" and iso > 400`, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := makeImageRecord("Sony", "A7", "800", "", 0)
	got, err := ev.Match(r)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("expected match")
	}
}

func TestEvaluator_Match_NeedleSubstring(t *testing.T) {
	ev, err := NewEvaluator(`lower(camera_make) contains lower(needle1)`, []string{"Sony"})
	if err != nil {
		t.Fatal(err)
	}
	r := makeImageRecord("SONY CORPORATION", "A7", "800", "", 0)
	got, err := ev.Match(r)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("lowercase substring should match SONY")
	}
}

func TestEvaluator_Match_ChineseContains(t *testing.T) {
	ev, err := NewEvaluator(`province contains needle1`, []string{"云南"})
	if err != nil {
		t.Fatal(err)
	}
	r := makeImageRecord("Sony", "A7", "800", "云南省", 12)
	got, _ := ev.Match(r)
	if !got {
		t.Error("Chinese substring should match 云南省")
	}
}

// nil-safe: audio 文件没 EXIF/ISO，filter `iso > 800` 应 false 不报错
// 实现是 typed-zero（iso=0），0 > 800 → false
func TestEvaluator_Match_NilSafeViaTypedZero(t *testing.T) {
	ev, err := NewEvaluator("iso > 800", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ev.Match(nil) // nil record → 所有字段填 zero
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("nil record iso > 800 should be false")
	}
}

func TestEvaluator_Match_NilSafeChineseEqual(t *testing.T) {
	// 没 GPS 的照片 → province=""，province == "云南" → false（不报错）
	ev, err := NewEvaluator(`province == "云南"`, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := makeImageRecord("Sony", "A7", "800", "", 0) // 无 province
	got, _ := ev.Match(r)
	if got {
		t.Error("no GPS record should not match province==云南")
	}
}
