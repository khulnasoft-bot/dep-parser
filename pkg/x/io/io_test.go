package io

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	t.Run("successful copy", func(t *testing.T) {
		ctx := t.Context()
		src := strings.NewReader("hello world")
		dst := &bytes.Buffer{}

		n, err := Copy(ctx, dst, src)
		require.NoError(t, err)
		assert.Equal(t, int64(11), n)
		assert.Equal(t, "hello world", dst.String())
	})

	t.Run("context canceled before read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		src := strings.NewReader("hello world")
		dst := &bytes.Buffer{}

		n, err := Copy(ctx, dst, src)
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, int64(0), n)
		assert.Empty(t, dst.String())
	})

	t.Run("context canceled during read", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		// Create a reader that will be canceled after first read
		reader := &dummyReader{
			cancel: cancel, // Cancel after first read
		}
		dst := &bytes.Buffer{}

		n, err := Copy(ctx, dst, reader)
		require.ErrorIs(t, err, context.Canceled)
		// Should have written first chunk before cancellation
		assert.Equal(t, int64(5), n)
		assert.Equal(t, "dummy", dst.String())
	})
}

// dummyReader returns the same data on every Read call
type dummyReader struct {
	cancel context.CancelFunc
}

func (r *dummyReader) Read(p []byte) (int, error) {
	n := copy(p, "dummy")
	if r.cancel != nil {
		r.cancel() // Simulate cancellation after first read
	}
	return n, nil
}

func TestNewReadSeekerAtAlreadyImplemented(t *testing.T) {
	br := bytes.NewReader([]byte("hello"))
	rsa, err := NewReadSeekerAt(br)
	require.NoError(t, err)
	assert.Same(t, br, rsa)
}

func TestNewReadSeekerAtRegularReader(t *testing.T) {
	r := strings.NewReader("hello world")
	rsa, err := NewReadSeekerAt(r)
	require.NoError(t, err)
	assert.NotNil(t, rsa)

	// Verify it can seek and read
	data := make([]byte, 5)
	_, err = rsa.ReadAt(data, 6)
	require.NoError(t, err)
	assert.Equal(t, "world", string(data))
}

func TestNewReadSeekerAtReadError(t *testing.T) {
	rsa, err := NewReadSeekerAt(&errReader{})
	require.Error(t, err)
	assert.Nil(t, rsa)
	assert.Contains(t, err.Error(), "copy error")
}

func TestNewReadSeekerAtWithSizeBytesReader(t *testing.T) {
	br := bytes.NewReader([]byte("hello"))
	rsa, size, err := NewReadSeekerAtWithSize(br)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)
	assert.Same(t, br, rsa)
}

func TestNewReadSeekerAtWithSizeNonBytesReader(t *testing.T) {
	r := strings.NewReader("hello world")
	rsa, size, err := NewReadSeekerAtWithSize(r)
	require.NoError(t, err)
	assert.Equal(t, int64(11), size)
	assert.NotNil(t, rsa)
}

func TestGetSeekerSize(t *testing.T) {
	br := bytes.NewReader([]byte("hello"))
	size, err := getSeekerSize(br)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)
}

func TestGetSeekerSizeSeekError(t *testing.T) {
	_, err := getSeekerSize(&errSeeker{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "seek error")
}

func TestNopCloser(t *testing.T) {
	br := bytes.NewReader([]byte("hello"))
	nc := NopCloser(br)
	assert.NoError(t, nc.Close())
	data := make([]byte, 5)
	n, err := nc.Read(data)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
}

type errReader struct{}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

type errSeeker struct{}

func (s *errSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("seek error")
}
