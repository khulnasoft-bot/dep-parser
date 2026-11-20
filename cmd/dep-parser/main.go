package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/khulnasoft/dep-parser/pkg/parser/c/conan"
	"github.com/khulnasoft/dep-parser/pkg/parser/conda/meta"
	"github.com/khulnasoft/dep-parser/pkg/parser/dart/pub"
	"github.com/khulnasoft/dep-parser/pkg/parser/dotnet/core_deps"
	gobinary "github.com/khulnasoft/dep-parser/pkg/parser/golang/binary"
	"github.com/khulnasoft/dep-parser/pkg/parser/golang/mod"
	"github.com/khulnasoft/dep-parser/pkg/parser/golang/sum"
	gradlelock "github.com/khulnasoft/dep-parser/pkg/parser/gradle/lockfile"
	"github.com/khulnasoft/dep-parser/pkg/parser/hex/mix"
	"github.com/khulnasoft/dep-parser/pkg/parser/java/pom"
	juliamanifest "github.com/khulnasoft/dep-parser/pkg/parser/julia/manifest"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/bun"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/npm"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/packagejson"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/pnpm"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/yarn"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/config"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/lock"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/packagesprops"
	"github.com/khulnasoft/dep-parser/pkg/parser/php/composer"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/packaging"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pip"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pipenv"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/poetry"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pylock"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pyproject"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/uv"
	"github.com/khulnasoft/dep-parser/pkg/parser/ruby/bundler"
	"github.com/khulnasoft/dep-parser/pkg/parser/ruby/gemspec"
	rustbinary "github.com/khulnasoft/dep-parser/pkg/parser/rust/binary"
	"github.com/khulnasoft/dep-parser/pkg/parser/rust/cargo"
	sbtlock "github.com/khulnasoft/dep-parser/pkg/parser/sbt/lockfile"
	"github.com/khulnasoft/dep-parser/pkg/parser/swift/cocoapods"
	"github.com/khulnasoft/dep-parser/pkg/parser/swift/swift"
	"github.com/khulnasoft/dep-parser/pkg/version/app"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type parseFunc func(context.Context, xio.ReadSeekerAt) (any, any, error)

var parserRegistry = map[string]parseFunc{
	"conan": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := conan.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"conda-meta": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := meta.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"dart-pub": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pub.NewParser(false).Parse(ctx, r)
		return pkgs, deps, err
	},
	"dotnet": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := core_deps.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"golang-binary": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := gobinary.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"golang-mod": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, _, err := mod.NewParser(false, false).Parse(ctx, r)
		return pkgs, deps, err
	},
	"golang-sum": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := sum.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"gradle": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := gradlelock.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"hex-mix": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := mix.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"java-pom": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pom.NewParser("").Parse(ctx, r)
		return pkgs, deps, err
	},
	"julia": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := juliamanifest.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nodejs-bun": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := bun.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nodejs-npm": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := npm.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nodejs-pkg": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkg, err := packagejson.NewParser().Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return []any{pkg}, nil, nil
	},
	"nodejs-pnpm": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pnpm.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nodejs-yarn": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, _, err := yarn.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nuget-config": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := config.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nuget-lock": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := lock.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"nuget-props": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := packagesprops.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"php-composer": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := composer.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-pkg": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := packaging.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-pip": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pip.NewParser(false).Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-pipenv": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pipenv.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-poetry": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := poetry.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-pylock": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := pylock.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"python-pyproject": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		conf, err := pyproject.NewParser().Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return []any{conf}, nil, nil
	},
	"python-uv": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := uv.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"ruby-bundler": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := bundler.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"ruby-gemspec": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := gemspec.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"rust-binary": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := rustbinary.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"rust-cargo": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := cargo.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"sbt": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := sbtlock.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"swift-cocoapods": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := cocoapods.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
	"swift": func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, err := swift.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	},
}

type output struct {
	Packages any `json:"packages,omitempty"`
	Deps     any `json:"dependencies,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dep-parser", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lang := fs.String("lang", "", "parser to use (e.g. nodejs-npm, python-pyproject, golang-mod)")
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
		writef(stderr, "Usage: dep-parser [-lang <parser>] <file>\n")
		writef(stderr, "Available parsers:\n")
		for _, name := range parserNames() {
			writef(stderr, "  %s\n", name)
		}
		return 1
	}

	filePath := rest[0]

	var parse parseFunc
	if *lang != "" {
		var ok bool
		parse, ok = parserRegistry[*lang]
		if !ok {
			writef(stderr, "unknown parser: %s\n", *lang)
			writef(stderr, "Available parsers:\n")
			for _, name := range parserNames() {
				writef(stderr, "  %s\n", name)
			}
			return 1
		}
	} else {
		parse = autoSelect(filePath)
	}
	if parse == nil {
		writef(stderr, "could not auto-select parser for %s\n", filePath)
		writef(stderr, "Use -lang to specify a parser explicitly.\n")
		return 1
	}

	f, err := os.Open(filePath)
	if err != nil {
		writef(stderr, "error opening %s: %v\n", filePath, err)
		return 1
	}
	defer func() {
		if err := f.Close(); err != nil {
			writef(stderr, "error closing %s: %v\n", filePath, err)
		}
	}()

	rsa, err := xio.NewReadSeekerAt(f)
	if err != nil {
		writef(stderr, "error wrapping reader: %v\n", err)
		return 1
	}

	ctx := context.Background()
	pkgs, deps, err := parse(ctx, rsa)
	if err != nil {
		writef(stderr, "error parsing %s: %v\n", filePath, err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output{Packages: pkgs, Deps: deps}); err != nil {
		writef(stderr, "error encoding output: %v\n", err)
		return 1
	}
	return 0
}

// writef writes formatted output; a failed write to a CLI stream is not actionable.
func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func parserNames() []string {
	names := make([]string, 0, len(parserRegistry))
	for name := range parserRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func autoSelect(path string) parseFunc {
	switch filepath.Base(path) {
	case "go.mod":
		return parserRegistry["golang-mod"]
	case "go.sum":
		return parserRegistry["golang-sum"]
	case "package-lock.json":
		return parserRegistry["nodejs-npm"]
	case "package.json":
		return parserRegistry["nodejs-pkg"]
	case "pnpm-lock.yaml":
		return parserRegistry["nodejs-pnpm"]
	case "yarn.lock":
		return parserRegistry["nodejs-yarn"]
	case "Pipfile":
		return parserRegistry["python-pipenv"]
	case "pyproject.toml":
		return parserRegistry["python-pyproject"]
	case "poetry.lock":
		return parserRegistry["python-poetry"]
	case "uv.lock":
		return parserRegistry["python-uv"]
	case "Cargo.lock":
		return parserRegistry["rust-cargo"]
	case "pom.xml":
		return parserRegistry["java-pom"]
	case "composer.lock":
		return parserRegistry["php-composer"]
	case "Gemfile.lock":
		return parserRegistry["ruby-bundler"]
	case "Package.resolved":
		return parserRegistry["swift"]
	case "requirements.txt":
		return parserRegistry["python-pip"]
	default:
		return nil
	}
}
