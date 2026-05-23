package v5common

import "testing"

func TestDec(t *testing.T) {
	t.Parallel()
	type tc struct {
		in   string
		want string
	}
	var cases []tc = []tc{
		{"", "0"},
		{"0", "0"},
		{"0.001", "0.001"},
		{"123456.789", "123456.789"},
		{"not-a-number", "0"},
		{"1e2", "100"},
	}
	for _, c := range cases {
		var got = Dec(c.in)
		if got.String() != c.want {
			t.Errorf("Dec(%q) = %s, want %s", c.in, got.String(), c.want)
		}
	}
}

func TestMs(t *testing.T) {
	t.Parallel()
	type tc struct {
		in   string
		want int64
	}
	var cases []tc = []tc{
		{"", 0},
		{"0", 0},
		{"1700000000000", 1700000000000},
		{"abc", 0},
		{"-1", -1},
	}
	for _, c := range cases {
		var got = Ms(c.in)
		if got != c.want {
			t.Errorf("Ms(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizeRejectReason(t *testing.T) {
	t.Parallel()
	type tc struct {
		in   string
		want string
	}
	var cases []tc = []tc{
		{"EC_NoError", ""},
		{"", ""},
		{"EC_PostOnlyWillTakeLiquidity", "EC_PostOnlyWillTakeLiquidity"},
		{"foo", "foo"},
	}
	for _, c := range cases {
		var got = NormalizeRejectReason(c.in)
		if got != c.want {
			t.Errorf("NormalizeRejectReason(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
