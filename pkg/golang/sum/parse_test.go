package sum

import (
	"os"
	"path"
	"sort"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParse(t *testing.T) {
	vectors := []struct {
		file string
		want []types.Library
	}{
		{
			file: "testdata/gomod_normal.sum",
			want: GoModNormal,
		},
		{
			file: "testdata/gomod_emptyline.sum",
			want: GoModEmptyLine,
		},
		{
			file: "testdata/gomod_many.sum",
			want: GoModMany,
		},
		{
			file: "testdata/gomod_trivy.sum",
			want: GoModTrivy,
		},
	}

	for _, v := range vectors {
		t.Run(path.Base(v.file), func(t *testing.T) {
			f, err := os.Open(v.file)
			require.NoError(t, err)
			defer f.Close()

			got, _, err := NewParser().Parse(f)
			require.NoError(t, err)

			for i := range got {
				got[i].ID = "" // Not compare IDs, tested in mod.TestModuleID()
			}

			sort.Slice(got, func(i, j int) bool {
				return got[i].Name < got[j].Name
			})
			sort.Slice(v.want, func(i, j int) bool {
				return v.want[i].Name < v.want[j].Name
			})

			assert.Equal(t, v.want, got)
		})
	}
}

func TestParseIsDeterministic(t *testing.T) {
	// The parser collects modules in a map, so the output order must be
	// explicitly sorted. Without sorting this assertion fails intermittently.
	first, _, err := NewParser().Parse(mustOpen(t, "testdata/gomod_many.sum"))
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		got, _, err := NewParser().Parse(mustOpen(t, "testdata/gomod_many.sum"))
		require.NoError(t, err)
		require.Equal(t, first, got, "library order differs on run %d", i)
	}

	// and the order must actually be sorted
	assert.IsIncreasing(t, lo.Map(first, func(l types.Library, _ int) string {
		return l.ID
	}))
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	return f
}
