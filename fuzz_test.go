package bannerlord_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// FuzzSort parses NUL-separated SubModule.xml documents and sorts what parses: every id appears once, comes from the
// input, lands in the order or a cycle, and follows each module its first declaration says loads before it.
func FuzzSort(f *testing.F) {
	files, err := filepath.Glob("testdata/submodule-*.xml")
	if err != nil {
		f.Fatal(err)
	}

	var docs [][]byte

	for _, name := range files {
		b, err := os.ReadFile(name) //nolint:gosec // seeds come from this repo's testdata glob
		if err != nil {
			f.Fatal(err)
		}

		docs = append(docs, b)
	}

	f.Add(bytes.Join(docs, []byte{0}))
	f.Add([]byte(`<Module><Id value="A"/><DependedModules><DependedModule Id="B"/></DependedModules></Module>` + "\x00" +
		`<Module><Id value="B"/><DependedModuleMetadatas><DependedModuleMetadata id="A" order="LoadAfterThis"/></DependedModuleMetadatas></Module>` + "\x00" +
		`<Module><Id value="C"/><Official value="true"/><ModulesToLoadAfterThis><Module Id="A"/></ModulesToLoadAfterThis></Module>`))
	f.Fuzz(func(t *testing.T, b []byte) {
		var mods []bannerlord.Module

		for doc := range bytes.SplitSeq(b, []byte{0}) {
			if m, err := bannerlord.ParseModule(bytes.NewReader(doc)); err == nil {
				mods = append(mods, m)
			}
		}

		first := map[string]bannerlord.Module{}

		for _, m := range mods {
			if _, ok := first[m.ID]; !ok {
				first[m.ID] = m
			}
		}

		order, issues := bannerlord.Sort(mods)
		at := map[string]int{}

		for i, id := range order {
			if _, ok := first[id]; !ok {
				t.Fatalf("order names %q, which no module declares", id)
			}

			if _, dup := at[id]; dup {
				t.Fatalf("order lists %q twice: %v", id, order)
			}

			at[id] = i
		}

		placed := len(order)

		for _, is := range issues {
			if is.Kind == bannerlord.CycleDetected {
				placed += len(is.Modules)
			}
		}

		if placed != len(first) {
			t.Fatalf("%d of %d modules placed: %v %v", placed, len(first), order, issues)
		}

		before := func(a, b string) {
			ia, aok := at[a]
			ib, bok := at[b]

			if a != b && aok && bok && ia > ib {
				t.Fatalf("%s must load before %s: %v", a, b, order)
			}
		}

		for id, m := range first {
			for _, d := range m.Depends {
				before(d.ID, id)
			}

			for _, md := range m.Metadata {
				switch md.Order {
				case bannerlord.LoadBeforeThis:
					before(md.ID, id)
				case bannerlord.LoadAfterThis:
					before(id, md.ID)
				}
			}
		}
	})
}

// FuzzSetMods edits a fuzzed launcher file: on success the enabled ids lead the list, selected and in order, nothing
// after them is selected, and the edit survives a write and a re-read.
func FuzzSetMods(f *testing.F) {
	launcher, err := os.ReadFile("testdata/launcher.xml")
	if err != nil {
		f.Fatal(err)
	}

	f.Add(launcher, "Native,SandBoxCore,Sandbox", false)
	f.Add(launcher, "Bannerlord.Harmony", true)
	f.Add([]byte(`<UserData/>`), "A", false)
	f.Fuzz(func(t *testing.T, b []byte, csv string, multi bool) {
		l, err := bannerlord.ParseLauncher(bytes.NewReader(b))
		if err != nil {
			return
		}

		mode := bannerlord.Singleplayer
		if multi {
			mode = bannerlord.Multiplayer
		}

		var enabled []string
		if csv != "" {
			enabled = strings.Split(csv, ",")
		}

		if l.SetMods(mode, enabled) != nil {
			return
		}

		check := func(l *bannerlord.Launcher) {
			got := l.Mods(mode)
			if len(got) < len(enabled) {
				t.Fatalf("%d mods after enabling %d", len(got), len(enabled))
			}

			for i, m := range got {
				if i < len(enabled) && (m.ID != enabled[i] || !m.Selected) {
					t.Fatalf("mods[%d] = %+v, want %q selected", i, m, enabled[i])
				}

				if i >= len(enabled) && m.Selected {
					t.Fatalf("mods[%d] = %+v is selected but was not enabled", i, m)
				}
			}
		}
		check(l)

		out, err := l.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		back, err := bannerlord.ParseLauncher(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("re-read: %v", err)
		}
		check(back)
	})
}
