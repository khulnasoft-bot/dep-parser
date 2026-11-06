package slices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(i int) string {
		return string(rune('0' + i))
	})
	assert.Equal(t, []string{"1", "2", "3"}, got)
}

func TestMapEmpty(t *testing.T) {
	assert.Nil(t, Map([]int{}, func(i int) string { return "" }))
}

func TestZeroToNil(t *testing.T) {
	assert.Nil(t, ZeroToNil[int]([]int{}))
	assert.Nil(t, ZeroToNil[int](nil))

	got := ZeroToNil([]int{1, 2})
	assert.Equal(t, []int{1, 2}, got)
}
