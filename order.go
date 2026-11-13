package bannerlord

import (
	"container/heap"
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
	g := &loadGraph{
		byID:  make(map[string]*Module, len(modules)),
		after: make(map[string][]string, len(modules)),
		indeg: make(map[string]int, len(modules)),
		seen:  make(map[[2]string]bool),
	}
	var ids []string
	for i := range modules {
		m := &modules[i]
		if _, dup := g.byID[m.ID]; dup {
			issues = append(issues, Issue{Kind: DuplicateModule, Module: m.ID})
			continue
		}
		g.byID[m.ID] = m
		ids = append(ids, m.ID)
	}
	for _, id := range ids {
		issues = append(issues, g.constrain(id)...)
	}
	order := g.topoSort(ids)
	if len(order) < len(ids) {
		var stuck []string
		for _, id := range ids {
			if !slices.Contains(order, id) {
				stuck = append(stuck, id)
			}
		}
		slices.SortFunc(stuck, g.less)
		issues = append(issues, Issue{Kind: CycleDetected, Modules: stuck})
	}
	return order, issues
}

// loadGraph holds the ordering constraints between the modules of one Sort.
type loadGraph struct {
	byID  map[string]*Module
	after map[string][]string // before -> modules that must follow it
	indeg map[string]int
	seen  map[[2]string]bool
}

func (g *loadGraph) edge(before, follower string) {
	k := [2]string{before, follower}
	if before == follower || g.seen[k] {
		return
	}
	g.seen[k] = true
	g.after[before] = append(g.after[before], follower)
	g.indeg[follower]++
}

// constrain adds the edges module id declares and returns the issues its dependencies and metadata raise.
func (g *loadGraph) constrain(id string) []Issue {
	var issues []Issue
	m := g.byID[id]
	for _, d := range m.Depends {
		dm, ok := g.byID[d.ID]
		switch {
		case !ok && !d.Optional:
			issues = append(issues, Issue{Kind: MissingDependency, Module: id, Other: d.ID})
		case ok:
			g.edge(d.ID, id)
			if !d.Version.IsZero() && dm.Version.Compare(d.Version) < 0 {
				issues = append(issues, Issue{Kind: OutdatedDependency, Module: id, Other: d.ID, Detail: d.Version.String() + " or newer, found " + dm.Version.String()})
			}
		}
	}
	for _, md := range m.Metadata {
		dm, ok := g.byID[md.ID]
		if ok && !md.Version.IsZero() && dm.Version.Compare(md.Version) < 0 {
			issues = append(issues, Issue{Kind: OutdatedDependency, Module: id, Other: md.ID, Detail: md.Version.String() + " or newer, found " + dm.Version.String()})
		}
		if md.Incompatible && ok {
			issues = append(issues, Issue{Kind: IncompatibleModule, Module: id, Other: md.ID})
		}
		switch md.Order {
		case LoadBeforeThis:
			if ok {
				g.edge(md.ID, id)
			} else if !md.Optional {
				issues = append(issues, Issue{Kind: MissingDependency, Module: id, Other: md.ID})
			}
		case LoadAfterThis:
			if ok {
				g.edge(id, md.ID)
			}
		}
	}
	return issues
}

func (g *loadGraph) less(a, b string) int {
	ao, bo := g.byID[a].Official, g.byID[b].Official
	if ao != bo {
		if ao {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// topoSort is Kahn's algorithm, taking the least ready module each step; modules in or behind a cycle are left out.
func (g *loadGraph) topoSort(ids []string) []string {
	ready := &readyHeap{g: g}
	var order []string
	for _, id := range ids {
		if g.indeg[id] == 0 {
			heap.Push(ready, id)
		}
	}
	for ready.Len() > 0 {
		id, _ := heap.Pop(ready).(string)
		order = append(order, id)
		for _, f := range g.after[id] {
			if g.indeg[f]--; g.indeg[f] == 0 {
				heap.Push(ready, f)
			}
		}
	}
	return order
}

// readyHeap is a min-heap of module ids under loadGraph.less.
type readyHeap struct {
	g   *loadGraph
	ids []string
}

func (h *readyHeap) Len() int           { return len(h.ids) }
func (h *readyHeap) Less(i, j int) bool { return h.g.less(h.ids[i], h.ids[j]) < 0 }
func (h *readyHeap) Swap(i, j int)      { h.ids[i], h.ids[j] = h.ids[j], h.ids[i] }
func (h *readyHeap) Push(x any) {
	id, _ := x.(string)
	h.ids = append(h.ids, id)
}

func (h *readyHeap) Pop() any {
	n := len(h.ids) - 1
	x := h.ids[n]
	h.ids = h.ids[:n]
	return x
}
