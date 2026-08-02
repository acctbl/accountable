package apierror_test

import (
	"testing"

	"github.com/acctbl/accountable/internal/apierror"
)

func TestValidMessageKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key  string
		want bool
	}{
		{"errors.unknown", true},
		{"screen.thing", true},
		{"a.b.c", true},
		{"", false},
		{"nodot", false},
		{".leading", false},
		{"trailing.", false},
		{"has space.x", false},
		{"1bad.start", false},
	}
	for _, tc := range cases {
		if got := apierror.ValidMessageKey(tc.key); got != tc.want {
			t.Fatalf("ValidMessageKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func FuzzValidMessageKey(f *testing.F) {
	f.Add("errors.unknown")
	f.Add("screen.thing")
	f.Add("")
	f.Add("nodot")
	f.Add("a.b.c")

	f.Fuzz(func(t *testing.T, key string) {
		_ = apierror.ValidMessageKey(key)
	})
}
