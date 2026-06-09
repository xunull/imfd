package output

import (
	"strings"
	"testing"
)

func TestBar_HappyPath(t *testing.T) {
	got := Bar(1, 4)
	// 25% = 5 个 █（向下取整 1*20/4 = 5）
	if !strings.Contains(got, "25%") {
		t.Errorf("want 25%% in output, got %q", got)
	}
	// 5 个 █ + 15 个 ░
	if !strings.HasPrefix(got, "█████░░░░░░░░░░░░░░░") {
		t.Errorf("want 5/20 filled, got %q", got)
	}
}

func TestBar_FullScale(t *testing.T) {
	got := Bar(4, 4)
	if !strings.Contains(got, "100%") {
		t.Errorf("want 100%%, got %q", got)
	}
	if !strings.HasPrefix(got, strings.Repeat("█", 20)) {
		t.Errorf("want all filled, got %q", got)
	}
}

func TestBar_ZeroValue(t *testing.T) {
	got := Bar(0, 4)
	if !strings.Contains(got, "0%") {
		t.Errorf("want 0%%, got %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("zero value should have no filled blocks, got %q", got)
	}
}

func TestBar_NonZeroButRoundsTo0_ForcesOneBlock(t *testing.T) {
	// 1/100 = 0.2 格，整数除法是 0；但 value > 0 应该强制 1 格视觉反馈
	got := Bar(1, 100)
	if strings.Count(got, "█") != 1 {
		t.Errorf("tiny non-zero value should show 1 block, got %q", got)
	}
	if !strings.Contains(got, "  1%") {
		t.Errorf("want 1%%, got %q", got)
	}
}

func TestBar_MaxZero_Defensive(t *testing.T) {
	// max=0 不该 panic（除零保护）
	got := Bar(5, 0)
	if !strings.Contains(got, "0%") {
		t.Errorf("max=0 should yield 0%%, got %q", got)
	}
}

func TestBar_ValueExceedsMax_Clamped(t *testing.T) {
	got := Bar(10, 5)
	if !strings.Contains(got, "100%") {
		t.Errorf("value > max should clamp to 100%%, got %q", got)
	}
}

func TestBar_ASCIIFallback(t *testing.T) {
	t.Setenv("IMFD_ASCII", "1")
	got := Bar(2, 4)
	if strings.Contains(got, "█") || strings.Contains(got, "░") {
		t.Errorf("ASCII mode should not use Unicode block chars, got %q", got)
	}
	if !strings.Contains(got, "#") || !strings.Contains(got, ".") {
		t.Errorf("ASCII mode should use # and ., got %q", got)
	}
}
