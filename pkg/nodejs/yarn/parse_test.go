package yarn

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParsePattern(t *testing.T) {
	vectors := []struct {
		name           string
		target         string
		expectName     string
		expectProtocol string
		expactVersion  string
		occurErr       bool
	}{
		{
			name:          "normal",
			target:        `asn1@~0.2.3:`,
			expectName:    "asn1",
			expactVersion: "~0.2.3",
		},
		{
			name:           "normal with protocol",
			target:         `asn1@npm:~0.2.3:`,
			expectName:     "asn1",
			expectProtocol: "npm",
			expactVersion:  "~0.2.3",
		},
		{
			name:          "scope",
			target:        `@babel/code-frame@^7.0.0:`,
			expectName:    "@babel/code-frame",
			expactVersion: "^7.0.0",
		},
		{
			name:           "scope with protocol",
			target:         `@babel/code-frame@npm:^7.0.0:`,
			expectName:     "@babel/code-frame",
			expectProtocol: "npm",
			expactVersion:  "^7.0.0",
		},
		{
			name:           "scope with protocol and quotes",
			target:         `"@babel/code-frame@npm:^7.0.0":`,
			expectName:     "@babel/code-frame",
			expectProtocol: "npm",
			expactVersion:  "^7.0.0",
		},
		{
			name:          "unusual version",
			target:        `grunt-contrib-cssmin@3.0.*:`,
			expectName:    "grunt-contrib-cssmin",
			expactVersion: "3.0.*",
		},
		{
			name:          "conditional version",
			target:        `"js-tokens@^3.0.0 || ^4.0.0":`,
			expectName:    "js-tokens",
			expactVersion: "^3.0.0 || ^4.0.0",
		},
		{
			target:        "grunt-contrib-uglify-es@gruntjs/grunt-contrib-uglify#harmony:",
			expectName:    "grunt-contrib-uglify-es",
			expactVersion: "gruntjs/grunt-contrib-uglify#harmony",
		},
		{
			target:         `"jquery@git+https://xxxx:x-oauth-basic@github.com/tomoyamachi/jquery":`,
			expectName:     "jquery",
			expectProtocol: "git+https",
			expactVersion:  "//xxxx:x-oauth-basic@github.com/tomoyamachi/jquery",
		},
		{
			target:   `normal line`,
			occurErr: true,
		},
	}

	for _, v := range vectors {
		gotName, gotProtocol, gotVersion, err := parsePattern(v.target)

		if v.occurErr != (err != nil) {
			t.Errorf("expect error %t but err is %s", v.occurErr, err)
			continue
		}

		if gotName != v.expectName {
			t.Errorf("name mismatch: got %s, want %s, target :%s", gotName, v.expectName, v.target)
		}

		if gotProtocol != v.expectProtocol {
			t.Errorf("protocol mismatch: got %s, want %s, target :%s", gotProtocol, v.expectProtocol, v.target)
		}

		if gotVersion != v.expactVersion {
			t.Errorf("version mismatch: got %s, want %s, target :%s", gotVersion, v.expactVersion, v.target)
		}
	}
}

func TestParsePackagePatterns(t *testing.T) {
	vectors := []struct {
		name           string
		target         string
		expectName     string
		expectProtocol string
		expactPatterns []string
		occurErr       bool
	}{
		{
			name:       "normal",
			target:     `asn1@~0.2.3:`,
			expectName: "asn1",
			expactPatterns: []string{
				"asn1@~0.2.3",
			},
		},
		{
			name:       "normal with quotes",
			target:     `"asn1@~0.2.3":`,
			expectName: "asn1",
			expactPatterns: []string{
				"asn1@~0.2.3",
			},
		},
		{
			name:           "normal with protocol",
			target:         `asn1@npm:~0.2.3:`,
			expectName:     "asn1",
			expectProtocol: "npm",
			expactPatterns: []string{
				"asn1@~0.2.3",
			},
		},
		{
			name:       "multiple patterns",
			target:     `loose-envify@^1.1.0, loose-envify@^1.4.0:`,
			expectName: "loose-envify",
			expactPatterns: []string{
				"loose-envify@^1.1.0",
				"loose-envify@^1.4.0",
			},
		},
		{
			name:           "multiple patterns v2",
			target:         `"loose-envify@npm:^1.1.0, loose-envify@npm:^1.4.0":`,
			expectName:     "loose-envify",
			expectProtocol: "npm",
			expactPatterns: []string{
				"loose-envify@^1.1.0",
				"loose-envify@^1.4.0",
			},
		},
		{
			target:   `normal line`,
			occurErr: true,
		},
	}

	for _, v := range vectors {
		gotName, gotProtocol, gotPatterns, err := parsePackagePatterns(v.target)

		if v.occurErr != (err != nil) {
			t.Errorf("expect error %t but err is %s", v.occurErr, err)
			continue
		}

		if gotName != v.expectName {
			t.Errorf("name mismatch: got %s, want %s, target: %s", gotName, v.expectName, v.target)
		}

		if gotProtocol != v.expectProtocol {
			t.Errorf("protocol mismatch: got %s, want %s, target: %s", gotProtocol, v.expectProtocol, v.target)
		}

		sort.Strings(gotPatterns)
		sort.Strings(v.expactPatterns)

		assert.Equal(t, v.expactPatterns, gotPatterns)
	}
}

