package jar

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

var ArtifactNotFoundErr = xerrors.New("no artifact found")

type Properties struct {
	GroupID    string
	ArtifactID string
	Version    string
	FilePath   string // path to file containing these props
}

func (p Properties) Package() types.Package {
	return types.Package{
		Name:     packageName(p.GroupID, p.ArtifactID),
		Version:  p.Version,
		FilePath: p.FilePath,
	}
}

// packageName returns the package name in "groupID:artifactID" format.
func packageName(groupID, artifactID string) string {
	return groupID + ":" + artifactID
}

func (p Properties) Valid() bool {
	return p.GroupID != "" && p.ArtifactID != "" && p.Version != ""
}

func (p Properties) String() string {
	return fmt.Sprintf("%s:%s:%s", p.GroupID, p.ArtifactID, p.Version)
}
