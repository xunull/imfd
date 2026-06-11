package output

import (
	"bytes"
	"testing"
)

func TestListPrinter_Plain(t *testing.T) {
	var buf bytes.Buffer
	p := NewListPrinter(&buf, false)
	for _, path := range []string{"/a.jpg", "/b.jpg"} {
		if err := p.Print(path); err != nil {
			t.Fatal(err)
		}
	}
	got := buf.String()
	want := "/a.jpg\n/b.jpg\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestListPrinter_NUL(t *testing.T) {
	var buf bytes.Buffer
	p := NewListPrinter(&buf, true)
	for _, path := range []string{"/a.jpg", "/b\nwith-newline.jpg"} {
		if err := p.Print(path); err != nil {
			t.Fatal(err)
		}
	}
	got := buf.String()
	want := "/a.jpg\x00/b\nwith-newline.jpg\x00"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
