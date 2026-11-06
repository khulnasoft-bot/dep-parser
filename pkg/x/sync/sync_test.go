package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapStoreLoad(t *testing.T) {
	m := &Map[string, int]{}
	m.Store("a", 1)
	m.Store("b", 2)

	v, ok := m.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	_, ok = m.Load("c")
	assert.False(t, ok)
}

func TestMapLoadAndDelete(t *testing.T) {
	m := &Map[string, int]{}
	m.Store("a", 1)

	v, loaded := m.LoadAndDelete("a")
	assert.True(t, loaded)
	assert.Equal(t, 1, v)

	_, loaded = m.LoadAndDelete("a")
	assert.False(t, loaded)
}

func TestMapLoadOrStore(t *testing.T) {
	m := &Map[string, int]{}

	actual, loaded := m.LoadOrStore("a", 1)
	assert.False(t, loaded)
	assert.Equal(t, 1, actual)

	actual, loaded = m.LoadOrStore("a", 2)
	assert.True(t, loaded)
	assert.Equal(t, 1, actual)
}

func TestMapDelete(t *testing.T) {
	m := &Map[string, int]{}
	m.Store("a", 1)
	m.Delete("a")

	_, ok := m.Load("a")
	assert.False(t, ok)
}

func TestMapRange(t *testing.T) {
	m := &Map[string, int]{}
	m.Store("a", 1)
	m.Store("b", 2)

	count := 0
	m.Range(func(k string, v int) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count)
}

func TestMapLen(t *testing.T) {
	m := &Map[string, int]{}
	assert.Equal(t, 0, m.Len())
	m.Store("a", 1)
	m.Store("b", 2)
	assert.Equal(t, 2, m.Len())
	m.Delete("a")
	assert.Equal(t, 1, m.Len())
}
