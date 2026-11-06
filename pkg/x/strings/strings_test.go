package strings

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type myStringer struct {
	v string
}

func (m myStringer) String() string { return m.v }

func TestToStringSlice(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, ToStringSlice[string]([]string{"a", "b"}))
	assert.Equal(t, []string{"a", "b"}, ToStringSlice([]myStringer{{v: "a"}, {v: "b"}}))
	assert.Nil(t, ToStringSlice[string]([]string{}))
	assert.Nil(t, ToStringSlice[string](nil))

	// fallback to fmt.Sprint for non-stringer types
	assert.Equal(t, []string{"42"}, ToStringSlice([]int{42}))
}

func TestToTSlice(t *testing.T) {
	got := ToTSlice[MyString]([]string{"a", "b"})
	assert.Equal(t, []MyString{"a", "b"}, got)
	assert.Nil(t, ToTSlice[MyString](nil))
	assert.Nil(t, ToTSlice[MyString]([]string{}))
}

type MyString string

func (m MyString) String() string { return string(m) }

var _ = strings.NewReader
