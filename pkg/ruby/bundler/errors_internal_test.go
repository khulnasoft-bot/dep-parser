package bundler

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dio "github.com/khulnasoft/dep-parser/pkg/io"
)

// newScanner wraps a string so it satisfies the parser's reader interface.
func newScanner(s string) dio.ReadSeekerAt {
	return dio.NopCloser(strings.NewReader(s))
}

func TestCountLeadingSpace(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{name: "no indent", line: "GEM", want: 0},
		{name: "two spaces", line: "  rails", want: 2},
		{name: "four spaces", line: "    rails (6.0.0)", want: 4},
		{name: "six spaces", line: "      nokogiri", want: 6},
		{name: "leading tab", line: "\trails", want: 0},
		{name: "empty", line: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countLeadingSpace(tt.line))
		})
	}
}

func TestParseDirectDeps(t *testing.T) {
	// parseDirectDeps is called with the scanner already positioned past the
	// DEPENDENCIES header
	rest := `  rails (~> 6.0)
  rake

BUNDLED WITH
   2.2.3
`
	scanner := bufio.NewScanner(strings.NewReader(rest))
	got := parseDirectDeps(scanner)

	assert.Equal(t, []string{"rails", "rake"}, got)
}

func TestParseDirectDepsStopsAtNextSection(t *testing.T) {
	rest := "PLATFORMS\n  ruby\n"
	scanner := bufio.NewScanner(strings.NewReader(rest))

	assert.Nil(t, parseDirectDeps(scanner))
}

func TestParseMalformedDependencyLine(t *testing.T) {
	// a spec line without a version is skipped rather than recorded
	lock := `GEM
  remote: https://rubygems.org/
  specs:
    rails

    rake (13.0.6)
`

	got, _, err := NewParser().Parse(newScanner(lock))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "rake", got[0].Name)
}

func TestParseUnknownDependencyInGraph(t *testing.T) {
	// a graph entry pointing at a gem that is not in the specs is dropped
	lock := `GEM
  remote: https://rubygems.org/
  specs:
    rails (6.0.3)
      rake
      absent-gem

    rake (13.0.6)
`

	got, deps, err := NewParser().Parse(newScanner(lock))
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, deps, 1)
	assert.Equal(t, "rails@6.0.3", deps[0].ID)
	assert.Equal(t, []string{"rake@13.0.6"}, deps[0].DependsOn)
}

func TestParseScannerError(t *testing.T) {
	// bufio.Scanner's default token limit is 64KiB
	long := strings.Repeat("x", 128*1024)

	_, _, err := NewParser().Parse(newScanner(long))
	require.Error(t, err)
	assert.ErrorContains(t, err, "scan error")
}
