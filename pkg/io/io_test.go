package io

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNopCloser(t *testing.T) {
	inner := bytes.NewReader([]byte("hello"))

	c := NopCloser(inner)

	// Close is a no-op that reports success
	require.NoError(t, c.Close())
	require.NoError(t, c.Close())

	// The wrapped ReadSeekerAt behavior is preserved
	buf := make([]byte, 5)
	n, err := c.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf))

	pos, err := c.Seek(0, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(0), pos)

	n, err = c.ReadAt(buf, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	require.NoError(t, c.Close())
}
