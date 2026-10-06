package poetry

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantLibs []types.Library
		wantDeps []types.Dependency
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "normal",
			file:     "testdata/poetry_normal.lock",
			wantLibs: poetryNormal,
			wantErr:  assert.NoError,
		},
		{
			name:     "many",
			file:     "testdata/poetry_many.lock",
			wantLibs: poetryMany,
			wantDeps: poetryManyDeps,
			wantErr:  assert.NoError,
		},
		{
			name:     "flask",
			file:     "testdata/poetry_flask.lock",
			wantLibs: poetryFlask,
			wantDeps: poetryFlaskDeps,
			wantErr:  assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			require.NoError(t, err)
			defer f.Close()

			p := &Parser{}
			gotLibs, gotDeps, err := p.Parse(f)
			if !tt.wantErr(t, err, fmt.Sprintf("Parse(%v)", tt.file)) {
				return
			}
			assert.Equalf(t, tt.wantLibs, gotLibs, "Parse(%v)", tt.file)
			assert.Equalf(t, tt.wantDeps, gotDeps, "Parse(%v)", tt.file)
		})
	}
}

func TestParseDependency(t *testing.T) {
	tests := []struct {
		name         string
		packageName  string
		versionRange interface{}
		libsVersions map[string][]string
		want         string
		wantErr      string
	}{
		{
			name:         "handle package name",
			packageName:  "Test_project.Name",
			versionRange: "*",
			libsVersions: map[string][]string{
				"test-project-name": {"1.0.0"},
			},
			want: "test-project-name@1.0.0",
		},
		{
			name:         "version range as string",
			packageName:  "test",
			versionRange: ">=1.0.0",
			libsVersions: map[string][]string{
				"test": {"2.0.0"},
			},
			want: "test@2.0.0",
		},
		{
			name:         "version range == *",
			packageName:  "test",
			versionRange: "*",
			libsVersions: map[string][]string{
				"test": {"3.0.0"},
			},
			want: "test@3.0.0",
		},
		{
			name:        "version range as json",
			packageName: "test",
			versionRange: map[string]interface{}{
				"version": ">=4.8.3",
				"markers": "python_version < \"3.8\"",
			},
			libsVersions: map[string][]string{
				"test": {"5.0.0"},
			},
			want: "test@5.0.0",
		},
		{
			name:         "libsVersions doesn't contain required version",
			packageName:  "test",
			versionRange: ">=1.0.0",
			libsVersions: map[string][]string{},
			wantErr:      "no version found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDependency(tt.packageName, tt.versionRange, tt.libsVersions)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewParser(t *testing.T) {
	require.NotNil(t, NewParser())
}

func TestParseDecodeError(t *testing.T) {
	_, _, err := NewParser().Parse(strings.NewReader("this is not = valid toml [[["))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode poetry.lock")
}

func TestParseDependencyErrors(t *testing.T) {
	t.Run("unparsable version", func(t *testing.T) {
		_, err := parseDependency("test", ">=1.0.0", map[string][]string{"test": {"not-a-version"}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to match version for test")
		assert.ErrorContains(t, err, "python version error")
	})

	t.Run("unparsable constraint", func(t *testing.T) {
		_, err := parseDependency("test", "not-a-constraint", map[string][]string{"test": {"1.0.0"}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "python constraint error")
	})

	t.Run("no version satisfies the constraint", func(t *testing.T) {
		_, err := parseDependency("test", ">=9.0.0", map[string][]string{"test": {"1.0.0", "2.0.0"}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "no matched version found")
	})

	t.Run("version range of an unsupported shape", func(t *testing.T) {
		// a version range that is neither a string nor a table leaves the
		// constraint empty, which fails to parse
		_, err := parseDependency("test", 42, map[string][]string{"test": {"1.0.0"}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "python constraint error")
	})
}

func TestMatchVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		want       bool
		wantErr    bool
	}{
		{name: "satisfied", version: "1.2.3", constraint: ">=1.0.0", want: true},
		{name: "not satisfied", version: "0.9.0", constraint: ">=1.0.0", want: false},
		{name: "exact match", version: "1.2.3", constraint: "==1.2.3", want: true},
		{name: "invalid version", version: "not-a-version", constraint: ">=1.0.0", wantErr: true},
		{name: "invalid constraint", version: "1.2.3", constraint: ">>>", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchVersion(tt.version, tt.constraint)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizePkgName(t *testing.T) {
	assert.Equal(t, "flask", normalizePkgName("Flask"))
	assert.Equal(t, "zope-interface", normalizePkgName("zope.interface"))
	assert.Equal(t, "my-package", normalizePkgName("my_package"))
}
