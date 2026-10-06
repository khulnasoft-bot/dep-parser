package mod

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		replace bool
		want    []types.Library
	}{
		{
			name:    "normal",
			file:    "testdata/normal/go.mod",
			replace: true,
			want:    GoModNormal,
		},
		{
			name:    "without go version",
			file:    "testdata/no-go-version/go.mod",
			replace: true,
			want:    GoModNoGoVersion,
		},
		{
			name:    "replace",
			file:    "testdata/replaced/go.mod",
			replace: true,
			want:    GoModReplaced,
		},
		{
			name:    "no replace",
			file:    "testdata/replaced/go.mod",
			replace: false,
			want:    GoModUnreplaced,
		},
		{
			name:    "replace with version",
			file:    "testdata/replaced-with-version/go.mod",
			replace: true,
			want:    GoModReplacedWithVersion,
		},
		{
			name:    "replaced with version mismatch",
			file:    "testdata/replaced-with-version-mismatch/go.mod",
			replace: true,
			want:    GoModReplacedWithVersionMismatch,
		},
		{
			name:    "replaced with local path",
			file:    "testdata/replaced-with-local-path/go.mod",
			replace: true,
			want:    GoModReplacedWithLocalPath,
		},
		{
			name:    "replaced with local path and version",
			file:    "testdata/replaced-with-local-path-and-version/go.mod",
			replace: true,
			want:    GoModReplacedWithLocalPathAndVersion,
		},
		{
			name:    "replaced with local path and version, mismatch",
			file:    "testdata/replaced-with-local-path-and-version-mismatch/go.mod",
			replace: true,
			want:    GoModReplacedWithLocalPathAndVersionMismatch,
		},
		{
			name:    "go 1.16",
			file:    "testdata/go116/go.mod",
			replace: true,
			want:    GoMod116,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			require.NoError(t, err)

			got, _, err := NewParser(tt.replace).Parse(f)
			require.NoError(t, err)

			sort.Slice(got, func(i, j int) bool {
				return got[i].Name < got[j].Name
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Name < tt.want[j].Name
			})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModuleID(t *testing.T) {
	type args struct {
		name    string
		version string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				name:    "github.com/aquasecurity/trivy",
				version: "0.38.0",
			},
			want: "github.com/aquasecurity/trivy@v0.38.0",
		},
		{
			name: "pseudo version",
			args: args{
				name:    "github.com/khulnasoft/dep-parser",
				version: "0.0.0-20230130190635-5e31092b0621",
			},
			want: "github.com/khulnasoft/dep-parser@v0.0.0-20230130190635-5e31092b0621",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ModuleID(tt.args.name, tt.args.version), "ModuleID(%v, %v)", tt.args.name, tt.args.version)
		})
	}
}

func TestLessThan117(t *testing.T) {
	tests := []struct {
		name string
		ver  string
		want bool
	}{
		{name: "1.16", ver: "1.16", want: true},
		{name: "1.15", ver: "1.15", want: true},
		{name: "1.17", ver: "1.17", want: false},
		{name: "1.21", ver: "1.21", want: false},
		{name: "2.0", ver: "2.0", want: false},
		{name: "patch version", ver: "1.16.15", want: false},
		{name: "no minor", ver: "1", want: false},
		{name: "non-numeric major", ver: "x.16", want: false},
		{name: "non-numeric minor", ver: "1.x", want: false},
		{name: "empty", ver: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, lessThan117(tt.ver))
		})
	}
}

func TestResolveVCSUrl(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		want       string
	}{
		{
			name:       "github.com",
			modulePath: "github.com/khulnasoft/dep-parser",
			want:       "https://github.com/khulnasoft/dep-parser",
		},
		{
			name:       "github.com with major version suffix",
			modulePath: "github.com/khulnasoft/dep-parser/v2",
			want:       "https://github.com/khulnasoft/dep-parser",
		},
		{
			name:       "gopkg.in with user",
			modulePath: "gopkg.in/user/pkg.v3",
			want:       "https://github.com/user/pkg",
		},
		{
			name:       "gopkg.in without user",
			modulePath: "gopkg.in/pkg.v3",
			want:       "https://github.com/go-pkg/pkg",
		},
		{
			name:       "unrecognized host",
			modulePath: "golang.org/x/mod",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveVCSUrl(tt.modulePath))
		})
	}
}

func TestGetExternalRefs(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []types.ExternalRef
	}{
		{
			name: "known VCS host",
			path: "github.com/khulnasoft/dep-parser",
			want: []types.ExternalRef{
				{Type: types.RefVCS, URL: "https://github.com/khulnasoft/dep-parser"},
			},
		},
		{
			name: "unknown host",
			path: "golang.org/x/mod",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := NewParser(false).(*Parser)
			require.True(t, ok)
			assert.Equal(t, tt.want, p.GetExternalRefs(tt.path))
		})
	}
}
