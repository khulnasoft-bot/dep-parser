package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	s := New("a", "b", "c")
	assert.Equal(t, 3, s.Size())
	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
	assert.True(t, s.Contains("c"))
	assert.False(t, s.Contains("d"))
}

func TestNewEmpty(t *testing.T) {
	s := New[string]()
	assert.Equal(t, 0, s.Size())
	assert.Empty(t, s.Items())
}

func TestClone(t *testing.T) {
	s := New("a", "b")
	clone := s.Clone()
	assert.Equal(t, s.Size(), clone.Size())
	assert.True(t, clone.Contains("a"))
	assert.True(t, clone.Contains("b"))
	clone.Append("c")
	assert.False(t, s.Contains("c"))
	assert.True(t, clone.Contains("c"))
}

func TestAppend(t *testing.T) {
	s := New("a")
	s.Append("b", "c")
	assert.Equal(t, 3, s.Size())
	assert.True(t, s.Contains("b"))
	assert.True(t, s.Contains("c"))
}

func TestItems(t *testing.T) {
	s := New("a", "b", "c")
	items := s.Items()
	assert.ElementsMatch(t, []string{"a", "b", "c"}, items)
}

func TestContains(t *testing.T) {
	s := New("a", "b")
	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
	assert.False(t, s.Contains("c"))
	assert.False(t, New[string]().Contains("a"))
}

func TestSize(t *testing.T) {
	assert.Equal(t, 0, New[string]().Size())
	assert.Equal(t, 1, New("a").Size())
	assert.Equal(t, 2, New("a", "b").Size())
}

func TestIter(t *testing.T) {
	s := New("a", "b")
	assert.ElementsMatch(t, []string{"a", "b"}, s.Iter())
}
