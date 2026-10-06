package npm

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		file     string // Test input file
		want     []types.Library
		wantDeps []types.Dependency
	}{
		{
			name:     "lock version v1",
			file:     "testdata/package-lock_v1.json",
			want:     npmV1Libs,
			wantDeps: npmDeps,
		},
		{
			name:     "lock version v2",
			file:     "testdata/package-lock_v2.json",
			want:     npmV2Libs,
			wantDeps: npmDeps,
		},
		{
			name:     "lock version v3",
			file:     "testdata/package-lock_v3.json",
			want:     npmV2Libs,
			wantDeps: npmDeps,
		},
		{
			name:     "lock version v3 with workspace",
			file:     "testdata/package-lock_v3_with_workspace.json",
			want:     npmV3WithWorkspaceLibs,
			wantDeps: npmV3WithWorkspaceDeps,
		},
		{
			name:     "lock version v3 with workspace and without direct deps field",
			file:     "testdata/package-lock_v3_without_root_deps_field.json",
			want:     npmV3WithoutRootDepsField,
			wantDeps: npmV3WithoutRootDepsFieldDeps,
		},
		{
			name:     "lock version v1 with a missing transitive dep",
			file:     "testdata/package-lock_v1_with_missing_deps.json",
			want:     npmV1WithMissingDeps,
			wantDeps: nil,
		},
		{
			name:     "lock version v3 with missing direct and transitive deps",
			file:     "testdata/package-lock_v3_with_missing_deps.json",
			want:     npmV3WithMissingDeps,
			wantDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			require.NoError(t, err)

			got, deps, err := NewParser().Parse(f)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
			if tt.wantDeps != nil {
				assert.Equal(t, tt.wantDeps, deps)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Run("malformed json", func(t *testing.T) {
		_, _, err := NewParser().Parse(bytes.NewReader([]byte("this is not json")))
		require.Error(t, err)
		assert.ErrorContains(t, err, "decode error")
	})

	t.Run("unreadable input", func(t *testing.T) {
		_, _, err := NewParser().Parse(errReader{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "read error")
	})
}

// errReader always fails on Read, exercising the io.ReadAll error path.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, xerrors.New("read boom")
}

func (errReader) Seek(int64, int) (int64, error) { return 0, nil }

func (errReader) ReadAt([]byte, int64) (int, error) { return 0, nil }

func TestIsWorkspace(t *testing.T) {
	assert.True(t, isWorkspace("functions/func1", []string{"functions/*"}))
	assert.True(t, isWorkspace("functions/func1", []string{"other", "functions/func1"}))
	assert.False(t, isWorkspace("functions/func1", []string{"other/*"}))

	// a malformed pattern is logged and skipped rather than aborting
	assert.False(t, isWorkspace("functions/func1", []string{"[", "other"}))
}

func TestPkgNameFromPath(t *testing.T) {
	assert.Equal(t, "debug", pkgNameFromPath("node_modules/debug"))
	assert.Equal(t, "debug", pkgNameFromPath("node_modules/a/node_modules/debug"))
	// paths without a node_modules prefix are returned as-is
	assert.Equal(t, "functions/func1", pkgNameFromPath("functions/func1"))
}

func TestJoinPaths(t *testing.T) {
	assert.Equal(t, "node_modules/debug", joinPaths("node_modules", "debug"))
	assert.Equal(t, "node_modules/a/node_modules/debug", joinPaths("node_modules", "a", "node_modules", "debug"))
	assert.Equal(t, "", joinPaths())
}

func TestFindDependsOn(t *testing.T) {
	packages := map[string]Package{
		"node_modules/debug":                               {Version: "2.6.9"},
		"node_modules/body-parser/node_modules/debug":      {Version: "1.0.0"},
		"node_modules/with-missing-dep/node_modules/debug": {Version: "3.0.0"},
	}

	tests := []struct {
		name    string
		pkgPath string
		depName string
		want    string
		wantErr bool
	}{
		{name: "hoisted dependency", pkgPath: "node_modules/a", depName: "debug", want: "debug@2.6.9"},
		{name: "nested nearest wins", pkgPath: "node_modules/body-parser/node_modules/a", depName: "debug", want: "debug@1.0.0"},
		{name: "unknown dependency", pkgPath: "node_modules/a", depName: "absent", wantErr: true},
		{name: "path without node_modules", pkgPath: "functions/func1", depName: "debug", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findDependsOn(tt.pkgPath, tt.depName, packages)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "can't find dependsOn")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
