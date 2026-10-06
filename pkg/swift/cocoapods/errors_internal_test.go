package cocoapods

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

// failReader fails on Read so the io.ReadAll error path is exercised.
type failReader struct{}

func (failReader) Read([]byte) (int, error)          { return 0, xerrors.New("read boom") }
func (failReader) Seek(int64, int) (int64, error)    { return 0, nil }
func (failReader) ReadAt([]byte, int64) (int, error) { return 0, nil }

func TestParseDecodeError(t *testing.T) {
	_, _, err := NewParser().Parse(bytes.NewReader([]byte("\tnot: [valid: yaml")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode cocoapods lock file")
}

func TestParseInvalidChildDeps(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			// the map key is not a recognizable dependency, so it is logged
			// and skipped before the child list is inspected
			name: "unparsable dependency key",
			body: "PODS:\n  - AppCenter = 4.2.0:\n      - AppCenter/Core (4.2.0)\n",
		},
		{
			name: "child dependencies are not a list",
			body: "PODS:\n  - AppCenter/Core (4.2.0): 4.2.0\n",
			want: "invalid value of cocoapods direct dependency",
		},
		{
			name: "child dependency is not a string",
			body: "PODS:\n  - AppCenter/Core (4.2.0):\n      - 42\n",
			want: "must be string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := NewParser().Parse(bytes.NewReader([]byte(tt.body)))
			if tt.want == "" {
				require.NoError(t, err)
				assert.Empty(t, got)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestParseDep(t *testing.T) {
	tests := []struct {
		name    string
		dep     string
		want    types.Library
		wantErr bool
	}{
		{
			name: "plain pod",
			dep:  "AppCenter (4.2.0)",
			want: types.Library{ID: "AppCenter@4.2.0", Name: "AppCenter", Version: "4.2.0"},
		},
		{
			name: "subspec with exact version",
			dep:  "AppCenter/Analytics (= 4.2.0)",
			want: types.Library{ID: "AppCenter/Analytics@= 4.2.0", Name: "AppCenter/Analytics", Version: "= 4.2.0"},
		},
		{
			name:    "missing version",
			dep:     "AppCenter = 4.2.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDep(tt.dep)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "Unable to determine cocoapods dependency")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
