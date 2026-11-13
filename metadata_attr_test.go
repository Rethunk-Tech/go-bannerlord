package bannerlord

import (
	"strings"
	"testing"
)

func TestPascalCaseMetadataOrder(t *testing.T) {
	const xml = `<Module><Id value="M"/><DependedModuleMetadatas><DependedModuleMetadata id="Native" Order="LoadBeforeThis"/></DependedModuleMetadatas></Module>`
	m, err := ParseModule(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Metadata) != 1 {
		t.Fatalf("Metadata len=%d, want 1", len(m.Metadata))
	}
	if m.Metadata[0].Order != "LoadBeforeThis" {
		t.Fatalf("Order=%q, want LoadBeforeThis", m.Metadata[0].Order)
	}
}

func TestMetadataVersionOutdated(t *testing.T) {
	need, err := ParseVersion("v2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	have, err := ParseVersion("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	_, issues := Sort([]Module{
		{ID: "A", Metadata: []Metadata{{ID: "Lib", Order: LoadBeforeThis, Version: need}}},
		{ID: "Lib", Version: have},
	})
	for _, iss := range issues {
		if iss.Kind == OutdatedDependency && iss.Module == "A" && iss.Other == "Lib" {
			return
		}
	}
	t.Fatalf("issues=%v, want OutdatedDependency A/Lib", issues)
}

// .NET's bool.Parse, which the game and BUTR use, ignores case and surrounding white space.
func TestBooleansParseLikeDotNet(t *testing.T) {
	const xml = `<Module><Id value="M"/><Official value="True"/><DefaultModule value=" TRUE "/>` +
		`<DependedModules><DependedModule Id="Lib" Optional="True"/></DependedModules>` +
		`<DependedModuleMetadatas><DependedModuleMetadata id="X" optional="TRUE" incompatible="True"/></DependedModuleMetadatas></Module>`
	m, err := ParseModule(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Official || !m.Default || !m.Depends[0].Optional || !m.Metadata[0].Optional || !m.Metadata[0].Incompatible {
		t.Fatalf("module = %+v", m)
	}
}
