// Package scan recursively scans a directory tree for dependency files and
// parses each one with the matching parser from pkg/parsers.
package scan

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/khulnasoft/dep-parser/pkg/detect"
	"github.com/khulnasoft/dep-parser/pkg/parsers"
	"github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

// Options controls directory scanning.
type Options struct {
	// ExcludeDirs lists directory base names to skip entirely.
	// Defaults to common VCS/dependency dirs when nil.
	ExcludeDirs []string
	// MaxFileSize caps files considered for parsing (0 = unlimited).
	// Defaults to 256 MiB.
	MaxFileSize int64
	// HeaderSize is how many leading bytes are read for content sniffing.
	HeaderSize int
}

func (o *Options) withDefaults() Options {
	out := *o
	if out.ExcludeDirs == nil {
		out.ExcludeDirs = []string{".git", "node_modules", ".svn", ".hg", "vendor", ".venv", "venv"}
	}
	if out.MaxFileSize == 0 {
		out.MaxFileSize = 256 << 20
	}
	if out.HeaderSize == 0 {
		out.HeaderSize = 512
	}
	return out
}

// Result is the parse outcome for a single file.
type Result struct {
	// Path is the file path relative to the scan root (or absolute if root was).
	Path string
	// Parser is the registry name of the parser that succeeded.
	Parser string
	// Lang is the parser's ecosystem.
	Lang types.LangType
	// Packages holds []types.Package for lockfile parsers, or nil.
	Packages []types.Package
	// Dependencies holds []types.Dependency for lockfile parsers, or nil.
	Dependencies []types.Dependency
	// Config holds the parsed document for config-style parsers
	// (package.json, pyproject.toml), or nil.
	Config any
}

// FileError records a file that was detected but failed to parse with all
// candidate parsers.
type FileError struct {
	Path   string
	Parser string // best-guess parser name, if any
	Err    error
}

func (e FileError) Error() string {
	return fmt.Sprintf("%s (%s): %v", e.Path, e.Parser, e.Err)
}

// Report summarizes a scan.
type Report struct {
	Results []Result
	Errors  []FileError
	// Skipped counts files that matched a parser name but were not attempted
	// is not tracked; Files counts all regular files visited.
	Files int
}

// Scan walks root recursively, detects dependency files, and parses them.
// It returns a Report; individual file failures are collected in Errors and
// do not abort the scan.
func Scan(ctx context.Context, root string, opts Options) (*Report, error) {
	o := opts.withDefaults()
	exclude := make(map[string]struct{}, len(o.ExcludeDirs))
	for _, d := range o.ExcludeDirs {
		exclude[d] = struct{}{}
	}

	rep := &Report{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if path != root {
				if _, skip := exclude[d.Name()]; skip {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rep.Files++
		res, ferr := scanFile(ctx, path, o)
		if res != nil {
			rep.Results = append(rep.Results, *res)
		}
		if ferr != nil {
			rep.Errors = append(rep.Errors, *ferr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(rep.Results, func(i, j int) bool { return rep.Results[i].Path < rep.Results[j].Path })
	return rep, nil
}

// scanFile attempts to parse a single file. It returns (result, fileError);
// exactly one is non-nil unless the file was not a dependency file at all.
func scanFile(ctx context.Context, path string, o Options) (*Result, *FileError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, &FileError{Path: path, Err: err}
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, &FileError{Path: path, Err: err}
	}
	if o.MaxFileSize > 0 && info.Size() > o.MaxFileSize {
		return nil, nil
	}

	header := make([]byte, o.HeaderSize)
	n, _ := io.ReadFull(f, header)
	header = header[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, &FileError{Path: path, Err: err}
	}

	candidates := detect.Detect(path, header)
	if len(candidates) == 0 {
		return nil, nil
	}

	var lastErr error
	var lastName string
	for _, name := range candidates {
		entry, ok := parsers.Registry[name]
		if !ok {
			continue
		}
		lastName = name
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, &FileError{Path: path, Parser: name, Err: err}
		}
		rsa, err := xio.NewReadSeekerAt(f)
		if err != nil {
			lastErr = err
			continue
		}
		pkgsAny, depsAny, err := entry.Parse(ctx, rsa)
		if err != nil {
			lastErr = err
			continue
		}
		res := &Result{Path: path, Parser: name, Lang: entry.Lang}
		if pkgs, ok := toPackages(pkgsAny); ok {
			res.Packages = pkgs
			res.Dependencies = toDependencies(depsAny)
		} else if pkgsAny != nil {
			res.Config = pkgsAny
		}
		return res, nil
	}
	if lastErr != nil {
		return nil, &FileError{Path: path, Parser: lastName, Err: lastErr}
	}
	return nil, nil
}

// toPackages normalizes the various package slice types parsers return
// ([]types.Package, types.Packages) into []types.Package.
func toPackages(v any) ([]types.Package, bool) {
	switch p := v.(type) {
	case []types.Package:
		return p, true
	case types.Packages:
		return []types.Package(p), true
	case nil:
		return nil, false
	default:
		return nil, false
	}
}

// toDependencies normalizes dependency slice types ([]types.Dependency,
// types.Dependencies).
func toDependencies(v any) []types.Dependency {
	switch d := v.(type) {
	case []types.Dependency:
		return d
	case types.Dependencies:
		return []types.Dependency(d)
	default:
		return nil
	}
}

// Packages flattens all []types.Package results across the report.
func (r *Report) AllPackages() []types.Package {
	var out []types.Package
	for _, res := range r.Results {
		out = append(out, res.Packages...)
	}
	return out
}

// AllDependencies flattens all dependency edges. Edges are namespaced by
// file path is not possible (IDs may collide across ecosystems); callers
// needing per-file graphs should iterate Results directly.
func (r *Report) AllDependencies() []types.Dependency {
	var out []types.Dependency
	for _, res := range r.Results {
		out = append(out, res.Dependencies...)
	}
	return out
}

// Summary returns a human-readable one-line-per-file summary.
func (r *Report) Summary() string {
	var b strings.Builder
	for _, res := range r.Results {
		fmt.Fprintf(&b, "%s\t%s\t%d packages\n", res.Path, res.Parser, len(res.Packages))
	}
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "%s\tERROR\t%v\n", e.Path, e.Err)
	}
	return b.String()
}
