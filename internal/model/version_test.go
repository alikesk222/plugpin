package model

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2.1.179", "2.1.179", 0},
		{"2.1.178", "2.1.179", -1},
		{"2.1.180", "2.1.179", 1},
		{"0.146.0", "0.146.0", 0},
		{"0.145.9", "0.146.0", -1},
		{"v2.1.179", "2.1.179", 0},
		{"2.1.179-rc.1", "2.1.179", -1},
		{"2.1", "2.1.0", 0},
		{"2.1.179+build.5", "2.1.179", 0},
		{"2.1.281 (Claude Code)", "2.1.179", 1},
	}
	for _, tc := range cases {
		got, ok := Compare(tc.a, tc.b)
		if !ok || got != tc.want {
			t.Fatalf("Compare(%q, %q) = %d, %v; want %d", tc.a, tc.b, got, ok, tc.want)
		}
	}
	if _, ok := Compare("", "1.0.0"); ok {
		t.Fatal("empty version parsed")
	}
	if _, ok := Compare("not-a-version", "1.0.0"); ok {
		t.Fatal("prose parsed")
	}
}
