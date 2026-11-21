# dep-parser 🚀  
*A powerful and extensible dependency parser for multiple programming languages.*

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Build](https://img.shields.io/github/actions/workflow/status/khulnasoft/dep-parser/test.yml)

---

## 🌟 Features  
✅ Supports ~30 package managers and lockfile formats across ecosystems:

| Ecosystem | Parsers |
|---|---|
| Node.js | npm (`package-lock.json`), pnpm, yarn, bun, `package.json` |
| Go | `go.mod`, `go.sum`, compiled binaries (ELF/Mach-O/PE) |
| Rust | `Cargo.lock`, compiled binaries |
| Python | pip, pipenv, poetry, pylock, uv, `pyproject.toml`, installed `METADATA`/`PKG-INFO` |
| Java/JVM | `pom.xml`, JAR manifests, Gradle & sbt lockfiles |
| .NET | NuGet (`packages.lock.json`, `.csproj`/`packages.config`, `.deps.json`) |
| Ruby | Bundler (`Gemfile.lock`), gemspec |
| PHP | Composer (`composer.lock`) |
| Swift | `Package.resolved`, CocoaPods |
| Others | Dart pub, Hex/mix, Julia Manifest, Conan, Conda, WordPress plugins |

✅ Structured project layout: `pkg/parser/<ecosystem>/<tool>`
✅ Location-aware results (line numbers in lockfiles, file paths in archives)
✅ Dependency graph output (`DependsOn`) alongside flat package lists
✅ Automated, testdata-driven tests for every parser
✅ **SBOM export** — CycloneDX 1.5 and SPDX 2.3 JSON (`-o cyclonedx|spdx`)
✅ **Recursive scanning** — `dep-parser scan <dir>` auto-detects lockfiles by name and content (ELF/Mach-O/PE/JAR magic)
✅ **Package URL (purl) enrichment** for every ecosystem
✅ **License resolution** from npm, PyPI, and Maven Central (`-enrich`)
✅ **Dependency graph utilities** — transitive closure, affected-by, shortest paths (`pkg/graph`)
✅ Fuzz tests for representative parsers (`go test -fuzz=FuzzParse ./...`)

## 📦 Installation

```bash
go get github.com/khulnasoft/dep-parser
```

Or build the CLI from source:

```bash
go build -o dep-parser ./cmd/dep-parser
```

## 🚀 Usage

### CLI

```bash
# Auto-detect the parser from the file name
dep-parser package-lock.json

# Or select a parser explicitly
dep-parser -lang golang-mod go.mod

# SBOM output
dep-parser -o cyclonedx package-lock.json > sbom.cdx.json
dep-parser -o spdx go.mod > sbom.spdx.json

# Recursively scan a project for all lockfiles/binaries
dep-parser scan ./my-project

# Resolve missing licenses from public registries
dep-parser -enrich package-lock.json
```

Output formats: `json` (default), `cyclonedx`, `spdx`. Scan results include
per-file parser, packages, dependency edges, and per-file errors.

### Go library

```go
package main

import (
	"context"
	"os"

	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/npm"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

func main() {
	f, err := os.Open("package-lock.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rsa, err := xio.NewReadSeekerAt(f)
	if err != nil {
		panic(err)
	}

	pkgs, deps, err := npm.NewParser().Parse(context.Background(), rsa)
	if err != nil {
		panic(err)
	}

	for _, p := range pkgs {
		println(p.Name, p.Version)
	}
	_ = deps
}
```

Each parser lives under `pkg/parser/<ecosystem>/<tool>` and follows the same
`NewParser(...).Parse(ctx, reader)` convention, returning
`([]types.Package, []types.Dependency, error)`.

## 🏗️ Project Structure

```
cmd/dep-parser/        CLI entrypoint (registry, scan, SBOM output)
pkg/
  types/               Core model: Package, Dependency, Library, Location
  parser/              One package per lockfile format, grouped by ecosystem
  parsers/             Canonical parser registry + LangType metadata
  detect/              Filename + magic-byte parser detection
  scan/                Recursive directory scanner
  sbom/                CycloneDX 1.5 and SPDX 2.3 exporters
  enrich/              purl generation + registry license resolution
  graph/               Dependency graph utils (transitive, paths, roots)
  x/                   Internal helpers (io, json, http, os, path, ...)
  licensing/           License file detection
```

## 🧪 Testing

```bash
go test ./...
```

Parser tests are table-driven against files in each package's `testdata/`
directory. The full quality gate (lint, license check, coverage) is available
via `make`:

```bash
make bootstrap   # install tooling into ./.tmp
make all         # static analysis + unit tests
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
