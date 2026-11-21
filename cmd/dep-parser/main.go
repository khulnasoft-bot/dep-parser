package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/khulnasoft/dep-parser/pkg/detect"
	"github.com/khulnasoft/dep-parser/pkg/enrich"
	"github.com/khulnasoft/dep-parser/pkg/parsers"
	"github.com/khulnasoft/dep-parser/pkg/sbom"
	"github.com/khulnasoft/dep-parser/pkg/scan"
	"github.com/khulnasoft/dep-parser/pkg/types"
	"github.com/khulnasoft/dep-parser/pkg/version/app"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dep-parser", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lang := fs.String("lang", "", "parser to use (e.g. nodejs-npm, python-pyproject, golang-mod)")
	format := fs.String("o", "json", "output format: json, cyclonedx, or spdx")
	enrichLicenses := fs.Bool("enrich", false, "resolve missing licenses from public registries (npm, PyPI, Maven Central)")
	version := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *version {
		writef(stdout, "%s\n", app.Version())
		return 0
	}

	rest := fs.Args()
	if len(rest) == 0 {
		usage(stderr)
		return 1
	}

	ctx := context.Background()

	// scan subcommand: dep-parser scan <dir>
	if rest[0] == "scan" {
		if len(rest) < 2 {
			writef(stderr, "Usage: dep-parser scan <dir> [-o json|cyclonedx|spdx] [-enrich]\n")
			return 1
		}
		return runScan(ctx, rest[1], *format, *enrichLicenses, stdout, stderr)
	}

	// single-file mode: dep-parser [-lang X] [-o fmt] <file>
	filePath := rest[0]
	candidates, errMsg := parserCandidates(filePath, *lang, stderr)
	if errMsg != "" {
		writef(stderr, "%s\n", errMsg)
		return 1
	}

	pkgsAny, depsAny, parserName, err := parseFile(ctx, filePath, candidates)
	if err != nil {
		writef(stderr, "error parsing %s: %v\n", filePath, err)
		return 1
	}

	// stamp LangType so downstream consumers (purl, SBOM) know the ecosystem
	pkgs := toPackages(pkgsAny)
	for i := range pkgs {
		if pkgs[i].LangType == "" {
			pkgs[i].LangType = parsers.Registry[parserName].Lang
		}
	}
	deps := toDependencies(depsAny)

	if *enrichLicenses {
		enrichPackages(ctx, pkgs, parsers.Registry[parserName].Lang, stderr)
	}

	return writeOutput(stdout, stderr, *format, filePath, pkgsAny, depsAny, pkgs, deps, parserName)
}

func usage(w io.Writer) {
	writef(w, "Usage: dep-parser [-lang <parser>] [-o json|cyclonedx|spdx] [-enrich] <file>\n")
	writef(w, "       dep-parser scan [-o json|cyclonedx|spdx] [-enrich] <dir>\n")
	writef(w, "       dep-parser -version\n")
	writef(w, "Available parsers:\n")
	for _, name := range parsers.Names() {
		writef(w, "  %s\n", name)
	}
}

// parserCandidates resolves which parser names to try for a file.
func parserCandidates(filePath, lang string, stderr io.Writer) ([]string, string) {
	if lang != "" {
		if _, ok := parsers.Registry[lang]; !ok {
			return nil, fmt.Sprintf("unknown parser: %s\nAvailable parsers:\n%s", lang, parserList())
		}
		return []string{lang}, ""
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Sprintf("error opening %s: %v", filePath, err)
	}
	defer func() { _ = f.Close() }()
	header := make([]byte, 512)
	n, _ := f.Read(header)
	candidates := detect.Detect(filePath, header[:n])
	if len(candidates) == 0 {
		return nil, fmt.Sprintf("could not auto-select parser for %s\nUse -lang to specify a parser explicitly.", filePath)
	}
	return candidates, ""
}

func parserList() string {
	var b []byte
	for _, name := range parsers.Names() {
		b = append(b, "  "...)
		b = append(b, name...)
		b = append(b, '\n')
	}
	return string(b)
}

// parseFile tries each candidate parser in order until one succeeds.
func parseFile(ctx context.Context, filePath string, candidates []string) (pkgsAny, depsAny any, parserName string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, "", err
	}
	defer func() { _ = f.Close() }()

	var lastErr error
	for _, name := range candidates {
		entry, ok := parsers.Registry[name]
		if !ok {
			continue
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, nil, "", err
		}
		rsa, err := xio.NewReadSeekerAt(f)
		if err != nil {
			lastErr = err
			continue
		}
		pkgsAny, depsAny, err = entry.Parse(ctx, rsa)
		if err != nil {
			lastErr = err
			continue
		}
		return pkgsAny, depsAny, name, nil
	}
	return nil, nil, "", lastErr
}

// toPackages normalizes []types.Package / types.Packages.
func toPackages(v any) []types.Package {
	switch p := v.(type) {
	case []types.Package:
		return p
	case types.Packages:
		return []types.Package(p)
	default:
		return nil
	}
}

