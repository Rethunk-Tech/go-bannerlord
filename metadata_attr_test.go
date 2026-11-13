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
