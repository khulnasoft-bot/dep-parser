package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/version/app"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

func TestAutoSelect(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "bare go.mod", path: "go.mod", expected: "golang-mod"},
		{name: "nested go.mod", path: "sub/dir/go.mod", expected: "golang-mod"},
		{name: "dot-prefixed go.mod", path: "./go.mod", expected: "golang-mod"},
		{name: "absolute package-lock.json", path: "/tmp/x/package-lock.json", expected: "nodejs-npm"},
		{name: "nested yarn.lock", path: "a/b/yarn.lock", expected: "nodejs-yarn"},
		{name: "unknown file", path: "README.md", expected: ""},
		{name: "similar name", path: "my-go.mod", expected: ""},
		{name: "empty", path: "", expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := autoSelect(tt.path)
			if tt.expected == "" {
				assert.Nil(t, fn)
				return
			}
			require.NotNil(t, fn)
			assert.Equal(t,
				reflect.ValueOf(parserRegistry[tt.expected]).Pointer(),
				reflect.ValueOf(fn).Pointer(),
			)
		})
	}
}

func TestParserRegistry(t *testing.T) {
	assert.NotEmpty(t, parserRegistry)
	for name, fn := range parserRegistry {
		assert.NotNil(t, fn, "parser %q is nil", name)
	}
	assert.Len(t, parserNames(), len(parserRegistry))
}

func TestRunEveryRegisteredParser(t *testing.T) {
	// Exercise every registry entry so wiring mistakes (nil parser, panic)
	// are caught even when the input is not parseable.
	for name, fn := range parserRegistry {
		t.Run(name, func(t *testing.T) {
			rsa, err := xio.NewReadSeekerAt(bytes.NewReader(nil))
			require.NoError(t, err)
			assert.NotPanics(t, func() {
				_, _, _ = fn(context.Background(), rsa)
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

func TestRunAutoSelectParses(t *testing.T) {
	// Use the repo's own go.mod via a nested path to exercise basename matching.
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

func TestParserNamesSorted(t *testing.T) {
	names := parserNames()
	require.Len(t, names, len(parserRegistry))
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i])
	}
}
