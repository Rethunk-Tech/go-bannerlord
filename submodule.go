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

	if m.Depends, err = parseChildren(root, "DependedModules", "DependedModule", parseDependency); err != nil {
		return Module{}, err
	}
	if m.Metadata, err = parseChildren(root, "DependedModuleMetadatas", "DependedModuleMetadata", parseMetadata); err != nil {
		return Module{}, err
	}
	for _, list := range []struct {
		container string
		meta      Metadata
	}{
		{"ModulesToLoadAfterThis", Metadata{Order: LoadAfterThis, Optional: true}},
		{"IncompatibleModules", Metadata{Optional: true, Incompatible: true}},
	} {
		if m.Metadata, err = appendModuleRefs(m.Metadata, root, list.container, list.meta); err != nil {
			return Module{}, err
		}
	}
	if m.DLLs, err = parseDLLs(root); err != nil {
		return Module{}, err
	}
	return m, nil
}

// parseChildren parses each item element under the container child of root, or returns nil when there is no container.
func parseChildren[T any](root *node, container, item string, parse func(*node) (T, error)) ([]T, error) {
	c := root.child(container)
	if c == nil {
		return nil, nil
	}
	var out []T
	for _, d := range c.childrenNamed(item) {
		v, err := parse(d)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// appendModuleRefs appends tmpl, with each Module id of the container child, to metas. A Module without an Id is skipped.
func appendModuleRefs(metas []Metadata, root *node, container string, tmpl Metadata) ([]Metadata, error) {
	l := root.child(container)
	if l == nil {
		return metas, nil
	}
	for _, d := range l.childrenNamed("Module") {
		id, ok := d.attr("Id")
		if !ok {
			continue
		}
		if !validID(id) {
			return nil, fmt.Errorf("%w: module id %q", ErrInvalid, id)
		}
		tmpl.ID = id
		metas = append(metas, tmpl)
	}
	return metas, nil
}

func parseDLLs(root *node) ([]string, error) {
	subs := root.child("SubModules")
	if subs == nil {
		return nil, nil
	}
	var dlls []string
	for _, s := range subs.childrenNamed("SubModule") {
		dll, ok := s.valueOf("DLLName")
		if !ok || dll == "" {
			continue
		}
		if !validDLLName(dll) {
			return nil, fmt.Errorf("%w: DLLName %q", ErrInvalid, dll)
		}
		dlls = append(dlls, dll)
	}
	return dlls, nil
}

// validDLLName accepts a plain file name: one element that Windows cannot read as a drive-relative path, an
// alternate data stream or a device (CON, NUL, COM1 and the like, with or without an extension).
func validDLLName(name string) bool {
	if name == "." || name == ".." || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return false
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return false
	}
	stem, _, _ := strings.Cut(name, ".")
	switch strings.ToUpper(strings.TrimRight(stem, " ")) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}

// isTrue is .NET's bool.Parse: case-insensitive, white space around it ignored.
func isTrue(v string) bool { return strings.EqualFold(strings.TrimSpace(v), "true") }

func flag(n *node, name string) bool {
	v, _ := n.valueOf(name)
	return isTrue(v)
}

func attrFlag(n *node, name string) bool {
	v, _ := n.attr(name)
	return isTrue(v)
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
