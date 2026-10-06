package jar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestManifestDetermineGroupID(t *testing.T) {
	tests := []struct {
		name     string
		manifest manifest
		want     string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "implementation vendor ID",
			manifest: manifest{implementationVendorId: "org.example", specificationVendor: "spec"},
			want:     "org.example",
			wantErr:  assert.NoError,
		},
		{
			name:     "bundle symbolic name with suffix",
			manifest: manifest{bundleSymbolicName: "com.fasterxml.jackson.core.jackson-databind"},
			want:     "com.fasterxml.jackson.core",
			wantErr:  assert.NoError,
		},
		{
			name:     "bundle symbolic name without suffix",
			manifest: manifest{bundleSymbolicName: "nodots"},
			want:     "nodots",
			wantErr:  assert.NoError,
		},
		{
			name:     "bundle symbolic name starting with a dot",
			manifest: manifest{bundleSymbolicName: ".leadingdot"},
			want:     ".leadingdot",
			wantErr:  assert.NoError,
		},
		{
			name:     "implementation vendor",
			manifest: manifest{implementationVendor: "Example Inc."},
			want:     "Example Inc.",
			wantErr:  assert.NoError,
		},
		{
			name:     "specification vendor",
			manifest: manifest{specificationVendor: "spec.vendor"},
			want:     "spec.vendor",
			wantErr:  assert.NoError,
		},
		{
			name:     "no group ID",
			manifest: manifest{},
			wantErr:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.manifest.determineGroupID()
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManifestDetermineArtifactID(t *testing.T) {
	tests := []struct {
		name     string
		manifest manifest
		want     string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "implementation title",
			manifest: manifest{implementationTitle: "impl", specificationTitle: "spec"},
			want:     "impl",
			wantErr:  assert.NoError,
		},
		{
			name:     "specification title",
			manifest: manifest{specificationTitle: "spec", bundleName: "bundle"},
			want:     "spec",
			wantErr:  assert.NoError,
		},
		{
			name:     "bundle name",
			manifest: manifest{bundleName: " bundle "},
			want:     "bundle",
			wantErr:  assert.NoError,
		},
		{
			name:     "no artifact ID",
			manifest: manifest{},
			wantErr:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.manifest.determineArtifactID()
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManifestDetermineVersion(t *testing.T) {
	tests := []struct {
		name     string
		manifest manifest
		want     string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "implementation version",
			manifest: manifest{implementationVersion: "1.0.0", specificationVersion: "2.0.0"},
			want:     "1.0.0",
			wantErr:  assert.NoError,
		},
		{
			name:     "specification version",
			manifest: manifest{specificationVersion: "2.0.0", bundleVersion: "3.0.0"},
			want:     "2.0.0",
			wantErr:  assert.NoError,
		},
		{
			name:     "bundle version",
			manifest: manifest{bundleVersion: " 3.0.0 "},
			want:     "3.0.0",
			wantErr:  assert.NoError,
		},
		{
			name:     "no version",
			manifest: manifest{},
			wantErr:  assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.manifest.determineVersion()
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManifestProperties(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m := manifest{
			implementationVendorId: "org.example",
			implementationTitle:    "example",
			implementationVersion:  "1.0.0",
		}

		got := m.properties("testdata/example.jar")
		assert.Equal(t, Properties{
			GroupID:    "org.example",
			ArtifactID: "example",
			Version:    "1.0.0",
			FilePath:   "testdata/example.jar",
		}, got)
		assert.True(t, got.Valid())
		assert.Equal(t, "org.example:example:1.0.0", got.String())
	})

	t.Run("no group ID", func(t *testing.T) {
		m := manifest{
			implementationTitle:   "example",
			implementationVersion: "1.0.0",
		}
		assert.Equal(t, Properties{}, m.properties("testdata/example.jar"))
	})

	t.Run("no artifact ID", func(t *testing.T) {
		m := manifest{
			implementationVendorId: "org.example",
			implementationVersion:  "1.0.0",
		}
		assert.Equal(t, Properties{}, m.properties("testdata/example.jar"))
	})

	t.Run("no version", func(t *testing.T) {
		m := manifest{
			implementationVendorId: "org.example",
			implementationTitle:    "example",
		}
		assert.Equal(t, Properties{}, m.properties("testdata/example.jar"))
	})
}

func TestPropertiesValid(t *testing.T) {
	require.True(t, Properties{GroupID: "g", ArtifactID: "a", Version: "v"}.Valid())
	require.False(t, Properties{ArtifactID: "a", Version: "v"}.Valid())
	require.False(t, Properties{GroupID: "g", Version: "v"}.Valid())
	require.False(t, Properties{GroupID: "g", ArtifactID: "a"}.Valid())
	require.False(t, Properties{}.Valid())
}

func TestIsArtifact(t *testing.T) {
	require.True(t, isArtifact("test.jar"))
	require.True(t, isArtifact("test.ear"))
	require.True(t, isArtifact("test.war"))
	require.False(t, isArtifact("test.zip"))
	require.False(t, isArtifact("test"))
}

func TestParseFileName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     Properties
	}{
		{
			name:     "jar with version",
			filePath: "testdata/heuristic-1.0.0-SNAPSHOT.jar",
			want: Properties{
				ArtifactID: "heuristic",
				Version:    "1.0.0-SNAPSHOT",
				FilePath:   "testdata/heuristic-1.0.0-SNAPSHOT.jar",
			},
		},
		{
			name:     "jar with numeric version",
			filePath: "testdata/commons-lang3-3.12.0.jar",
			want: Properties{
				ArtifactID: "commons-lang3",
				Version:    "3.12.0",
				FilePath:   "testdata/commons-lang3-3.12.0.jar",
			},
		},
		{
			name:     "no version in name",
			filePath: "testdata/commons-lang3.jar",
			want:     Properties{},
		},
		{
			name:     "unparseable name",
			filePath: "testdata/nonsense",
			want:     Properties{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFileName(tt.filePath))
		})
	}
}

func TestRemoveLibraryDuplicates(t *testing.T) {
	libs := removeLibraryDuplicates(libraryFixtures())
	assert.Len(t, libs, 2)
}

func libraryFixtures() []types.Library {
	return []types.Library{
		{ID: "a@1", Name: "a", Version: "1", FilePath: "one.jar"},
		{ID: "a@1", Name: "a", Version: "1", FilePath: "one.jar"},
		{ID: "a@1", Name: "a", Version: "1", FilePath: "two.jar"},
	}
}
