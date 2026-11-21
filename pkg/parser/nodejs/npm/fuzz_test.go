package npm

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

// FuzzParse exercises the npm lockfile parser against arbitrary input.
// Run with: go test -fuzz=FuzzParse -fuzztime=30s ./pkg/parser/nodejs/npm/
func FuzzParse(f *testing.F) {
	// Seed with the testdata corpus.
	matches, err := filepath.Glob("testdata/*.json")
	if err == nil {
		for _, m := range matches {
			if data, err := os.ReadFile(m); err == nil {
				f.Add(data)
			}
		}
	}
	f.Add([]byte(`{"lockfileVersion":3,"packages":{"":{"dependencies":{"a":"1.0.0"}},"node_modules/a":{"version":"1.0.0"}}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"lockfileVersion":1,"dependencies":{"a":{"version":"1.0.0","requires":{"b":"^2.0.0"}}}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		rsa, err := xio.NewReadSeekerAt(bytes.NewReader(data))
		if err != nil {
			t.Skip(err)
		}
		// Must never panic; errors are fine.
		_, _, _ = NewParser().Parse(context.Background(), rsa)
	})
}
