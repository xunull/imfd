package query

import (
	"strings"
	"testing"
)

func TestBuildFilter_AllEmpty_MatchesAll(t *testing.T) {
	expr, needles := BuildFilter(ListFlags{}, "")
	if expr != "true" {
		t.Errorf("expected 'true', got %q", expr)
	}
	if len(needles) != 0 {
		t.Errorf("expected no needles, got %v", needles)
	}
}

func TestBuildFilter_TypeOnly(t *testing.T) {
	expr, _ := BuildFilter(ListFlags{Type: "image"}, "")
	if expr != `(type == "image")` {
		t.Errorf("got %q", expr)
	}
}

func TestBuildFilter_TypeAll_NoFragment(t *testing.T) {
	expr, _ := BuildFilter(ListFlags{Type: "all"}, "")
	if expr != "true" {
		t.Errorf("type=all should emit no type fragment, got %q", expr)
	}
}

func TestBuildFilter_CameraMakeOR(t *testing.T) {
	expr, needles := BuildFilter(ListFlags{CameraMakes: []string{"Sony", "Nikon"}}, "")
	if !strings.Contains(expr, "or") {
		t.Errorf("multi camera_make should OR, got %q", expr)
	}
	if !strings.Contains(expr, "lower(camera_make)") {
		t.Errorf("should case-insensitive, got %q", expr)
	}
	if !strings.Contains(expr, "lower(needle1)") {
		t.Errorf("needle var should also be lowered, got %q", expr)
	}
	if len(needles) != 2 || needles[0] != "Sony" || needles[1] != "Nikon" {
		t.Errorf("needles wrong: %v", needles)
	}
}

func TestBuildFilter_FlagAndUserFilterAND(t *testing.T) {
	expr, _ := BuildFilter(
		ListFlags{DeviceType: "phone", Provinces: []string{"云南"}},
		"iso > 1000",
	)
	if !strings.Contains(expr, "and") {
		t.Errorf("multiple flags + user filter should AND, got %q", expr)
	}
	if !strings.Contains(expr, `device_type == "phone"`) {
		t.Errorf("device fragment missing, got %q", expr)
	}
	if !strings.Contains(expr, "iso > 1000") {
		t.Errorf("user filter not appended, got %q", expr)
	}
}

func TestBuildFilter_SpecialCharNeedleEnvInjection(t *testing.T) {
	// 用户传含单引号的值，translation 必须不字符串拼接（Q1=C）
	_, needles := BuildFilter(ListFlags{CameraMakes: []string{"Sony's Best"}}, "")
	if needles[0] != "Sony's Best" {
		t.Errorf("needle should preserve special chars, got %q", needles[0])
	}
}

func TestBuildFilter_SceneStarrySky(t *testing.T) {
	expr, _ := BuildFilter(ListFlags{Scene: "starry_sky"}, "")
	if !strings.Contains(expr, "scene_starry_sky == true") {
		t.Errorf("got %q", expr)
	}
}

func TestBuildRangeExpr(t *testing.T) {
	cases := []struct {
		field, spec string
		wantExpr    string
		wantOK      bool
	}{
		{"iso", "800", "(iso == 800)", true},
		{"iso", ">800", "(iso > 800)", true},
		{"iso", "<800", "(iso < 800)", true},
		{"iso", ">=800", "(iso >= 800)", true},
		{"iso", "<=800", "(iso <= 800)", true},
		{"iso", "800-1600", "(iso >= 800 and iso <= 1600)", true},
		{"iso", "auto", "", false},
		{"iso", "", "", false},
		{"capture_year", ">=2024", "(capture_year >= 2024)", true},
	}
	for _, c := range cases {
		got, ok := buildRangeExpr(c.field, c.spec)
		if got != c.wantExpr || ok != c.wantOK {
			t.Errorf("buildRangeExpr(%q, %q) = (%q, %v), want (%q, %v)",
				c.field, c.spec, got, ok, c.wantExpr, c.wantOK)
		}
	}
}
