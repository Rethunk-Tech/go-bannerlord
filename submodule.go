package bannerlord

import (
	"fmt"
	"io"
	"strings"
)

// Order says where a module sits relative to the module that lists it. The empty Order is
// BUTR's LoadType.None: no ordering, only the Optional and Incompatible flags apply.
type Order string

// The values of a DependedModuleMetadata order attribute.
const (
	LoadBeforeThis Order = "LoadBeforeThis"
	LoadAfterThis  Order = "LoadAfterThis"
)

// Dependency is a DependedModule entry: the named module must load before this one.
type Dependency struct {
	ID       string
	Version  Version // minimum version; zero when unspecified
	Optional bool
}

// Metadata is an ordering or incompatibility entry: a DependedModuleMetadata, a
// ModulesToLoadAfterThis module (LoadAfterThis, optional) or an IncompatibleModules module.
type Metadata struct {
	ID           string
	Order        Order
	Version      Version // zero when unspecified
	Optional     bool
	Incompatible bool
}

// Module is the parsed content of one SubModule.xml.
type Module struct {
	ID       string
	Name     string
	Version  Version
	Official bool // Official is true, or ModuleType is Official or OfficialOptional
	Default  bool
	Depends  []Dependency
	Metadata []Metadata
	DLLs     []string // SubModules[].DLLName, bare file names
}

// ParseModule parses a SubModule.xml the way Bannerlord.ModuleManager's ModuleInfoExtended.FromXml
// does: scalars are the value attribute of a child element, and an unparsable version is
// unspecified. A missing Id, an Id with characters outside letters, digits and ".-_", or a DLL
// name containing a path separator is an error.
func ParseModule(r io.Reader) (Module, error) {
	root, err := parseTree(r)
	if err != nil {
		return Module{}, err
	}
	if root.name != "Module" {
		return Module{}, fmt.Errorf("%w: root element %q, want Module", ErrInvalid, root.name)
	}
	var m Module
	m.ID, _ = root.valueOf("Id")
	if !validID(m.ID) {
		return Module{}, fmt.Errorf("%w: module id %q", ErrInvalid, m.ID)
	}
	m.Name, _ = root.valueOf("Name")
	if s, ok := root.valueOf("Version"); ok {
		m.Version, _ = ParseVersion(s)
	}
	ty, _ := root.valueOf("ModuleType")
	m.Official = flag(root, "Official") || ty == "Official" || ty == "OfficialOptional"
	m.Default = flag(root, "DefaultModule")

	if deps := root.child("DependedModules"); deps != nil {
		for _, d := range deps.childrenNamed("DependedModule") {
			dep, err := parseDependency(d)
			if err != nil {
				return Module{}, err
			}
			m.Depends = append(m.Depends, dep)
		}
	}
	if metas := root.child("DependedModuleMetadatas"); metas != nil {
		for _, d := range metas.childrenNamed("DependedModuleMetadata") {
			meta, err := parseMetadata(d)
			if err != nil {
				return Module{}, err
			}
			m.Metadata = append(m.Metadata, meta)
		}
	}
	if l := root.child("ModulesToLoadAfterThis"); l != nil {
		for _, d := range l.childrenNamed("Module") {
			id, ok := d.attr("Id")
			if !ok {
				continue
			}
			if !validID(id) {
				return Module{}, fmt.Errorf("%w: module id %q", ErrInvalid, id)
			}
			m.Metadata = append(m.Metadata, Metadata{ID: id, Order: LoadAfterThis, Optional: true})
		}
	}
	if l := root.child("IncompatibleModules"); l != nil {
		for _, d := range l.childrenNamed("Module") {
			id, ok := d.attr("Id")
			if !ok {
				continue
			}
			if !validID(id) {
				return Module{}, fmt.Errorf("%w: module id %q", ErrInvalid, id)
			}
			m.Metadata = append(m.Metadata, Metadata{ID: id, Optional: true, Incompatible: true})
		}
	}
	if subs := root.child("SubModules"); subs != nil {
		for _, s := range subs.childrenNamed("SubModule") {
			dll, ok := s.valueOf("DLLName")
			if !ok || dll == "" {
				continue
			}
			if strings.ContainsAny(dll, "/\\\x00") || dll == ".." {
				return Module{}, fmt.Errorf("%w: DLLName %q", ErrInvalid, dll)
			}
			m.DLLs = append(m.DLLs, dll)
		}
	}
	return m, nil
}

func flag(n *node, name string) bool {
	v, _ := n.valueOf(name)
	return v == "true"
}

func attrFlag(n *node, name string) bool {
	v, _ := n.attr(name)
	return v == "true"
}

func attrVersion(n *node, name string) Version {
	s, _ := n.attr(name)
	v, _ := ParseVersion(s)
	return v
}

func parseDependency(n *node) (Dependency, error) {
	id, _ := n.attr("Id")
	if !validID(id) {
		return Dependency{}, fmt.Errorf("%w: dependency id %q", ErrInvalid, id)
	}
	return Dependency{ID: id, Version: attrVersion(n, "DependentVersion"), Optional: attrFlag(n, "Optional")}, nil
}

func parseMetadata(n *node) (Metadata, error) {
	id, _ := n.attr("id")
	if !validID(id) {
		return Metadata{}, fmt.Errorf("%w: metadata id %q", ErrInvalid, id)
	}
	ord, _ := n.attr("order")
	o := Order(ord)
	if o != LoadBeforeThis && o != LoadAfterThis {
		o = ""
	}
	return Metadata{ID: id, Order: o, Version: attrVersion(n, "version"), Optional: attrFlag(n, "optional"), Incompatible: attrFlag(n, "incompatible")}, nil
}
