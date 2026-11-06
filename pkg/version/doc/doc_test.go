package doc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	assert.Equal(t, "https://dep-parser.dev/docs/latest/go.mod", URL("go.mod", ""))
	assert.Equal(t, "https://dep-parser.dev/docs/latest/go.mod#section", URL("go.mod", "#section"))
	assert.Equal(t, "https://dep-parser.dev/docs/latest/", URL("", ""))
}
