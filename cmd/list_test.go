package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// resetListFlags 重置 list flag 状态防止子测试间互相污染
func resetListFlags(t *testing.T) {
	t.Helper()
	flagListType = "all"
	flagListCameraMakes = nil
	flagListCameraModels = nil
	flagListDevice = ""
	flagListProvinces = nil
	flagListCities = nil
	flagListScene = ""
	flagListISO = ""
	flagListYear = ""
	flagListFilter = ""
	flagListEdited = false
	flagListOOC = false
	flagListAI = false
	flagListNotAI = false
	flagListPrint0 = false
	flagListNoCache = true // 测试不依赖 cache；隔离副作用
	flagListWorkers = 8
	flagListExtractors = 0
	flagListChannelSize = 1024
	flagListGeoProvider = "offline"

	// 重置 cobra flag.Changed 状态——多次 Execute 之间 cobra 把 flag 视为 sticky
	// 不重置会让 mutex 检查在第二次 Execute 时误报
	listCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
}

type listCall struct {
	paths []string
}

func withFakeListRunner(t *testing.T) (*listCall, func()) {
	t.Helper()
	orig := listRunner
	cap := &listCall{}
	listRunner = func(paths []string, stdout, stderr io.Writer) error {
		cap.paths = paths
		return nil
	}
	return cap, func() { listRunner = orig }
}

func TestListRouting_SingleArg(t *testing.T) {
	resetListFlags(t)
	cap, restore := withFakeListRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"list", "/some/dir"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if len(cap.paths) != 1 || cap.paths[0] != "/some/dir" {
		t.Errorf("got %v", cap.paths)
	}
}

func TestListRouting_AllFlagsRouted(t *testing.T) {
	resetListFlags(t)
	_, restore := withFakeListRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{
		"list",
		"--type", "image",
		"--camera-make", "Sony",
		"--camera-make", "Nikon",
		"--device", "phone",
		"--province", "云南",
		"--scene", "starry_sky",
		"--iso", ">800",
		"--year", "2024",
		"--filter", "true",
		"-0",
		"/x",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if flagListType != "image" {
		t.Errorf("type: %s", flagListType)
	}
	if len(flagListCameraMakes) != 2 {
		t.Errorf("camera-makes: %v", flagListCameraMakes)
	}
	if flagListDevice != "phone" {
		t.Errorf("device: %s", flagListDevice)
	}
	if !flagListPrint0 {
		t.Error("-0 not set")
	}
	if flagListFilter != "true" {
		t.Errorf("filter: %s", flagListFilter)
	}
}

func TestListRouting_NoArgsOK(t *testing.T) {
	// path arg 是可选；runList 自己默认 "."
	resetListFlags(t)
	_, restore := withFakeListRunner(t)
	defer restore()

	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list 无 path 应允许: %v", err)
	}
}

func TestRunList_PathNotFound(t *testing.T) {
	resetListFlags(t)
	var stdout, stderr bytes.Buffer
	err := runList([]string{"/not/exist/zzz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr should print 'error:', got %q", stderr.String())
	}
}

func TestRunList_PathIsFile(t *testing.T) {
	resetListFlags(t)
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "somefile.txt")
	if err := os.WriteFile(tmpFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := runList([]string{tmpFile}, &stdout, &stderr)
	if err == nil {
		t.Error("expected error when path is a file (not dir)")
	}
	if !strings.Contains(stderr.String(), "imfd info") {
		t.Errorf("error should hint at 'imfd info', got %q", stderr.String())
	}
}

func TestRunList_EmptyDirReturnsNothing(t *testing.T) {
	resetListFlags(t)
	dir := t.TempDir() // 空目录
	var stdout, stderr bytes.Buffer
	err := runList([]string{dir}, &stdout, &stderr)
	if err != nil {
		t.Errorf("empty dir should not error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("empty dir stdout should be empty, got %q", stdout.String())
	}
}

func TestRunList_SyntaxErrorOnBadFilter(t *testing.T) {
	// 这个测试要小心：runList 在 SyntaxError 时调 os.Exit(2)
	// 我们绕开这个 path，确认 query.NewEvaluator 报错被传递
	// （runList 调 os.Exit 之前已 stderr print）
	// 此测试只验证 stderr 输出存在
	resetListFlags(t)
	flagListFilter = "iso >>> 800"
	var stdout, stderr bytes.Buffer
	// runList 会 os.Exit(2)，但 test 进程不能让它 exit
	// 改用直接调 query.NewEvaluator 验证：这部分已经在 query/eval_test.go 覆盖
	_ = stdout
	_ = stderr
	// skip 这个 path 的实测；query 包已有 syntax error 测试
	t.Skip("syntax error path 走 os.Exit；query/eval_test.go 已覆盖 NewEvaluator 报错")
}

// TestListEditedOOCMutex 验证 --edited 和 --ooc 同时给会被 cobra 拦截。
// 走 rootCmd.Execute() 而不是 runList，因为 mutex 校验是 cobra 在 RunE 前做的。
func TestListEditedOOCMutex(t *testing.T) {
	resetListFlags(t)
	_, restore := withFakeListRunner(t)
	defer restore()

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"list", "--edited", "--ooc", "/x"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for mutually exclusive --edited / --ooc")
	}
	// cobra's MarkFlagsMutuallyExclusive 错误文本是：
	// "if any flags in the group [edited ooc] are set none of the others can be; [edited ooc] were all set"
	// 匹配 "edited" 和 "ooc" 共存即可，不绑定具体话术
	if !strings.Contains(err.Error(), "edited") || !strings.Contains(err.Error(), "ooc") {
		t.Errorf("error should mention both 'edited' and 'ooc' flag names, got: %v", err)
	}
}

// TestListEditedAlone 单独 --edited 应该正常通过到 runner
func TestListEditedAlone(t *testing.T) {
	resetListFlags(t)
	_, restore := withFakeListRunner(t)
	defer restore()

	rootCmd.SetArgs([]string{"list", "--edited", "/x"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--edited alone should not error: %v", err)
	}
	if !flagListEdited {
		t.Error("flagListEdited should be true")
	}
	if flagListOOC {
		t.Error("flagListOOC should be false")
	}
}
