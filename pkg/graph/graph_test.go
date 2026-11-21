package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func testGraph() Graph {
	// a -> b -> d, a -> c -> d, e -> d (diamond + cycle f <-> g)
	return New([]types.Dependency{
		{ID: "a", DependsOn: []string{"b", "c"}},
		{ID: "b", DependsOn: []string{"d"}},
		{ID: "c", DependsOn: []string{"d"}},
		{ID: "d", DependsOn: []string{"f"}},
		{ID: "e", DependsOn: []string{"d"}},
		{ID: "f", DependsOn: []string{"g"}},
		{ID: "g", DependsOn: []string{"f"}},
	})
}

func TestNew(t *testing.T) {
	g := New([]types.Dependency{{ID: "a", DependsOn: []string{"b"}}})
	assert.Equal(t, []string{"b"}, g.Dependencies("a"))
	assert.Empty(t, g.Dependencies("missing"))
}

func TestNodes(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g"}, testGraph().Nodes())
}

func TestTransitive(t *testing.T) {
	g := testGraph()
	assert.Equal(t, []string{"b", "c", "d", "f", "g"}, g.Transitive("a"))
	assert.Equal(t, []string{"f", "g"}, g.Transitive("d")) // cycle terminates
	assert.Empty(t, g.Transitive("missing"))
}

func TestAffectedBy(t *testing.T) {
	g := testGraph()
	// everything that reaches d: a (via b/c), b, c, e
	assert.Equal(t, []string{"a", "b", "c", "e"}, g.AffectedBy("d"))
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g"}, g.AffectedBy("f"))
	assert.Empty(t, g.AffectedBy("missing"))
}

func TestPath(t *testing.T) {
	g := testGraph()
	require.Equal(t, []string{"a", "b", "d"}, g.Path("a", "d")) // shortest, sorted adjacency
	assert.Equal(t, []string{"a"}, g.Path("a", "a"))
	assert.Nil(t, g.Path("d", "a"))
	assert.Nil(t, g.Path("a", "missing"))
}

func TestRoots(t *testing.T) {
	assert.Equal(t, []string{"a", "e"}, testGraph().Roots())
}
