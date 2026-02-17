package timebucket

import (
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		hour     int
		expected string
	}{
		{"midnight_0", 0, "凌晨(00:00-05:59)"},
		{"early_3", 3, "凌晨(00:00-05:59)"},
		{"dawn_5", 5, "凌晨(00:00-05:59)"},
		{"morning_6", 6, "上午(06:00-10:59)"},
		{"morning_9", 9, "上午(06:00-10:59)"},
		{"morning_10", 10, "上午(06:00-10:59)"},
		{"noon_11", 11, "中午左右(11:00-13:59)"},
		{"noon_12", 12, "中午左右(11:00-13:59)"},
		{"noon_13", 13, "中午左右(11:00-13:59)"},
		{"afternoon_14", 14, "下午(14:00-17:59)"},
		{"afternoon_16", 16, "下午(14:00-17:59)"},
		{"afternoon_17", 17, "下午(14:00-17:59)"},
		{"evening_18", 18, "晚上(18:00-22:59)"},
		{"evening_20", 20, "晚上(18:00-22:59)"},
		{"evening_22", 22, "晚上(18:00-22:59)"},
		{"latenight_23", 23, "半夜(23:00-23:59)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := time.Date(2024, 1, 1, tt.hour, 30, 0, 0, time.Local)
			result := Classify(tm)
			if result != tt.expected {
				t.Errorf("Classify(%d:30) = %q, want %q", tt.hour, result, tt.expected)
			}
		})
	}
}
