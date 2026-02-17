package geo

import (
	"testing"
)

func TestParseGeoProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected GeoProvider
		wantErr  bool
	}{
		{"offline", ProviderOffline, false},
		{"nominatim", ProviderNominatim, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseGeoProvider(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseGeoProvider(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewResolverOffline(t *testing.T) {
	resolver, err := NewResolver(ProviderOffline)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolver.Name() != "offline" {
		t.Errorf("expected name 'offline', got %q", resolver.Name())
	}

	// 验证接口可用
	loc, err := resolver.Resolve(39.9042, 116.4074)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Province != "北京" {
		t.Errorf("expected '北京', got %q", loc.Province)
	}
}

func TestNewResolverNominatim(t *testing.T) {
	resolver, err := NewResolver(ProviderNominatim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolver.Name() != "nominatim" {
		t.Errorf("expected name 'nominatim', got %q", resolver.Name())
	}
}

func TestNewResolverInvalid(t *testing.T) {
	_, err := NewResolver(GeoProvider("nonexistent"))
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}

// 验证 OfflineResolver 和 NominatimResolver 都实现了 GeoResolver 接口
func TestInterfaceCompliance(t *testing.T) {
	var _ GeoResolver = (*OfflineResolver)(nil)
	var _ GeoResolver = (*NominatimResolver)(nil)
}
