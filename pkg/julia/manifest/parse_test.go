package julia

import (
	"os"
	"sort"
	"testing"

	"github.com/BurntSushi/toml"
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
			name:     "Manifest v1.6",
			file:     "testdata/primary/Manifest_v1.6.toml",
			want:     juliaV1_6Libs,
			wantDeps: juliaV1_6Deps,
		},
		{
			name:     "Manifest v1.8",
			file:     "testdata/primary/Manifest_v1.8.toml",
			want:     juliaV1_8Libs,
			wantDeps: juliaV1_8Deps,
		},
		{
			name:     "no deps v1.6",
			file:     "testdata/no_deps_v1.6/Manifest.toml",
			want:     nil,
			wantDeps: nil,
		},
		{
			name:     "no deps v1.9",
			file:     "testdata/no_deps_v1.9/Manifest.toml",
			want:     nil,
			wantDeps: nil,
		},
		{
			name:     "dep extensions v1.9",
			file:     "testdata/dep_ext_v1.9/Manifest.toml",
			want:     juliaV1_9DepExtLibs,
			wantDeps: nil,
		},
		{
			name:     "shadowed dep v1.9",
			file:     "testdata/shadowed_dep_v1.9/Manifest.toml",
			want:     juliaV1_9ShadowedDepLibs,
			wantDeps: juliaV1_9ShadowedDepDeps,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.file)
			require.NoError(t, err)

			gotLibs, gotDeps, err := NewParser().Parse(f)
			require.NoError(t, err)

			sort.Sort(types.Libraries(tt.want))
			assert.Equal(t, tt.want, gotLibs)
			if tt.wantDeps != nil {
				sort.Sort(types.Dependencies(tt.wantDeps))
				assert.Equal(t, tt.wantDeps, gotDeps)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Run("invalid manifest", func(t *testing.T) {
		f, err := os.Open("testdata/invalid_manifest/Manifest.toml")
		require.NoError(t, err)
		defer f.Close()

		_, _, err = NewParser().Parse(f)
		require.Error(t, err)
		assert.ErrorContains(t, err, "decode error")
	})

	t.Run("dependency resolving to multiple packages", func(t *testing.T) {
		f, err := os.Open("testdata/multiple_deps/Manifest.toml")
		require.NoError(t, err)
		defer f.Close()

		_, _, err = NewParser().Parse(f)
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to decode manifest dependencies")
		assert.ErrorContains(t, err, "parsed multiple deps")
	})

	t.Run("seek failure", func(t *testing.T) {
		_, _, err := NewParser().Parse(failSeekReader{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "seek error")
	})
}

// failSeekReader fails on Seek, which the parser needs in order to try both
// the old and new manifest formats.
type failSeekReader struct{}

func (failSeekReader) Read([]byte) (int, error) {
	return 0, xerrors.New("read boom")
}

func (failSeekReader) Seek(int64, int) (int64, error) {
	return 0, xerrors.New("seek boom")
}

func (failSeekReader) ReadAt([]byte, int64) (int, error) {
	return 0, xerrors.New("readat boom")
}

func TestDepVersion(t *testing.T) {
	// stdlib packages carry no version of their own and inherit Julia's
	assert.Equal(t, "1.9.0", depVersion(&primitiveDependency{}, "1.9.0"))
	assert.Equal(t, "1.3.1", depVersion(&primitiveDependency{Version: "1.3.1"}, "1.9.0"))
}

func TestDecodeDependencyUndecodableShape(t *testing.T) {
	var doc struct {
		Deps toml.Primitive `toml:"deps"`
	}
	// a map whose values are not strings fits neither supported shape
	metadata, err := toml.Decode("deps = { a = 1 }\n", &doc)
	require.NoError(t, err)

	man := &primitiveManifest{}
	_, err = decodeDependency(man, primitiveDependency{Dependencies: doc.Deps}, &metadata)
	require.Error(t, err)
}
