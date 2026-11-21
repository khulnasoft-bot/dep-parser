// Package graph provides utilities for working with the dependency graph
// emitted by parsers as []types.Dependency.
package graph

import (
	"slices"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

// Graph is an adjacency list of package ID -> IDs it depends on.
type Graph map[string][]string

// New builds a Graph from parser dependency output. Dependency entries whose
// IDs never appear as packages are still represented so callers can detect
// dangling edges.
func New(deps []types.Dependency) Graph {
	g := make(Graph, len(deps))
	for _, d := range deps {
		g[d.ID] = slices.Clone(d.DependsOn)
	}
	return g
}

// Nodes returns all node IDs in the graph (both dependents and dependencies).
func (g Graph) Nodes() []string {
	seen := make(map[string]struct{}, len(g))
	for id, deps := range g {
		seen[id] = struct{}{}
		for _, d := range deps {
			seen[d] = struct{}{}
		}
	}
	nodes := make([]string, 0, len(seen))
	for n := range seen {
		nodes = append(nodes, n)
	}
	slices.Sort(nodes)
	return nodes
}

// Dependencies returns the direct dependencies of id.
func (g Graph) Dependencies(id string) []string {
	return g[id]
}

// Transitive returns all package IDs reachable from id (excluding id itself),
// sorted. Cycles are handled safely.
func (g Graph) Transitive(id string) []string {
	seen := make(map[string]struct{})
	var visit func(cur string)
	visit = func(cur string) {
		for _, next := range g[cur] {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			visit(next)
		}
	}
	visit(id)
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// AffectedBy returns all package IDs that transitively depend on target
// (i.e. would be impacted if target were vulnerable), sorted. Cycles handled.
func (g Graph) AffectedBy(target string) []string {
	rev := make(Graph, len(g))
	for id, deps := range g {
		for _, d := range deps {
			rev[d] = append(rev[d], id)
		}
	}
	seen := make(map[string]struct{})
	var visit func(cur string)
	visit = func(cur string) {
		for _, next := range rev[cur] {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			visit(next)
		}
	}
	visit(target)
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// Path returns one shortest dependency path from root to target as a slice of
// IDs [root, ..., target], or nil if unreachable. BFS; deterministic by
// sorting adjacency at each step.
func (g Graph) Path(root, target string) []string {
	if root == target {
		return []string{root}
	}
	type item struct {
		node string
		path []string
	}
	queue := []item{{node: root, path: []string{root}}}
	visited := map[string]struct{}{root: {}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		nexts := slices.Clone(g[cur.node])
		slices.Sort(nexts)
		for _, n := range nexts {
			if n == target {
				return append(slices.Clone(cur.path), n)
			}
			if _, ok := visited[n]; ok {
				continue
			}
			visited[n] = struct{}{}
			queue = append(queue, item{node: n, path: append(slices.Clone(cur.path), n)})
		}
	}
	return nil
}

// Roots returns node IDs that nothing depends on (top-level packages), sorted.
func (g Graph) Roots() []string {
	dependedOn := make(map[string]struct{}, len(g))
	for _, deps := range g {
		for _, d := range deps {
			dependedOn[d] = struct{}{}
		}
	}
	var roots []string
	for id := range g {
		if _, ok := dependedOn[id]; !ok {
			roots = append(roots, id)
		}
	}
	slices.Sort(roots)
	return roots
}
