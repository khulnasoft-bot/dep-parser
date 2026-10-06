package conan

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

// failReader fails on Read so the io.ReadAll error path is exercised.
type failReader struct{}

func (failReader) Read([]byte) (int, error)          { return 0, xerrors.New("read boom") }
func (failReader) Seek(int64, int) (int64, error)    { return 0, nil }
func (failReader) ReadAt([]byte, int64) (int, error) { return 0, nil }

func TestParseReadError(t *testing.T) {
	_, _, err := NewParser().Parse(failReader{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read canon lock file")
}

func TestParseDecodeError(t *testing.T) {
	_, _, err := NewParser().Parse(bytes.NewReader([]byte("this is not valid {json}")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode canon lock file")
}
