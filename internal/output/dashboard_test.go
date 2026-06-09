package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xunull/imfd/internal/stats"
)

// makeReport 构造一份用于 dashboard 测试的 fake StatsReport
func makeReport(audioBuckets ...stats.Bucket) stats.StatsReport {
	report := stats.StatsReport{
		Totals: stats.Totals{
			ImageCount: 1,
			VideoCount: 1,
			AudioCount: 3,
			TotalCount: 5,
		},
	}

	// 媒体类型（OVERVIEW 内部不读这个，但 sectionDefs 里挂在 OVERVIEW）
	report.Dimensions = append(report.Dimensions, stats.DimensionResult{
		DimensionName: "媒体类型",
		Buckets: []stats.Bucket{
			{Key: "image", Count: 1},
			{Key: "video", Count: 1},
			{Key: "audio", Count: 3},
		},
	})

	// DEVICE：相机型号 = 全 Unknown（应该折叠）
	report.Dimensions = append(report.Dimensions, stats.DimensionResult{
		DimensionName: "相机型号",
		Buckets:       []stats.Bucket{{Key: "Unknown", Count: 5}},
	})

	// EXIF：ISO感光度 = 全 Unknown
	report.Dimensions = append(report.Dimensions, stats.DimensionResult{
		DimensionName: "ISO感光度",
		Buckets:       []stats.Bucket{{Key: "Unknown", Count: 5}},
	})

	// AUDIO：编解码器
	if len(audioBuckets) > 0 {
		report.Dimensions = append(report.Dimensions, stats.DimensionResult{
			DimensionName: "音频编解码器",
			Buckets:       audioBuckets,
		})
	}

	return report
}

func runDashboard(t *testing.T, verbose bool, report stats.StatsReport) string {
	t.Helper()
	var buf bytes.Buffer
	p := NewDashboardPrinter(&buf, verbose, "/tmp/test", 50*time.Millisecond)
	// 强制关闭颜色让断言简单（不必匹配 ANSI escape）
	p.color = NewColorer(false)
	if err := p.Print(report); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	return buf.String()
}

func TestDashboard_HeaderShowsMetaInfo(t *testing.T) {
	out := runDashboard(t, false, makeReport(
		stats.Bucket{Key: "mp3", Count: 3},
		stats.Bucket{Key: "flac", Count: 1},
	))

	for _, want := range []string{
		"imfd",
		"/tmp/test",
		"scanned 5 files",
		"50ms",
		"0 errors",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q\n---OUTPUT---\n%s", want, out)
		}
	}
}

func TestDashboard_OverviewShowsAllThreeTypes(t *testing.T) {
	out := runDashboard(t, false, makeReport())
	for _, want := range []string{"OVERVIEW", "图像", "视频", "音频", "总计"} {
		if !strings.Contains(out, want) {
			t.Errorf("overview missing %q", want)
		}
	}
}

func TestDashboard_EmptyDimsCollapsedByDefault(t *testing.T) {
	out := runDashboard(t, false, makeReport(stats.Bucket{Key: "mp3", Count: 3}))

	// 相机型号 + ISO感光度 都是全 Unknown，应该被折叠提示而非展开
	if strings.Contains(out, "相机型号  ") || strings.Contains(out, "相机型号 Unknown") {
		// 这里检测真的"展开"形式（dim 名 + 桶值），而非折叠提示里的名字
		// 真正的字段名行会带具体桶 key 或 count；折叠提示行只是名字字符串
		// 用 "无数据" 兜一下：如果出现 "(N 维度无数据" 字样就是折叠形态
		if !strings.Contains(out, "维度无数据") {
			t.Error("expected DEVICE/EXIF dims to be collapsed when no -v")
		}
	}

	if !strings.Contains(out, "维度无数据") {
		t.Errorf("expected collapse hint, got:\n%s", out)
	}
}