func TestGetDependency(t *testing.T) {
	vectors := []struct {
		name          string
		target        string
		expectName    string
		expactVersion string
		occurErr      bool
	}{
		{
			name:          "normal",
			target:        `    chalk "^2.0.1"`,
			expectName:    "chalk",
			expactVersion: "^2.0.1",
		},
		{
			name:          "range",
			target:        `    js-tokens "^3.0.0 || ^4.0.0"`,
			expectName:    "js-tokens",
			expactVersion: "^3.0.0 || ^4.0.0",
		},
		{
			name:          "normal v2",
			target:        `    depd: ~1.1.2`,
			expectName:    "depd",
			expactVersion: "~1.1.2",
		},
		{
			name:          "range version v2",
			target:        `    statuses: ">= 1.5.0 < 2"`,
			expectName:    "statuses",
			expactVersion: ">= 1.5.0 < 2",
		},
		{
			name:          "name with scope",
			target:        `    "@types/color-name": ^1.1.1`,
			expectName:    "@types/color-name",
			expactVersion: "^1.1.1",
		},
		{
			name:          "version with protocol",
			target:        `    ms: "npm:2.1.2"`,
			expectName:    "ms",
			expactVersion: "2.1.2",
		},
	}

	for _, v := range vectors {
		gotName, gotVersion, err := getDependency(v.target)

		if v.occurErr != (err != nil) {
			t.Errorf("expect error %t but err is %s", v.occurErr, err)
			continue
		}

		if gotName != v.expectName {
			t.Errorf("name mismatch: got %s, want %s, target: %s", gotName, v.expectName, v.target)
		}

		if gotVersion != v.expactVersion {
			t.Errorf("version mismatch: got %s, want %s, target: %s", gotVersion, v.expactVersion, v.target)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		file     string // Test input file
		want     []types.Library
		wantDeps []types.Dependency
	}{
		{
			name:     "normal",
			file:     "testdata/yarn_normal.lock",
			want:     yarnNormal,
			wantDeps: yarnNormalDeps,
		},
		{
			name:     "react",
			file:     "testdata/yarn_react.lock",
			want:     yarnReact,
			wantDeps: yarnReactDeps,
		},
		{
			name:     "yarn with dev",
			file:     "testdata/yarn_with_dev.lock",
			want:     yarnWithDev,
			wantDeps: yarnWithDevDeps,
		},
		{
			name:     "yarn many",
			file:     "testdata/yarn_many.lock",
			want:     yarnMany,
			wantDeps: yarnManyDeps,
		},
		{
			name:     "yarn real world",
			file:     "testdata/yarn_realworld.lock",
			want:     yarnRealWorld,
			wantDeps: yarnRealWorldDeps,
		},
		{
			file: "testdata/yarn_with_npm.lock",
			want: yarnWithNpm,
		},
		{
			name:     "yarn v2 normal",
			file:     "testdata/yarn_v2_normal.lock",
			want:     yarnV2Normal,
			wantDeps: yarnV2NormalDeps,
		},
		{
			name:     "yarn v2 react",
			file:     "testdata/yarn_v2_react.lock",
			want:     yarnV2React,
			wantDeps: yarnV2ReactDeps,
		},
		{
			name:     "yarn v2 with dev",
			file:     "testdata/yarn_v2_with_dev.lock",
			want:     yarnV2WithDev,
			wantDeps: yarnV2WithDevDeps,
		},
		{
			name:     "yarn v2 many",
			file:     "testdata/yarn_v2_many.lock",
			want:     yarnV2Many,
			wantDeps: yarnV2ManyDeps,
		},
		{
			name:     "yarn with local dependency",
			file:     "testdata/yarn_with_local.lock",
			want:     yarnNormal,
			wantDeps: yarnNormalDeps,
		},
		{
			name:     "yarn v2 with protocols in dependency section",
			file:     "testdata/yarn_v2_deps_with_protocol.lock",
			want:     yarnV2DepsWithProtocol,
			wantDeps: yarnV2DepsWithProtocolDeps,
		},
		{
			name: "yarn with git dependency",
			file: "testdata/yarn_with_git.lock",
		},
		{
			name: "yarn file with bad protocol",
			file: "testdata/yarn_with_bad_protocol.lock",
			want: yarnBadProtocol,
		},
		{
			name: "yarn v3 with metadata block",
			file: "testdata/yarn_v3_with_metadata.lock",
			want: yarnV3WithMetadata,
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
	for _, lib := range libs {
		sortLocations(lib.Locations)
	}
}

func sortLocations(locs []types.Location) {
	sort.Slice(locs, func(i, j int) bool {
		return locs[i].StartLine < locs[j].StartLine
	})
}

// yarnV3WithMetadata covers a yarn v3 lockfile whose leading metadata block
// carries no library and must be skipped.
var yarnV3WithMetadata = []types.Library{
	{ID: "lodash@4.17.21", Name: "lodash", Version: "4.17.21", Locations: []types.Location{{StartLine: 11, EndLine: 13}}},
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "quoted version", target: `version "2.0.6"`, want: "2.0.6"},
		{name: "unquoted version", target: `version 2.0.6`, want: "2.0.6"},
		{name: "empty version", target: `version`, wantErr: true},
		{name: "no version keyword", target: `resolved "https://example.com"`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVersion(tt.target)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to parse version")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidAndIgnoreProtocol(t *testing.T) {
	for _, proto := range []string{"npm", ""} {
		assert.True(t, validProtocol(proto), proto)
	}
	assert.False(t, validProtocol("git"))
	assert.False(t, validProtocol("https"))

	for _, proto := range []string{"workspace", "patch", "file", "link", "portal", "github", "git", "git+ssh", "git+http", "git+https", "git+file"} {
		assert.True(t, ignoreProtocol(proto), proto)
	}
	assert.False(t, ignoreProtocol("npm"))
	assert.False(t, ignoreProtocol(""))
}

func TestParseDependency(t *testing.T) {
	got, err := parseDependency(`    chalk "^2.0.1"`)
	require.NoError(t, err)
	assert.Equal(t, "chalk@^2.0.1", got)

	// not a dependency line
	_, err = parseDependency(`  version "2.0.6"`)
	require.Error(t, err)
}

func TestParseResults(t *testing.T) {
	patternIDs := map[string]string{
		"debug@^2.6.9": "debug@2.6.9",
		"ms@2.0.0":     "ms@2.0.0",
	}

	got := parseResults(patternIDs, map[string][]string{
		"debug@2.6.9": {"ms@2.0.0"},
	})

	assert.Equal(t, []types.Dependency{
		{ID: "debug@2.6.9", DependsOn: []string{"ms@2.0.0"}},
	}, got)
}

func TestScanBlocks(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		atEOF      bool
		wantAdvace int
		wantToken  string
	}{
		{name: "newline terminated blocks", input: "a\nb\n\nc\n\n", wantAdvace: 5, wantToken: "a\nb"},
		{name: "CRLF terminated blocks", input: "a\r\nb\r\n\r\nc\r\n", wantAdvace: 8, wantToken: "a\r\nb"},
		{name: "final unterminated block at EOF", input: "last", atEOF: true, wantAdvace: 4, wantToken: "last"},
		{name: "empty at EOF", input: "", atEOF: true, wantAdvace: 0},
		{name: "request more data", input: "partial", wantAdvace: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance, token, err := scanBlocks([]byte(tt.input), tt.atEOF)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAdvace, advance)
			assert.Equal(t, tt.wantToken, string(token))
		})
	}
}

