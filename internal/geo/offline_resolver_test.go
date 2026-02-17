package geo

import (
	"testing"
)

func TestResolveBeijing(t *testing.T) {
	resolver := NewOfflineResolver()

	// 故宫坐标
	loc, err := resolver.Resolve(39.9163, 116.3972)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Province != "北京" {
		t.Errorf("expected province '北京', got %q", loc.Province)
	}
	if loc.City != "北京" {
		t.Errorf("expected city '北京', got %q", loc.City)
	}
}

func TestResolveShanghai(t *testing.T) {
	resolver := NewOfflineResolver()

	// 外滩坐标
	loc, err := resolver.Resolve(31.2400, 121.4900)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Province != "上海" {
		t.Errorf("expected province '上海', got %q", loc.Province)
	}
}

func TestResolveOverseas(t *testing.T) {
	resolver := NewOfflineResolver()

	// 纽约坐标
	loc, err := resolver.Resolve(40.7128, -74.0060)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Province != "海外" {
		t.Errorf("expected province '海外', got %q", loc.Province)
	}
}

func TestResolveCache(t *testing.T) {
	resolver := NewOfflineResolver()

	loc1, _ := resolver.Resolve(39.9163, 116.3972)
	loc2, _ := resolver.Resolve(39.9163, 116.3972)

	if loc1 != loc2 {
		t.Error("expected same pointer from cache")
	}
}

func TestHaversineDistance(t *testing.T) {
	// 北京到上海约 1068 km
	dist := haversineDistance(39.9042, 116.4074, 31.2304, 121.4737)
	if dist < 1000 || dist > 1200 {
		t.Errorf("expected distance ~1068km, got %.0fkm", dist)
	}
}
