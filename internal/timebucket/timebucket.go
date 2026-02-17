package timebucket

import "time"

// TimeBucket 时间段
type TimeBucket struct {
	Name      string
	StartHour int // 包含
	EndHour   int // 包含
}

// DefaultBuckets 默认时间段规则
var DefaultBuckets = []TimeBucket{
	{Name: "凌晨(00:00-05:59)", StartHour: 0, EndHour: 5},
	{Name: "上午(06:00-10:59)", StartHour: 6, EndHour: 10},
	{Name: "中午左右(11:00-13:59)", StartHour: 11, EndHour: 13},
	{Name: "下午(14:00-17:59)", StartHour: 14, EndHour: 17},
	{Name: "晚上(18:00-22:59)", StartHour: 18, EndHour: 22},
	{Name: "半夜(23:00-23:59)", StartHour: 23, EndHour: 23},
}

// Classify 根据时间判定所属时间段
func Classify(t time.Time) string {
	return ClassifyWithBuckets(t, DefaultBuckets)
}

// ClassifyWithBuckets 使用自定义时间段规则进行分类
func ClassifyWithBuckets(t time.Time, buckets []TimeBucket) string {
	hour := t.Hour()
	for _, b := range buckets {
		if hour >= b.StartHour && hour <= b.EndHour {
			return b.Name
		}
	}
	return "Unknown"
}