func TestDashboard_VerboseExpandsEmptyDims(t *testing.T) {
	out := runDashboard(t, true, makeReport(stats.Bucket{Key: "mp3", Count: 3}))

	// -v 模式应该把"相机型号 Unknown 5"那条真展开出来
	if strings.Contains(out, "维度无数据") {
		t.Errorf("verbose mode should NOT collapse, got:\n%s", out)
	}
	if !strings.Contains(out, "相机型号") {
		t.Error("verbose mode should print 相机型号 dim")
	}
}

func TestDashboard_UnknownBucketSortedLast(t *testing.T) {
	out := runDashboard(t, false, makeReport(
		stats.Bucket{Key: "Unknown", Count: 100}, // 故意 Unknown count 最高
		stats.Bucket{Key: "mp3", Count: 3},
		stats.Bucket{Key: "flac", Count: 1},
	))

	// 找 AUDIO section，断言 mp3 出现在 Unknown 之前
	audioIdx := strings.Index(out, "AUDIO")
	if audioIdx < 0 {
		t.Fatal("AUDIO section missing")
	}
	body := out[audioIdx:]
	mp3Idx := strings.Index(body, "mp3")
	unknownIdx := strings.Index(body, "Unknown")
	if mp3Idx < 0 || unknownIdx < 0 {
		t.Fatalf("missing mp3 or Unknown bucket:\n%s", body)
	}
	if mp3Idx > unknownIdx {
		t.Errorf("Unknown should sort last; mp3 at %d, Unknown at %d", mp3Idx, unknownIdx)
	}
}

func TestDashboard_NoANSIWhenColorerDisabled(t *testing.T) {
	out := runDashboard(t, false, makeReport(stats.Bucket{Key: "mp3", Count: 1}))
	// 没有 ANSI escape 字符（\x1b[）
	if strings.Contains(out, "\x1b[") {
		t.Errorf("expected no ANSI when colorer disabled, got escape in output:\n%q", out)
	}
}

func TestDashboard_CmdLabelInferredFromTotals(t *testing.T) {
	tests := []struct {
		name  string
		image int
		video int
		audio int
		want  string
	}{
		{"audio only", 0, 0, 5, "scan audio"},
		{"image only", 5, 0, 0, "scan image"},
		{"video only", 0, 5, 0, "scan video"},
		{"mixed", 1, 1, 1, "scan"},
		{"all zero (edge)", 0, 0, 0, "scan"},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewDashboardPrinter(&buf, false, "/x", 0)
			p.color = NewColorer(false)
			rep := stats.StatsReport{
				Totals: stats.Totals{
					ImageCount: c.image,
					VideoCount: c.video,
					AudioCount: c.audio,
					TotalCount: c.image + c.video + c.audio,
				},
			}
			if err := p.Print(rep); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), c.want) {
				t.Errorf("want header contain %q, got:\n%s", c.want, buf.String())
			}
		})
	}
}

func TestDashboard_DurationFormatting(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "—"},
		{500 * time.Microsecond, "500µs"},
		{30 * time.Millisecond, "30ms"},
		{2500 * time.Millisecond, "2.50s"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewDashboardPrinter(&buf, false, "/x", c.d)
			p.color = NewColorer(false)
			if err := p.Print(stats.StatsReport{}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), c.want) {
				t.Errorf("duration %v: want %q in header, got:\n%s", c.d, c.want, buf.String())
			}
		})
	}
}

func TestIsEmptyDim(t *testing.T) {
	cases := []struct {
		name string
		dim  stats.DimensionResult
		want bool
	}{
		{"all unknown", stats.DimensionResult{Buckets: []stats.Bucket{{Key: "Unknown", Count: 3}}}, true},
		{"empty buckets", stats.DimensionResult{Buckets: nil}, true},
		{"has real data", stats.DimensionResult{Buckets: []stats.Bucket{{Key: "mp3", Count: 1}}}, false},
		{"unknown + real", stats.DimensionResult{Buckets: []stats.Bucket{{Key: "Unknown", Count: 5}, {Key: "mp3", Count: 1}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isEmptyDim(c.dim); got != c.want {
				t.Errorf("want %v, got %v", c.want, got)
			}
		})
	}
}