func TestLineScanner(t *testing.T) {
	s := NewLineScanner(strings.NewReader("one\ntwo\nthree\n"))

	// nothing consumed yet
	assert.Equal(t, 0, s.LineNum(1))

	require.True(t, s.Scan())
	assert.Equal(t, "one", s.Text())
	assert.Equal(t, 1, s.LineNum(1))

	require.True(t, s.Scan())
	require.True(t, s.Scan())
	assert.Equal(t, "three", s.Text())
	assert.Equal(t, 3, s.LineNum(1))

	assert.False(t, s.Scan())
	assert.Equal(t, 3, s.LineNum(1))
}

func TestParseScannerError(t *testing.T) {
	// A single block larger than bufio.Scanner's max token size makes the
	// scanner fail, which surfaces as a scan error from Parse.
	big := "\"pkg@^1.0.0\":\n  version \"1.0.0\"\n" + strings.Repeat("x", 128*1024)

	_, _, err := NewParser().Parse(strings.NewReader(big))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to scan yarn.lock")
}

func TestGetDependencyWithUnsupportedProtocol(t *testing.T) {
	// a non-npm protocol yields empty name and version without an error, so
	// the caller can tell "ignored" apart from "malformed"
	name, version, err := getDependency(`    lib "git+https://example.com/lib.git"`)
	require.NoError(t, err)
	assert.Empty(t, name)
	assert.Empty(t, version)
}

func TestParseSkipsBlocksWithoutName(t *testing.T) {
	// A block that yields no library name (comments and blank content) is
	// skipped without failing the whole file.
	lock := "# just a comment\n\n\"lodash@^4.17.21\":\n  version \"4.17.21\"\n"

	got, _, err := NewParser().Parse(strings.NewReader(lock))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "lodash@4.17.21", got[0].ID)
}

func TestParseCRLF(t *testing.T) {
	// CRLF-terminated blocks must be split correctly. The file is built in
	// memory rather than checked in as testdata, because git normalizes line
	// endings in text files.
	lock := strings.ReplaceAll(
		"asap@~2.0.6:\n  version\n\njquery@^3.4.1:\n  version \"3.4.1\"\n",
		"\n", "\r\n")

	got, _, err := NewParser().Parse(strings.NewReader(lock))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "jquery@3.4.1", got[0].ID)
	assert.Equal(t, "3.4.1", got[0].Version)
	assert.Equal(t, types.Location{StartLine: 4, EndLine: 5}, got[0].Locations[0])
}