// toDependencies normalizes []types.Dependency / types.Dependencies.
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

// enrichPackages fills in missing licenses via public registries.
func enrichPackages(ctx context.Context, pkgs []types.Package, lang types.LangType, stderr io.Writer) {
	resolver := enrich.NewRegistry(nil)
	for i := range pkgs {
		if len(pkgs[i].Licenses) > 0 {
			continue
		}
		lic, err := resolver.License(ctx, pkgs[i], lang)
		if err != nil {
			writef(stderr, "warning: license lookup failed for %s: %v\n", pkgs[i].Name, err)
			continue
		}
		if lic != "" {
			pkgs[i].Licenses = []string{lic}
		}
	}
}

// writeOutput emits the parse result in the requested format.
func writeOutput(stdout, stderr io.Writer, format, filePath string, pkgsAny, depsAny any, pkgs []types.Package, deps []types.Dependency, parserName string) int {
	switch format {
	case "json", "":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output{Packages: pkgsAny, Deps: depsAny}); err != nil {
			writef(stderr, "error encoding output: %v\n", err)
			return 1
		}
		return 0
	case "cyclonedx", "spdx":
		if pkgs == nil {
			writef(stderr, "format %q requires a lockfile parser (got config-style output from %s)\n", format, parserName)
			return 1
		}
		lang := parsers.Registry[parserName].Lang
		meta := sbom.Metadata{
			Name:        filepath.Base(filePath),
			ToolVersion: app.Version(),
			Lang:        lang,
		}
		out, err := sbom.Marshal(sbom.Format(format), pkgs, deps, meta)
		if err != nil {
			writef(stderr, "error generating %s: %v\n", format, err)
			return 1
		}
		if _, err := stdout.Write(append(out, '\n')); err != nil {
			writef(stderr, "error writing output: %v\n", err)
			return 1
		}
		return 0
	default:
		writef(stderr, "unknown output format: %s (want json, cyclonedx, or spdx)\n", format)
		return 1
	}
}

// runScan handles `dep-parser scan <dir>`.
func runScan(ctx context.Context, dir, format string, doEnrich bool, stdout, stderr io.Writer) int {
	rep, err := scan.Scan(ctx, dir, scan.Options{})
	if err != nil {
		writef(stderr, "error scanning %s: %v\n", dir, err)
		return 1
	}

	if doEnrich {
		resolver := enrich.NewRegistry(nil)
		for _, res := range rep.Results {
			for i := range res.Packages {
				if len(res.Packages[i].Licenses) > 0 {
					continue
				}
				lic, err := resolver.License(ctx, res.Packages[i], res.Lang)
				if err == nil && lic != "" {
					res.Packages[i].Licenses = []string{lic}
				}
			}
		}
	}

	switch format {
	case "json", "":
		results := make([]scanResultJSON, 0, len(rep.Results))
		for _, res := range rep.Results {
			results = append(results, scanResultJSON{
				Path:         res.Path,
				Parser:       res.Parser,
				Lang:         res.Lang,
				Packages:     res.Packages,
				Dependencies: res.Dependencies,
				Config:       res.Config,
			})
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(scanReportJSON{
			Files:   rep.Files,
			Results: results,
			Errors:  rep.Errors,
		}); err != nil {
			writef(stderr, "error encoding output: %v\n", err)
			return 1
		}
		return 0
	case "cyclonedx", "spdx":
		pkgs := rep.AllPackages()
		deps := rep.AllDependencies()
		if len(pkgs) == 0 {
			writef(stderr, "no packages found under %s\n", dir)
			return 1
		}
		meta := sbom.Metadata{
			Name:        filepath.Base(dir),
			ToolVersion: app.Version(),
		}
		out, err := sbom.Marshal(sbom.Format(format), pkgs, deps, meta)
		if err != nil {
			writef(stderr, "error generating %s: %v\n", format, err)
			return 1
		}
		if _, err := stdout.Write(append(out, '\n')); err != nil {
			writef(stderr, "error writing output: %v\n", err)
			return 1
		}
		return 0
	default:
		writef(stderr, "unknown output format: %s (want json, cyclonedx, or spdx)\n", format)
		return 1
	}
}

// scanReportJSON is the JSON shape for scan results.
type scanReportJSON struct {
	Files   int              `json:"files"`
	Results []scanResultJSON `json:"results"`
	Errors  []scan.FileError `json:"errors,omitempty"`
}

type scanResultJSON struct {
	Path         string             `json:"path"`
	Parser       string             `json:"parser"`
	Lang         types.LangType     `json:"lang,omitempty"`
	Packages     []types.Package    `json:"packages,omitempty"`
	Dependencies []types.Dependency `json:"dependencies,omitempty"`
	Config       any                `json:"config,omitempty"`
}

type output struct {
	Packages any `json:"packages,omitempty"`
	Deps     any `json:"dependencies,omitempty"`
}

// writef writes formatted output; a failed write to a CLI stream is not actionable.
func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
