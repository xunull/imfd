package output

import (
	"strings"
	"testing"

	"github.com/xunull/imfd/internal/media"
)

func TestColorer_DisabledReturnsPlain(t *testing.T) {
	c := NewColorer(false)
	cases := []string{
		c.Bold("hi"),
		c.Dim("hi"),
		c.SectionHeader("OVERVIEW"),
		c.Media(media.TypeAudio, "mp3"),
	}
	for _, got := range cases {
		if strings.Contains(got, "\x1b[") {
			t.Errorf("disabled colorer leaked ANSI: %q", got)
		}
	}
}

func TestColorer_EnabledWrapsWithANSI(t *testing.T) {
	c := NewColorer(true)
	if !strings.Contains(c.Bold("X"), "\x1b[1m") {
		t.Error("Bold should emit ANSI bold")
	}
	if !strings.HasSuffix(c.Bold("X"), "\x1b[0m") {
		t.Error("Bold should reset")
	}
}

func TestColorer_MediaTypeColors(t *testing.T) {
	c := NewColorer(true)
	cases := []struct {
		t    media.MediaType
		want string
	}{
		{media.TypeImage, "\x1b[34m"},
		{media.TypeVideo, "\x1b[35m"},
		{media.TypeAudio, "\x1b[32m"},
	}
	for _, c2 := range cases {
		got := c.Media(c2.t, "x")
		if !strings.Contains(got, c2.want) {
			t.Errorf("type %v: want %q in %q", c2.t, c2.want, got)
		}
	}
}

func TestColorer_EmptyStringNotWrapped(t *testing.T) {
	c := NewColorer(true)
	if got := c.Bold(""); got != "" {
		t.Errorf("empty string should pass through unchanged, got %q", got)
	}
}

func TestColorer_UnknownMediaType(t *testing.T) {
	c := NewColorer(true)
	// TypeUnknown 应当原样返回（保守降级）
	got := c.Media(media.TypeUnknown, "x")
	if strings.Contains(got, "\x1b[") {
		t.Errorf("unknown type should not be colored, got %q", got)
	}
}

func TestNoColor_EnvVar(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("IMFD_NO_COLOR", "")
	if !NoColor() {
		t.Error("NO_COLOR=1 should disable color")
	}
}

func TestNoColor_IMFDVariant(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("IMFD_NO_COLOR", "1")
	if !NoColor() {
		t.Error("IMFD_NO_COLOR=1 should disable color")
	}
}

func TestNoColor_Default(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("IMFD_NO_COLOR", "")
	if NoColor() {
		t.Error("default should be color enabled")
	}
}
