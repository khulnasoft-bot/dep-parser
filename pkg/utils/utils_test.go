package utils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		want []string
	}{
		{
			name: "happy path",
			ss:   []string{"a", "b", "a", "c", "b"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "no duplicates",
			ss:   []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "empty input",
			ss:   nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, UniqueStrings(tt.ss))
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name   string
		parent map[string]string
		child  map[string]string
		want   map[string]string
	}{
		{
			name:   "nil parent returns child",
			parent: nil,
			child:  map[string]string{"a": "1"},
			want:   map[string]string{"a": "1"},
		},
		{
			name:   "child overwrites parent",
			parent: map[string]string{"a": "1", "b": "2"},
			child:  map[string]string{"b": "3", "c": "4"},
			want:   map[string]string{"a": "1", "b": "3", "c": "4"},
		},
		{
			name:   "nil child keeps parent",
			parent: map[string]string{"a": "1"},
			child:  nil,
			want:   map[string]string{"a": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeMaps(tt.parent, tt.child)
			require.Equal(t, tt.want, got)

			// the parent map must not be mutated
			if tt.parent != nil {
				require.NotSame(t, &tt.parent, &got)
			}
		})
	}
}

func TestPackageID(t *testing.T) {
	require.Equal(t, "asn1@0.2.6", PackageID("asn1", "0.2.6"))
	require.Equal(t, "@", PackageID("", ""))
}

func TestUniqueLibraries(t *testing.T) {
	tests := []struct {
		name     string
		libs     []types.Library
		wantLibs []types.Library
	}{
		{
			name: "happy path merge locations",
			libs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Locations: []types.Location{
						{
							StartLine: 10,
							EndLine:   14,
						},
					},
				},
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Locations: []types.Location{
						{
							StartLine: 24,
							EndLine:   30,
						},
					},
				},
			},
			wantLibs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Locations: []types.Location{
						{
							StartLine: 10,
							EndLine:   14,
						},
						{
							StartLine: 24,
							EndLine:   30,
						},
					},
				},
			},
		},
		{
			name: "happy path Dev and Root deps",
			libs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     true,
				},
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     false,
				},
			},
			wantLibs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     false,
				},
			},
		},
		{
			name: "happy path Root and Dev deps",
			libs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     false,
				},
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     true,
				},
			},
			wantLibs: []types.Library{
				{
					ID:      "asn1@0.2.6",
					Name:    "asn1",
					Version: "0.2.6",
					Dev:     false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLibs := UniqueLibraries(tt.libs)
			require.Equal(t, tt.wantLibs, gotLibs)
		})
	}
}
