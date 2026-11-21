package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/parsers"
	"github.com/khulnasoft/dep-parser/pkg/types"
	"github.com/khulnasoft/dep-parser/pkg/version/app"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

func TestParserRegistry(t *testing.T) {
	assert.NotEmpty(t, parsers.Registry)
	for name, entry := range parsers.Registry {
		assert.NotNil(t, entry.Parse, "parser %q is nil", name)
		assert.Equal(t, name, entry.Name)
	}
	assert.Len(t, parsers.Names(), len(parsers.Registry))
}

func TestRunEveryRegisteredParser(t *testing.T) {
	// Exercise every registry entry so wiring mistakes (nil parser, panic)
	// are caught even when the input is not parseable.
	for name, entry := range parsers.Registry {
		t.Run(name, func(t *testing.T) {
			rsa, err := xio.NewReadSeekerAt(bytes.NewReader(nil))
			require.NoError(t, err)
			assert.NotPanics(t, func() {
				_, _, _ = entry.Parse(context.Background(), rsa)
			})
		})
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-version"}, &stdout, &stderr)
	assert.Equal(t, 0, code)
	assert.Equal(t, app.Version()+"\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "Usage: dep-parser")
	assert.Contains(t, stderr.String(), "golang-mod")
	assert.Empty(t, stdout.String())
}

func TestRunBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-nope"}, &stdout, &stderr)
	assert.Equal(t, 2, code)
}

func TestRunMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"/nonexistent/path/go.mod"}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "error opening")
}

func TestRunUnknownLang(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-lang", "bogus", "go.mod"}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "unknown parser: bogus")
}

func TestRunAutoDetectParses(t *testing.T) {
	// Nested path exercises basename-based detection.
	var stdout, stderr bytes.Buffer
	code := run([]string{filepath.Join("..", "..", "go.mod")}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var out output
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))
	require.NotNil(t, out.Packages)
}

func TestRunExplicitLang(t *testing.T) {
	lockfile := filepath.Join("..", "..", "pkg", "parser", "nodejs", "npm", "testdata", "package-lock_v3.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-lang", "nodejs-npm", lockfile}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var out output
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))
	assert.NotNil(t, out.Packages)
}

func TestRunCycloneDXOutput(t *testing.T) {
	lockfile := filepath.Join("..", "..", "pkg", "parser", "nodejs", "npm", "testdata", "package-lock_v3.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-lang", "nodejs-npm", "-o", "cyclonedx", lockfile}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var doc map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, "CycloneDX", doc["bomFormat"])
	assert.NotEmpty(t, doc["components"])
}

func TestRunSPDXOutput(t *testing.T) {
	lockfile := filepath.Join("..", "..", "pkg", "parser", "nodejs", "npm", "testdata", "package-lock_v3.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-lang", "nodejs-npm", "-o", "spdx", lockfile}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var doc map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, "SPDX-2.3", doc["spdxVersion"])
}

func TestRunBadFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", "bogus", filepath.Join("..", "..", "go.mod")}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "unknown output format")
}

func TestRunParseError(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "package-lock.json")
	require.NoError(t, os.WriteFile(bad, []byte("{not json"), 0o644))

	var stdout, stderr bytes.Buffer
	code := run([]string{bad}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "error parsing")
}

func TestRunNoParserForFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "random.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o644))

	var stdout, stderr bytes.Buffer
	code := run([]string{f}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "could not auto-select parser")
}

func TestRunScan(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module example.com/x\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("nothing"), 0o644))

	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", dir}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var rep struct {
		Files   int `json:"files"`
		Results []struct {
			Path     string          `json:"path"`
			Parser   string          `json:"parser"`
			Packages []types.Package `json:"packages"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &rep))
	assert.Equal(t, 2, rep.Files)
	require.Len(t, rep.Results, 1)
	assert.Equal(t, "golang-mod", rep.Results[0].Parser)
	assert.NotEmpty(t, rep.Results[0].Packages)
}

func TestRunScanMissingDir(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "/nonexistent/dir"}, &stdout, &stderr)
	assert.Equal(t, 1, code)
}

func TestRunScanNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan"}, &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "Usage: dep-parser scan")
}

func TestParserNamesSorted(t *testing.T) {
	names := parsers.Names()
	require.Len(t, names, len(parsers.Registry))
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i])
	}
}

func TestToPackages(t *testing.T) {
	assert.Nil(t, toPackages(nil))
	assert.Nil(t, toPackages([]any{"x"}))
	pkgs := []types.Package{{Name: "a"}}
	assert.Equal(t, pkgs, toPackages(pkgs))
	assert.Equal(t, pkgs, toPackages(types.Packages(pkgs)))
}

func TestToDependencies(t *testing.T) {
	assert.Nil(t, toDependencies(nil))
	deps := []types.Dependency{{ID: "a"}}
	assert.Equal(t, deps, toDependencies(deps))
	assert.Equal(t, deps, toDependencies(types.Dependencies(deps)))
}

func TestParserCandidatesExplicit(t *testing.T) {
	c, msg := parserCandidates("x", "nodejs-npm", nil)
	require.Empty(t, msg)
	assert.Equal(t, []string{"nodejs-npm"}, c)

	_, msg = parserCandidates("x", "bogus", nil)
	assert.Contains(t, msg, "unknown parser")
}

func TestParseFileMissing(t *testing.T) {
	_, _, _, err := parseFile(context.Background(), "/nonexistent", []string{"golang-mod"})
	require.Error(t, err)
}

func TestParseFileSkipsUnknownCandidates(t *testing.T) {
	gomod := filepath.Join("..", "..", "go.mod")
	_, _, name, err := parseFile(context.Background(), gomod, []string{"bogus", "golang-mod"})
	require.NoError(t, err)
	assert.Equal(t, "golang-mod", name)
}

func TestParseFileAllFail(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "x.txt")
	require.NoError(t, os.WriteFile(bad, []byte("not a lockfile"), 0o644))
	_, _, _, err := parseFile(context.Background(), bad, []string{"nodejs-npm"})
	require.Error(t, err)
}
