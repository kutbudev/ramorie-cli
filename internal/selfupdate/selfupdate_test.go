package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"9.1.0", "9.2.0", true},
		{"9.1.0", "9.1.1", true},
		{"9.1.0", "10.0.0", true},
		{"v9.1.0", "v9.2.0", true},
		{"9.2.0", "9.2.0", false},
		{"9.2.0", "9.1.9", false},
		{"9.2.0", "9.1.10", false},
		{"10.0.0", "9.9.9", false},
		{"9.2.0", "9.2.0-rc1", false}, // pre-release of same version is not newer
		{"9.1.0", "9.2.0-rc1", true},
		{"0.0.0", "9.1.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestMethodString(t *testing.T) {
	for m, want := range map[Method]string{
		MethodBinary: "binary",
		MethodBrew:   "homebrew",
		MethodNpm:    "npm",
	} {
		if got := m.String(); got != want {
			t.Errorf("Method(%d).String() = %q, want %q", m, got, want)
		}
	}
}
