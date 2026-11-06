package types

import (
	"slices"

	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type Library struct {
	ID                 string `json:",omitempty"`
	Name               string
	Version            string
	Dev                bool
	Indirect           bool          `json:",omitempty"`
	License            string        `json:",omitempty"`
	ExternalReferences []ExternalRef `json:",omitempty"`
	Locations          Locations     `json:",omitempty"`
	FilePath           string        `json:",omitempty"` // Required to show nested jars
}

type Libraries []Library

func (libs Libraries) Len() int { return len(libs) }
func (libs Libraries) Less(i, j int) bool {
	if libs[i].ID != libs[j].ID { // ID could be empty
		return libs[i].ID < libs[j].ID
	} else if libs[i].Name != libs[j].Name { // Name could be the same
		return libs[i].Name < libs[j].Name
	}
	return libs[i].Version < libs[j].Version
}
func (libs Libraries) Swap(i, j int) { libs[i], libs[j] = libs[j], libs[i] }

// Location in lock file
type Location struct {
	StartLine int `json:",omitempty"`
	EndLine   int `json:",omitempty"`
}

type Locations []Location

func (locs Locations) Len() int { return len(locs) }
func (locs Locations) Less(i, j int) bool {
	return locs[i].StartLine < locs[j].StartLine
}
func (locs Locations) Swap(i, j int) { locs[i], locs[j] = locs[j], locs[i] }

type ExternalRef struct {
	Type RefType
	URL  string
}

type Dependency struct {
	ID        string
	DependsOn []string
}

type Dependencies []Dependency

func (deps Dependencies) Len() int { return len(deps) }
func (deps Dependencies) Less(i, j int) bool {
	if deps[i].ID != deps[j].ID {
		return deps[i].ID < deps[j].ID
	}
	return slices.Compare(deps[j].DependsOn, deps[i].DependsOn) < 0
}
func (deps Dependencies) Swap(i, j int) { deps[i], deps[j] = deps[j], deps[i] }

// Identifier uniquely identifies a package
type Identifier struct {
	UID string `json:"uid,omitempty"`
}

// Package represents a dependency package
type Package struct {
	ID                 string        `json:",omitempty"`
	Name               string        `json:",omitempty"`
	Version            string        `json:",omitempty"`
	LangType           LangType      `json:",omitempty"`
	Locations          Locations     `json:",omitempty"`
	Relationship       Relationship  `json:",omitempty"`
	DependsOn          []string      `json:",omitempty"`
	ExternalReferences []ExternalRef `json:",omitempty"`
	Licenses           []string      `json:",omitempty"`
	Dev                bool          `json:",omitempty"`
	Indirect           bool          `json:",omitempty"`
	Identifier         Identifier    `json:",omitempty"`
	Module             bool          `json:",omitempty"`
	FilePath           string        `json:",omitempty"`
}

// Packages is a slice of Package
type Packages []Package

func (pkgs Packages) Len() int { return len(pkgs) }

func relationshipRank(r Relationship) int {
	switch r {
	case RelationshipRoot:
		return 0
	case RelationshipWorkspace:
		return 1
	case RelationshipDirect:
		return 2
	case RelationshipIndirect:
		return 3
	default:
		return 4
	}
}

func (pkgs Packages) Less(i, j int) bool {
	if ri, rj := relationshipRank(pkgs[i].Relationship), relationshipRank(pkgs[j].Relationship); ri != rj {
		return ri < rj
	}
	if pkgs[i].Name != pkgs[j].Name {
		return pkgs[i].Name < pkgs[j].Name
	}
	return pkgs[i].Version < pkgs[j].Version
}
func (pkgs Packages) Swap(i, j int) { pkgs[i], pkgs[j] = pkgs[j], pkgs[i] }

type Parser interface {
	// Parse parses the dependency file
	Parse(r xio.ReadSeekerAt) ([]Library, []Dependency, error)
}

type RefType string

const (
	RefWebsite      RefType = "website"
	RefLicense      RefType = "license"
	RefVCS          RefType = "vcs"
	RefIssueTracker RefType = "issue-tracker"
	RefOther        RefType = "other"
)

// LangType represents the type of package/language
type LangType string

const (
	Conan      LangType = "conan"
	DotNetCore LangType = "dotnet-core"
	GoModule   LangType = "go-module"
	GoBinary   LangType = "go-binary"
	Jar        LangType = "jar"
	Pom        LangType = "pom"
	Gradle     LangType = "gradle"
	Sbt        LangType = "sbt"
	Conda      LangType = "conda"
	RustBinary LangType = "rust-binary"
	Cargo      LangType = "cargo"
	Swift      LangType = "swift"
	Cocoapods  LangType = "cocoapods"
	Pub        LangType = "pub"
	Hex        LangType = "hex"
	NodePkg    LangType = "node-pkg"
	Pnpm       LangType = "pnpm"
	Yarn       LangType = "yarn"
	NuGet      LangType = "nuget"
	Composer   LangType = "composer"
	Poetry     LangType = "poetry"
	PyLock     LangType = "pylock"
	Bun        LangType = "bun"
	Uv         LangType = "uv"
	Bundler    LangType = "bundler"
	Npm        LangType = "npm"
	Pip        LangType = "pip"
)

// UserError is an error that should be surfaced to the user as-is.
type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}

// Relationship represents the dependency relationship
type Relationship string

const (
	RelationshipRoot      Relationship = "root"
	RelationshipDirect    Relationship = "direct"
	RelationshipIndirect  Relationship = "indirect"
	RelationshipUnknown   Relationship = "unknown"
	RelationshipWorkspace Relationship = "workspace"
)
