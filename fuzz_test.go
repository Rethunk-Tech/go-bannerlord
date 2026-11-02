package bannerlord_test

import (
	"bytes"
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

func FuzzParseModule(f *testing.F) {
	seedXML(f, "testdata/submodule-*.xml")
	f.Fuzz(func(_ *testing.T, b []byte) { _, _ = bannerlord.ParseModule(bytes.NewReader(b)) })
}

func FuzzParseLauncher(f *testing.F) {
	seedXML(f, "testdata/launcher*.xml")
	f.Fuzz(func(_ *testing.T, b []byte) { _, _ = bannerlord.ParseLauncher(bytes.NewReader(b)) })
}
