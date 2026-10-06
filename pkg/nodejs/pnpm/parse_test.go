package pnpm

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			name:     "normal",
			file:     "testdata/pnpm-lock_normal.yaml",
			want:     pnpmNormal,
			wantDeps: pnpmNormalDeps,
		},
		{
			name:     "with dev deps",
			file:     "testdata/pnpm-lock_with_dev.yaml",
			want:     pnpmWithDev,
			wantDeps: pnpmWithDevDeps,
		},
		{
			name:     "many",
			file:     "testdata/pnpm-lock_many.yaml",
			want:     pnpmMany,
			wantDeps: pnpmManyDeps,
		},
		{
			name:     "archives",
			file:     "testdata/pnpm-lock_archives.yaml",
			want:     pnpmArchives,
			wantDeps: pnpmArchivesDeps,
		},
		{
			name:     "v6",
			file:     "testdata/pnpm-lock_v6.yaml",
			want:     pnpmV6,
			wantDeps: pnpmV6Deps,
		},
		{
			name:     "v6 with dev deps",
			file:     "testdata/pnpm-lock_v6_with_dev.yaml",
			want:     pnpmV6WithDev,
			wantDeps: pnpmV6WithDevDeps,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			require.NoError(t, err)

			got, deps, err := NewParser().Parse(f)
			require.NoError(t, err)

			sortLibs(got)
			sortLibs(tt.want)

			assert.Equal(t, tt.want, got)
			if tt.wantDeps != nil {
				sortDeps(deps)
				sortDeps(tt.wantDeps)
				assert.Equal(t, tt.wantDeps, deps)
			}
		})
	}
}

func sortDeps(deps []types.Dependency) {
	sort.Slice(deps, func(i, j int) bool {
		return strings.Compare(deps[i].ID, deps[j].ID) < 0
	})

	for i := range deps {
		sort.Strings(deps[i].DependsOn)
	}
}

func sortLibs(libs []types.Library) {
	sort.Slice(libs, func(i, j int) bool {
		ret := strings.Compare(libs[i].Name, libs[j].Name)
		if ret == 0 {
			return libs[i].Version < libs[j].Version
		}
		return ret < 0
	})
}

