package bannerlord_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	bannerlord "github.com/Rethunk-Tech/go-bannerlord"
)

func seedXML(f *testing.F, pattern string) {
	f.Helper()

	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		f.Fatalf("no seeds for %s: %v", pattern, err)
	}

	for _, name := range files {
		b, err := os.ReadFile(name) //nolint:gosec // seeds come from this repo's testdata glob
		if err != nil {
			f.Fatal(err)
		}

		f.Add(b)
	}
}

func FuzzParseVersion(f *testing.F) {
	for _, s := range []string{
		"",
		"v1.2.3.0",
		"e1.0.0",
		"v1.2.*",
		"a0.0.0",
		"b1.2.3.4",
		"d9.9.9",
		"v1.2.3",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		v, err := bannerlord.ParseVersion(s)
		if err != nil {
			return
		}

		got, err := bannerlord.ParseVersion(v.String())
		if err != nil {
			t.Fatalf("round-trip parse %q -> %q: %v", s, v.String(), err)
		}

		if v.Compare(got) != 0 {
			t.Fatalf("round-trip %q -> %q changed the version", s, v.String())
		}
	})
}

func FuzzParseModule(f *testing.F) {
	seedXML(f, "testdata/submodule-*.xml")
	f.Fuzz(func(t *testing.T, b []byte) {
		_, err := bannerlord.ParseModule(bytes.NewReader(b))
		if len(b) > bannerlord.MaxXMLSize && !errors.Is(err, bannerlord.ErrTooLarge) {
			t.Fatalf("len=%d: want ErrTooLarge, got %v", len(b), err)
		}
	})
}

func FuzzParseLauncher(f *testing.F) {
	seedXML(f, "testdata/launcher*.xml")
	f.Fuzz(func(t *testing.T, b []byte) {
		l, err := bannerlord.ParseLauncher(bytes.NewReader(b))
		if len(b) > bannerlord.MaxXMLSize && !errors.Is(err, bannerlord.ErrTooLarge) {
			t.Fatalf("len=%d: want ErrTooLarge, got %v", len(b), err)
		}

		if err != nil {
			return
		}

		out, err := l.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		l2, err := bannerlord.ParseLauncher(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("round-trip parse: %v", err)
		}

		for _, mode := range []bannerlord.Mode{bannerlord.Singleplayer, bannerlord.Multiplayer} {
			a, bmods := l.Mods(mode), l2.Mods(mode)
			if len(a) != len(bmods) {
				t.Fatalf("%s mods %d -> %d", mode, len(a), len(bmods))
			}

			for i := range a {
				if a[i] != bmods[i] {
					t.Fatalf("%s mods[%d] %+v -> %+v", mode, i, a[i], bmods[i])
				}
			}
		}
	})
}
