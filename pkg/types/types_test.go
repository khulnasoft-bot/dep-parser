package types

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibrariesSort(t *testing.T) {
	tests := []struct {
		name string
		libs Libraries
		want Libraries
	}{
		{
			name: "sorted by ID",
			libs: Libraries{
				{ID: "b", Name: "b", Version: "1.0.0"},
				{ID: "a", Name: "a", Version: "1.0.0"},
			},
			want: Libraries{
				{ID: "a", Name: "a", Version: "1.0.0"},
				{ID: "b", Name: "b", Version: "1.0.0"},
			},
		},
		{
			name: "equal IDs fall back to name",
			libs: Libraries{
				{ID: "a", Name: "b", Version: "1.0.0"},
				{ID: "a", Name: "a", Version: "1.0.0"},
			},
			want: Libraries{
				{ID: "a", Name: "a", Version: "1.0.0"},
				{ID: "a", Name: "b", Version: "1.0.0"},
			},
		},
		{
			name: "equal IDs and names fall back to version",
			libs: Libraries{
				{ID: "a", Name: "a", Version: "2.0.0"},
				{ID: "a", Name: "a", Version: "1.0.0"},
			},
			want: Libraries{
				{ID: "a", Name: "a", Version: "1.0.0"},
				{ID: "a", Name: "a", Version: "2.0.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, len(tt.libs), tt.libs.Len())
			sort.Sort(tt.libs)
			assert.Equal(t, tt.want, tt.libs)
		})
	}
}

func TestLocationsSort(t *testing.T) {
	locs := Locations{
		{StartLine: 30, EndLine: 32},
		{StartLine: 10, EndLine: 12},
	}

	require.Equal(t, 2, locs.Len())
	assert.True(t, locs.Less(1, 0))
	assert.False(t, locs.Less(0, 1))

	sort.Sort(locs)
	assert.Equal(t, Locations{
		{StartLine: 10, EndLine: 12},
		{StartLine: 30, EndLine: 32},
	}, locs)
}

func TestDependenciesSort(t *testing.T) {
	deps := Dependencies{
		{ID: "b", DependsOn: []string{"c"}},
		{ID: "a"},
	}

	require.Equal(t, 2, deps.Len())
	assert.True(t, deps.Less(1, 0))
	assert.False(t, deps.Less(0, 1))

	sort.Sort(deps)
	assert.Equal(t, Dependencies{{ID: "a"}, {ID: "b", DependsOn: []string{"c"}}}, deps)
}

func TestRelationshipRank(t *testing.T) {
	assert.Equal(t, 0, relationshipRank(RelationshipRoot))
	assert.Equal(t, 1, relationshipRank(RelationshipWorkspace))
	assert.Equal(t, 2, relationshipRank(RelationshipDirect))
	assert.Equal(t, 3, relationshipRank(RelationshipIndirect))
	assert.Equal(t, 4, relationshipRank(RelationshipUnknown))
	assert.Equal(t, 4, relationshipRank(""))
}

func TestPackagesSort(t *testing.T) {
	pkgs := Packages{
		{Name: "b", Version: "1.0", Relationship: RelationshipDirect},
		{Name: "a", Version: "1.0", Relationship: RelationshipDirect},
		{Name: "a", Version: "1.0", Relationship: RelationshipRoot},
	}
	sort.Sort(pkgs)
	assert.Equal(t, RelationshipRoot, pkgs[0].Relationship)
	assert.Equal(t, "a", pkgs[1].Name)
	assert.Equal(t, "b", pkgs[2].Name)
}

func TestUserError(t *testing.T) {
	e := &UserError{Message: "boom"}
	assert.Equal(t, "boom", e.Error())
}