func Test_parsePackage(t *testing.T) {
	tests := []struct {
		name        string
		lockFileVer float64
		pkg         string
		wantName    string
		wantVersion string
	}{
		{
			name:        "v5 - relative path",
			lockFileVer: 5.0,
			pkg:         "/lodash/4.17.10",
			wantName:    "lodash",
			wantVersion: "4.17.10",
		},
		{
			name:        "v5 - registry",
			lockFileVer: 5.0,
			pkg:         "registry.npmjs.org/lodash/4.17.10",
			wantName:    "lodash",
			wantVersion: "4.17.10",
		},
		{
			name:        "v5 - relative path with slash",
			lockFileVer: 5.0,
			pkg:         "/@babel/generator/7.21.9",
			wantName:    "@babel/generator",
			wantVersion: "7.21.9",
		},
		{
			name:        "v5 - registry path with slash",
			lockFileVer: 5.0,
			pkg:         "registry.npmjs.org/@babel/generator/7.21.9",
			wantName:    "@babel/generator",
			wantVersion: "7.21.9",
		},
		{
			name:        "v5 - relative path with slash and peer deps",
			lockFileVer: 5.0,
			pkg:         "/@babel/helper-compilation-targets/7.21.5_@babel+core@7.21.8",
			wantName:    "@babel/helper-compilation-targets",
			wantVersion: "7.21.5",
		},
		{
			name:        "v5 - relative path with underline and peer deps",
			lockFileVer: 5.0,
			pkg:         "/lodash._baseclone/4.5.7_@babel+core@7.21.8",
			wantName:    "lodash._baseclone",
			wantVersion: "4.5.7",
		},
		{
			name:        "v5 - registry with slash and peer deps",
			lockFileVer: 5.0,
			pkg:         "registry.npmjs.org/@babel/helper-compilation-targets/7.21.5_@babel+core@7.21.8",
			wantName:    "@babel/helper-compilation-targets",
			wantVersion: "7.21.5",
		},
		{
			name:        "v5 - relative path with wrong version",
			lockFileVer: 5.0,
			pkg:         "/lodash/4-wrong",
			wantName:    "",
			wantVersion: "",
		},
		{
			name:        "v6 - relative path",
			lockFileVer: 6.0,
			pkg:         "/update-browserslist-db@1.0.11",
			wantName:    "update-browserslist-db",
			wantVersion: "1.0.11",
		},
		{
			name:        "v6 - registry",
			lockFileVer: 6.0,
			pkg:         "registry.npmjs.org/lodash@4.17.10",
			wantName:    "lodash",
			wantVersion: "4.17.10",
		},
		{
			name:        "v6 - relative path with slash",
			lockFileVer: 6.0,
			pkg:         "/@babel/helper-annotate-as-pure@7.18.6",
			wantName:    "@babel/helper-annotate-as-pure",
			wantVersion: "7.18.6",
		},
		{
			name:        "v6 - registry with slash",
			lockFileVer: 6.0,
			pkg:         "registry.npmjs.org/@babel/helper-annotate-as-pure@7.18.6",
			wantName:    "@babel/helper-annotate-as-pure",
			wantVersion: "7.18.6",
		},
		{
			name:        "v6 - relative path with slash and peer deps",
			lockFileVer: 6.0,
			pkg:         "/@babel/helper-compilation-targets@7.21.5(@babel/core@7.20.7)",
			wantName:    "@babel/helper-compilation-targets",
			wantVersion: "7.21.5",
		},
		{
			name:        "v6 - registry with slash and peer deps",
			lockFileVer: 6.0,
			pkg:         "registry.npmjs.org/@babel/helper-compilation-targets@7.21.5(@babel/core@7.20.7)",
			wantName:    "@babel/helper-compilation-targets",
			wantVersion: "7.21.5",
		},
		{
			name:        "v6 - relative path with wrong version",
			lockFileVer: 6.0,
			pkg:         "/lodash@4-wrong",
			wantName:    "",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotVersion := parsePackage(tt.pkg, tt.lockFileVer)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantVersion, gotVersion)
		})

	}
}

func TestParseDecodeError(t *testing.T) {
	_, _, err := NewParser().Parse(strings.NewReader("\tnot: [valid: yaml"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "decode error")
}

func TestParseUnknownLockfileVersion(t *testing.T) {
	// an unrecognized lockfile version yields no libraries rather than an error
	tests := []struct {
		name string
		body string
	}{
		{name: "unparsable version string", body: "lockfileVersion: 'not-a-number'\npackages: {}\n"},
		{name: "unexpected version type", body: "lockfileVersion:\n  nested: true\npackages: {}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, deps, err := NewParser().Parse(strings.NewReader(tt.body))
			require.NoError(t, err)
			assert.Empty(t, got)
			assert.Empty(t, deps)
		})
	}
}

func TestParseLockfileVersion(t *testing.T) {
	tests := []struct {
		name string
		in   LockFile
		want float64
	}{
		{name: "numeric version", in: LockFile{LockfileVersion: float64(5)}, want: 5},
		{name: "string version", in: LockFile{LockfileVersion: "6.0"}, want: 6},
		{name: "unparsable string version", in: LockFile{LockfileVersion: "six"}, want: -1},
		{name: "missing version", in: LockFile{}, want: -1},
		{name: "unexpected type", in: LockFile{LockfileVersion: []string{"6"}}, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLockfileVersion(tt.in))
		})
	}
}

func TestIsIndirectLib(t *testing.T) {
	direct := map[string]interface{}{"debug": "2.6.9"}

	assert.False(t, isIndirectLib("debug", direct))
	assert.True(t, isIndirectLib("ms", direct))
	assert.True(t, isIndirectLib("debug", nil))
}
