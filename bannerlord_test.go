package bannerlord_test

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	bannerlord "github.com/Rethunk-Tech/go-bannerlord"
)

func open(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open("testdata/" + name) //nolint:gosec // name is a constant fixture in this file
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestVersionCompare(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3.0", "v1.2.3", 0},
		{"v1.2.3", "v1.2.4", -1},
		{"v1.10.0", "v1.9.9", 1},
		{"e1.0.0", "v1.0.0", -1},
		{"v1.2.*", "v1.2.0", 0},
		{"v1.2.0.1", "v1.2.*", 1},
	}
	for _, c := range cases {
		a, err := bannerlord.ParseVersion(c.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := bannerlord.ParseVersion(c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Compare(b); got != c.want {
			t.Errorf("%s vs %s = %d, want %d", c.a, c.b, got, c.want)
		}
	}
	for _, bad := range []string{"", "v", "1.2.3", "x1.2.3", "v1.2", "v1.*", "v1.2.3.4.5", "v-1", "v1..2", "v1.99999999999"} {
		if _, err := bannerlord.ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q) accepted", bad)
		}
	}
	if v, _ := bannerlord.ParseVersion("v1.2.3"); v.String() != "v1.2.3" {
		t.Errorf("String = %q", v.String())
	}
}

func TestParseModule(t *testing.T) {
	t.Parallel()
	m, err := bannerlord.ParseModule(open(t, "submodule-full.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "Example.Mod" || m.Name != "Example Mod" || m.Version.String() != "v1.2.3.0" || m.Official || m.Default {
		t.Errorf("header = %+v", m)
	}
	if len(m.Depends) != 2 || m.Depends[0].ID != "Native" || m.Depends[0].Version.String() != "v1.2.0" || !m.Depends[1].Optional {
		t.Errorf("depends = %+v", m.Depends)
	}
	if len(m.Metadata) != 5 || m.Metadata[0].Order != bannerlord.LoadAfterThis || !m.Metadata[2].Incompatible {
		t.Errorf("metadata = %+v", m.Metadata)
	}
	if !reflect.DeepEqual(m.DLLs, []string{"Example.Mod.dll"}) {
		t.Errorf("dlls = %v", m.DLLs)
	}

	if m.Metadata[3] != (bannerlord.Metadata{ID: "Native", Order: bannerlord.LoadAfterThis, Optional: true}) || m.Metadata[4].ID != "Bad.Two" || !m.Metadata[4].Incompatible {
		t.Errorf("module lists = %+v", m.Metadata[3:])
	}

	n, err := bannerlord.ParseModule(open(t, "submodule-bom.xml"))
	if err != nil || !n.Official || !n.Default {
		t.Errorf("bom module = %+v, %v", n, err)
	}
}

func TestParseModuleRefuses(t *testing.T) {
	t.Parallel()
	if _, err := bannerlord.ParseModule(open(t, "submodule-doctype.xml")); err == nil {
		t.Error("DOCTYPE accepted")
	}
	if _, err := bannerlord.ParseModule(open(t, "submodule-badid.xml")); err == nil {
		t.Error("bad id accepted")
	}
	tp := `<Module><Id value="T"/><ModuleType value="OfficialOptional"/></Module>`
	if m, err := bannerlord.ParseModule(strings.NewReader(tp)); err != nil || !m.Official {
		t.Errorf("ModuleType module = %+v, %v", m, err)
	}
	big := "<Module><Id value=\"X\"/><!--" + strings.Repeat("a", bannerlord.MaxXMLSize) + "--></Module>"
	if _, err := bannerlord.ParseModule(strings.NewReader(big)); err == nil {
		t.Error("oversized document accepted")
	}
	deep := strings.Repeat("<a>", 100) + strings.Repeat("</a>", 100)
	if _, err := bannerlord.ParseModule(strings.NewReader(deep)); err == nil {
		t.Error("deep nesting accepted")
	}
}

func mod(id string, deps ...string) bannerlord.Module {
	m := bannerlord.Module{ID: id}
	for _, d := range deps {
		m.Depends = append(m.Depends, bannerlord.Dependency{ID: d})
	}
	return m
}

func official(id string, deps ...string) bannerlord.Module {
	m := mod(id, deps...)
	m.Official = true
	return m
}

func TestSort(t *testing.T) {
	t.Parallel()
	after := mod("A")
	after.Metadata = []bannerlord.Metadata{{ID: "Z", Order: bannerlord.LoadAfterThis}}
	before := mod("B")
	before.Metadata = []bannerlord.Metadata{{ID: "Y", Order: bannerlord.LoadBeforeThis}}
	missing := mod("M", "Nope")
	optional := mod("O")
	optional.Depends = []bannerlord.Dependency{{ID: "Nope", Optional: true}}
	old := mod("Old", "Lib")
	old.Depends[0].Version, _ = bannerlord.ParseVersion("v2.0.0")
	lib := mod("Lib")
	lib.Version, _ = bannerlord.ParseVersion("v1.0.0")
	bad := mod("Bad")
	bad.Metadata = []bannerlord.Metadata{{ID: "Lib", Incompatible: true}}

	cases := []struct {
		name   string
		in     []bannerlord.Module
		order  []string
		issues []bannerlord.IssueKind
	}{
		{
			"official first",
			[]bannerlord.Module{mod("Aaa"), official("StoryMode", "SandBox"), official("SandBox", "Native"), official("Native")},
			[]string{"Native", "SandBox", "StoryMode", "Aaa"},
			nil,
		},
		{"dependency", []bannerlord.Module{mod("B", "A"), mod("A"), mod("C")}, []string{"A", "B", "C"}, nil},
		{"load after", []bannerlord.Module{after, mod("Z")}, []string{"A", "Z"}, nil},
		{"load before", []bannerlord.Module{before, mod("Y")}, []string{"Y", "B"}, nil},
		{"cycle", []bannerlord.Module{mod("A", "B"), mod("B", "A"), mod("C", "A"), mod("D")}, []string{"D"}, []bannerlord.IssueKind{bannerlord.CycleDetected}},
		{"missing dependency", []bannerlord.Module{missing, optional}, []string{"M", "O"}, []bannerlord.IssueKind{bannerlord.MissingDependency}},
		{"outdated", []bannerlord.Module{old, lib}, []string{"Lib", "Old"}, []bannerlord.IssueKind{bannerlord.OutdatedDependency}},
		{"incompatible", []bannerlord.Module{bad, lib}, []string{"Bad", "Lib"}, []bannerlord.IssueKind{bannerlord.IncompatibleModule}},
		{"duplicate", []bannerlord.Module{mod("A"), mod("A")}, []string{"A"}, []bannerlord.IssueKind{bannerlord.DuplicateModule}},
	}
	for _, c := range cases {
		order, issues := bannerlord.Sort(c.in)
		if !reflect.DeepEqual(order, c.order) {
			t.Errorf("%s: order = %v, want %v", c.name, order, c.order)
		}
		var kinds []bannerlord.IssueKind
		for _, i := range issues {
			kinds = append(kinds, i.Kind)
			if i.String() == "" {
				t.Errorf("%s: empty issue text", c.name)
			}
		}
		if !reflect.DeepEqual(kinds, c.issues) {
			t.Errorf("%s: issues = %v, want %v", c.name, kinds, c.issues)
		}
	}
}

func TestLauncherRoundTrip(t *testing.T) {
	t.Parallel()
	l, err := bannerlord.ParseLauncher(open(t, "launcher.xml"))
	if err != nil {
		t.Fatal(err)
	}
	want := []bannerlord.LauncherMod{{ID: "Native", Selected: true}, {ID: "Old.Mod", Selected: true}}
	if got := l.Mods(bannerlord.Singleplayer); !reflect.DeepEqual(got, want) {
		t.Errorf("mods = %v", got)
	}
	if err := l.SetMods(bannerlord.Singleplayer, []string{"New.Mod", "Native"}); err != nil {
		t.Fatal(err)
	}
	if err := l.SetMods(bannerlord.Multiplayer, []string{"Native"}); err != nil {
		t.Fatal(err)
	}
	out, err := l.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`<Unknown attr="kept">data</Unknown>`, "<DelayedStart>false</DelayedStart>", "<LastKnownVersion>v1.2.9</LastKnownVersion>"} {
		if !strings.Contains(string(out), s) {
			t.Errorf("lost %s in\n%s", s, out)
		}
	}
	back, err := bannerlord.ParseLauncher(strings.NewReader(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	want = []bannerlord.LauncherMod{{ID: "New.Mod", Selected: true}, {ID: "Native", Selected: true}, {ID: "Old.Mod", Selected: false}}
	if got := back.Mods(bannerlord.Singleplayer); !reflect.DeepEqual(got, want) {
		t.Errorf("after = %v", got)
	}
	if got := back.Mods(bannerlord.Multiplayer); !reflect.DeepEqual(got, []bannerlord.LauncherMod{{ID: "Native", Selected: true}}) {
		t.Errorf("multiplayer = %v", got)
	}
	if err := back.SetMods(bannerlord.Singleplayer, []string{"a/b"}); err == nil {
		t.Error("bad id accepted")
	}
	if err := back.SetMods(bannerlord.Singleplayer, []string{"A", "A"}); err == nil {
		t.Error("duplicate id accepted")
	}
	if _, err := bannerlord.ParseLauncher(strings.NewReader("<UserData><SingleplayerData><ModDatas><UserModData><Id>../x</Id></UserModData></ModDatas></SingleplayerData></UserData>")); err == nil {
		t.Error("bad launcher id accepted")
	}
}

func TestDotIDsAreRefused(t *testing.T) {
	t.Parallel()
	for _, id := range []string{".", "..", "CON", "nul", "COM1", "CON.txt", "M."} {
		doc := `<Module><Id value="` + id + `"/></Module>`
		if _, err := bannerlord.ParseModule(strings.NewReader(doc)); err == nil {
			t.Errorf("module id %q accepted", id)
		}
		dep := `<Module><Id value="M"/><DependedModules><DependedModule Id="` + id + `"/></DependedModules></Module>`
		if _, err := bannerlord.ParseModule(strings.NewReader(dep)); err == nil {
			t.Errorf("dependency id %q accepted", id)
		}
	}
}

func TestDLLNamesMustBePlainFileNames(t *testing.T) {
	t.Parallel()
	for _, dll := range []string{"D:evil.dll", ":ads.dll", "a.dll:stream", "CON.dll", "nul", "com1.x.dll", "COM0", "lpt0.dll", "COM¹.dll", "LPT³", "CONIN$", "conout$.dll", "a.dll.", "a.dll ", "..", "a/b.dll", `a\b.dll`} {
		doc := `<Module><Id value="M"/><SubModules><SubModule><DLLName value="` + dll + `"/></SubModule></SubModules></Module>`
		if _, err := bannerlord.ParseModule(strings.NewReader(doc)); err == nil {
			t.Errorf("DLLName %q accepted", dll)
		}
	}
	ok := `<Module><Id value="M"/><SubModules><SubModule><DLLName value="Console.Mod.dll"/></SubModule></SubModules></Module>`
	if _, err := bannerlord.ParseModule(strings.NewReader(ok)); err != nil {
		t.Errorf("a plain name was refused: %v", err)
	}
}

// Comments split one text run into many fragments; joining them must not recopy the whole run each time.
func TestCommentSplitTextParsesInLinearTime(t *testing.T) {
	parse := func(body string) time.Duration {
		start := time.Now()
		if _, err := bannerlord.ParseModule(strings.NewReader(`<Module><Id value="m"/><Junk>` + body + `</Junk></Module>`)); err != nil {
			t.Fatal(err)
		}
		return time.Since(start)
	}
	const fragments = bannerlord.MaxXMLSize/8 - 64
	plain := parse(strings.Repeat("a", fragments*8))
	split := parse(strings.Repeat("a<!---->", fragments))
	if split > 20*plain+50*time.Millisecond {
		t.Fatalf("comment-split text took %v against %v for the same bytes unsplit", split, plain)
	}
}

func TestSetModsKeepsADuplicateEntrysFields(t *testing.T) {
	t.Parallel()
	doc := `<UserData><SingleplayerData><ModDatas>` +
		`<UserModData><Id>A</Id><IsSelected>true</IsSelected><Foo>1</Foo></UserModData>` +
		`<UserModData><Id>A</Id><IsSelected>true</IsSelected><Foo>2</Foo></UserModData>` +
		`</ModDatas></SingleplayerData></UserData>`
	l, err := bannerlord.ParseLauncher(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.SetMods(bannerlord.Singleplayer, []string{"A"}); err != nil {
		t.Fatal(err)
	}
	out, err := l.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"<Foo>1</Foo>", "<Foo>2</Foo>"} {
		if !strings.Contains(string(out), s) {
			t.Errorf("lost %s in\n%s", s, out)
		}
	}
}
