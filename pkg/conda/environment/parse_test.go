package environment_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/conda/environment"
	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    environment.Packages
		wantErr string
	}{
		{
			name:  "happy path",
			input: "testdata/happy.yaml",
			want: environment.Packages{
				Packages: []types.Library{
					{
						Name: "_openmp_mutex",
						Locations: types.Locations{
							{
								StartLine: 6,
								EndLine:   6,
							},
						},
					},
					{
						Name:    "asgiref",
						Version: "3.8.1",
						Locations: types.Locations{
							{
								StartLine: 21,
								EndLine:   21,
							},
						},
					},
					{
						Name:    "blas",
						Version: "1.0",
						Locations: types.Locations{
							{
								StartLine: 5,
								EndLine:   5,
							},
						},
					},
					{
						Name:    "bzip2",
						Version: "1.0.8",
						Locations: types.Locations{
							{
								StartLine: 19,
								EndLine:   19,
							},
						},
					},
					{
						Name:    "ca-certificates",
						Version: "2024.2",
						Locations: types.Locations{
							{
								StartLine: 7,
								EndLine:   7,
							},
						},
					},
					{
						Name:    "django",
						Version: "5.0.6",
						Locations: types.Locations{
							{
								StartLine: 22,
								EndLine:   22,
							},
						},
					},
					{
						Name: "ld_impl_linux-aarch64",
						Locations: types.Locations{
							{
								StartLine: 8,
								EndLine:   8,
							},
						},
					},
					{
						Name: "libblas",
						Locations: types.Locations{
							{
								StartLine: 9,
								EndLine:   9,
							},
						},
					},
					{
						Name: "libcblas",
						Locations: types.Locations{
							{
								StartLine: 10,
								EndLine:   10,
							},
						},
					},
					{
						Name:    "libexpat",
						Version: "2.6.2",
						Locations: types.Locations{
							{
								StartLine: 11,
								EndLine:   11,
							},
						},
					},
					{
						Name:    "libffi",
						Version: "3.4.2",
						Locations: types.Locations{
							{
								StartLine: 12,
								EndLine:   12,
							},
						},
					},
					{
						Name: "libgcc-ng",
						Locations: types.Locations{
							{
								StartLine: 13,
								EndLine:   13,
							},
						},
					},
					{
						Name: "libgfortran-ng",
						Locations: types.Locations{
							{
								StartLine: 14,
								EndLine:   14,
							},
						},
					},
					{
						Name: "libgfortran5",
						Locations: types.Locations{
							{
								StartLine: 15,
								EndLine:   15,
							},
						},
					},
					{
						Name:    "libgomp",
						Version: "13.2.0",
						Locations: types.Locations{
							{
								StartLine: 16,
								EndLine:   16,
							},
						},
					},
					{
						Name: "liblapack",
						Locations: types.Locations{
							{
								StartLine: 17,
								EndLine:   17,
							},
						},
					},
					{
						Name:    "libnsl",
						Version: "2.0.1",
						Locations: types.Locations{
							{
								StartLine: 18,
								EndLine:   18,
							},
						},
					},
				},
				Prefix: "/opt/conda/envs/test-env",
			},
		},
		{
			name:    "invalid yaml file",
			input:   "testdata/invalid.yaml",
			wantErr: "cannot unmarshal !!str `invalid` into environment.environment",
		},
		{
			name:    "`dependency` field uses unsupported type",
			input:   "testdata/wrong-deps-type.yaml",
			wantErr: `unsupported dependency type "!!int" on line 5`,
		},
		{
			name:    "nested field uses unsupported type",
			input:   "testdata/wrong-nested-type.yaml",
			wantErr: `unsupported dependency type "!!str" on line 5`,
		},
		{
			name:    "nested dependency uses unsupported type",
			input:   "testdata/wrong-nested-dep-type.yaml",
			wantErr: `unsupported dependency type "!!map" on line 6`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.input)
			require.NoError(t, err)
			defer f.Close()

			got, err := environment.NewParser().Parse(f)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
