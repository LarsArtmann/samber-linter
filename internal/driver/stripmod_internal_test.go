package driver

import "testing"

func TestStripModTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"vendor only", "-mod=vendor", ""},
		{"mod only", "-mod=mod", ""},
		{"bare mod", "-mod", ""},
		{"preserves other flags", "-mod=vendor -count=1", "-count=1"},
		{"preserves order", "-count=1 -mod=mod -json=1", "-count=1 -json=1"},
		{"no mod flag", "-count=1", "-count=1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := stripModTokens(tc.in); got != tc.want {
				t.Errorf("stripModTokens(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
