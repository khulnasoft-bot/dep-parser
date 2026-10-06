# dep-parser 🚀  
*A powerful and extensible dependency parser for multiple programming languages.*

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Build](https://img.shields.io/github/actions/workflow/status/khulnasoft/dep-parser/test.yml)
![Version](https://img.shields.io/github/v/release/khulnasoft/dep-parser)

---

## 🌟 Features  
✅ Supports multiple package managers (npm, pnpm, yarn, etc.)  
✅ Structured project layout inspired by modern package managers  
✅ Automated test cases & test data  
✅ Highly extensible & developer-friendly

## 📦 Installation

### Go Version
```bash
go get github.com/khulnasoft/dep-parser
```

### Rust Version
```bash
cargo add dep-parser
```

Or build from source:
```bash
cargo build --release
```

## 🚀 Usage

### Rust Example
```rust
use dep_parser::rust::cargo::parse::CargoParser;
use dep_parser::types::Parser;
use std::fs::File;

fn main() {
    let mut f = File::open("Cargo.lock").expect("Failed to open file");
    let parser = CargoParser::new();
    let (libs, deps) = parser.parse(&mut f).expect("Failed to parse");
    
    println!("Libraries: {:?}", libs);
    println!("Dependencies: {:?}", deps);
}
```

## 🏗️ Project Structure

This project is now available in both Go and Rust implementations:

- **Go**: Original implementation in `pkg/` directory
- **Rust**: New implementation in `src/` directory

### Rust Modules
- `types` - Core data structures (Library, Dependency, Location, etc.)
- `io` - I/O traits and utilities
- `utils` - Helper functions for package ID generation, deduplication
- `rust` - Rust-specific parsers (Cargo.lock, binary)
- `golang` - Go-specific parsers (go.mod, go.sum, binary)
- `nodejs` - Node.js parsers (npm, yarn, pnpm)
- `python` - Python parsers (pip, poetry, requirements)
- `java` - Java parsers (JAR, POM)
- Other language parsers (php, ruby, dart, dotnet, conda, gradle, hex, julia, sbt, swift, c, frameworks)

## 🧪 Testing

### Rust Tests
```bash
cargo test
```

### Go Tests
```bash
go test ./...
```

## 📝 Development Status

- ✅ Core types and utilities converted to Rust
- ✅ Rust/Cargo parser fully implemented and tested
- ⚠️ Other language parsers have placeholder implementations (return "not yet implemented" errors)
- 🔄 Binary parser requires Rust equivalent of go-rustaudit

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.  
