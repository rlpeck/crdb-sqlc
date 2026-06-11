package info

import "testing"

func TestIsReleaseVersion(t *testing.T) {
	cases := map[string]bool{
		"v1.13.6":                true,
		"v0.0.0":                 true,
		"v1.31.1":                true,
		"v1.13.5+dirty":          false, // local dirty build
		"v1.13.5-0.20260611-abc": false, // pseudo-version
		"(devel)":                false,
		"":                       false,
		"1.13.6":                 false, // missing v
		"v1.13":                  false, // not 3 parts
		"v1.13.x":                false, // non-numeric
	}
	for v, want := range cases {
		if got := isReleaseVersion(v); got != want {
			t.Errorf("isReleaseVersion(%q) = %v, want %v", v, got, want)
		}
	}
}
