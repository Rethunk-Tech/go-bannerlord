package bannerlord

import (
	"fmt"
	"slices"
	"strings"
)

// IssueKind classifies a problem Sort found.
type IssueKind int

// The problems Sort reports.
const (
	// MissingDependency: Module needs Other, which is not in the set.
	MissingDependency IssueKind = iota + 1
	// OutdatedDependency: Other is present but older than the version Module requires.
	OutdatedDependency
	// IncompatibleModule: Module declares Other incompatible and both are present.
	IncompatibleModule
	// DuplicateModule: Module appears more than once; later copies are ignored.
	DuplicateModule
	// CycleDetected: Modules could not be ordered because they depend on each other, directly or
	// through a module behind the cycle. They are absent from the sorted order.
	CycleDetected
)

// Issue is one problem found while sorting. Sort never panics on bad input.
type Issue struct {
	Kind    IssueKind
	Module  string
	Other   string
	Modules []string // CycleDetected only
	Detail  string
}

func (i Issue) String() string {
	switch i.Kind {
	case MissingDependency:
		return fmt.Sprintf("%s requires missing module %s", i.Module, i.Other)
	case OutdatedDependency:
		return fmt.Sprintf("%s requires %s %s", i.Module, i.Other, i.Detail)
	case IncompatibleModule:
		return fmt.Sprintf("%s is incompatible with %s", i.Module, i.Other)
	case DuplicateModule:
		return fmt.Sprintf("module %s listed more than once", i.Module)
	case CycleDetected:
		return "dependency cycle among " + strings.Join(i.Modules, ", ")
	}
	return "unknown issue"
}

// Sort orders modules so that every dependency and LoadBeforeThis module precedes its dependent
// and every LoadAfterThis module follows it. Where the constraints leave a choice, official modules
// come first, then ids in ascending order (the default order of LauncherManager's
// LauncherManagerHandler.Modules.Orderer.cs, without its numeric-aware id comparison), so the
// result is deterministic. It returns the ids in load
// order and the issues found. Modules in or behind a cycle are left out of the order; a missing
// dependency does not remove its dependent.
func Sort(modules []Module) ([]string, []Issue) {
	var issues []Issue
	byID := make(map[string]*Module, len(modules))
	var ids []string
	for i := range modules {
		m := &modules[i]
		if _, dup := byID[m.ID]; dup {
			issues = append(issues, Issue{Kind: DuplicateModule, Module: m.ID})
			continue
		}
		byID[m.ID] = m
		ids = append(ids, m.ID)
	}

	after := make(map[string][]string, len(ids)) // before -> modules that must follow it
	indeg := make(map[string]int, len(ids))
	seen := make(map[[2]string]bool)
	edge := func(before, follower string) {
		k := [2]string{before, follower}
		if before == follower || seen[k] {
			return
		}
		seen[k] = true
		after[before] = append(after[before], follower)
		indeg[follower]++
	}

	for _, id := range ids {
		m := byID[id]
		for _, d := range m.Depends {
			dm, ok := byID[d.ID]
			switch {
			case !ok && !d.Optional:
				issues = append(issues, Issue{Kind: MissingDependency, Module: id, Other: d.ID})
			case ok:
				edge(d.ID, id)
				if !d.Version.IsZero() && dm.Version.Compare(d.Version) < 0 {
					issues = append(issues, Issue{Kind: OutdatedDependency, Module: id, Other: d.ID, Detail: d.Version.String() + " or newer, found " + dm.Version.String()})
				}
			}
		}
		for _, md := range m.Metadata {
			_, ok := byID[md.ID]
			if md.Incompatible && ok {
				issues = append(issues, Issue{Kind: IncompatibleModule, Module: id, Other: md.ID})
			}
			switch md.Order {
			case LoadBeforeThis:
				if ok {
					edge(md.ID, id)
				} else if !md.Optional {
					issues = append(issues, Issue{Kind: MissingDependency, Module: id, Other: md.ID})
				}
			case LoadAfterThis:
				if ok {
					edge(id, md.ID)
				}
			}
		}
	}

	less := func(a, b string) int {
		ao, bo := byID[a].Official, byID[b].Official
		if ao != bo {
			if ao {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	}

	var ready, order []string
	for _, id := range ids {
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	for len(ready) > 0 {
		slices.SortFunc(ready, less)
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, f := range after[id] {
			if indeg[f]--; indeg[f] == 0 {
				ready = append(ready, f)
			}
		}
	}
	if len(order) < len(ids) {
		var stuck []string
		for _, id := range ids {
			if !slices.Contains(order, id) {
				stuck = append(stuck, id)
			}
		}
		slices.SortFunc(stuck, less)
		issues = append(issues, Issue{Kind: CycleDetected, Modules: stuck})
	}
	return order, issues
}
