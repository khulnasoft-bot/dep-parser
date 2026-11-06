package utils

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBinary(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		isBin, err := IsBinary(strings.NewReader("hello world"), 100)
		require.NoError(t, err)
		assert.False(t, isBin)
	})

	t.Run("binary content", func(t *testing.T) {
		isBin, err := IsBinary(bytes.NewReader([]byte{0, 1, 2, 3}), 100)
		require.NoError(t, err)
		assert.True(t, isBin)
	})

	t.Run("empty reader", func(t *testing.T) {
		isBin, err := IsBinary(strings.NewReader(""), 100)
		require.NoError(t, err)
		assert.False(t, isBin)
	})

	t.Run("read error", func(t *testing.T) {
		_, err := IsBinary(&errReader{}, 100)
		assert.Error(t, err)
	})

	t.Run("short read", func(t *testing.T) {
		isBin, err := IsBinary(strings.NewReader("hi"), 100)
		require.NoError(t, err)
		assert.False(t, isBin)
	})
}

type errReader struct{}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

var _ io.Reader = (*errReader)(nil)
