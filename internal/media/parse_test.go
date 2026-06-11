package media

import "testing"

func TestParseISO(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"800", 800, true},
		{"100", 100, true},
		{"6400", 6400, true},
		{"", 0, false},
		{"auto", 0, false},
		{"ISO 800", 0, false},
		{" 800 ", 800, true},
		{"0", 0, true},
	}
	for _, c := range cases {
		got, ok := ParseISO(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseISO(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseAperture(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"f/5", 5.0, true},
		{"F/5", 5.0, true},
		{"f/2.8", 2.8, true},
		{"f/22", 22.0, true},
		{"5", 5.0, true},
		{"5.0", 5.0, true},
		{"", 0, false},
		{"f/abc", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseAperture(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseAperture(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseShutter(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"1/250s", 0.004, true},
		{"1/4000", 0.00025, true},
		{"30s", 30.0, true},
		{"30", 30.0, true},
		{"0.5s", 0.5, true},
		{"", 0, false},
		{"auto", 0, false},
		{"1/0", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseShutter(c.in)
		if (got-c.want) > 1e-9 || (c.want-got) > 1e-9 || ok != c.ok {
			t.Errorf("ParseShutter(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseFocal(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"42mm", 42.0, true},
		{"85.5mm", 85.5, true},
		{"50", 50.0, true},
		{"42MM", 42.0, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseFocal(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseFocal(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
