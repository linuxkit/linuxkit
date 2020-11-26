package funk

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSubset(t *testing.T) {
	is := assert.New(t)

	r := Subset([]int{1, 2, 4}, []int{1, 2, 3, 4, 5})
	is.True(r)

	r = Subset([]string{"foo", "bar"},[]string{"foo", "bar", "hello", "bar", "hi"})
	is.True(r)

	r = Subset([]string{"hello", "foo", "bar", "hello", "bar", "hi"}, []string{})
	is.False(r)
  
        r = Subset([]string{}, []string{"hello", "foo", "bar", "hello", "bar", "hi"})
	is.True(r)

	r = Subset([]string{}, []string{})
	is.True(r)

	r = Subset([]string{}, []string{"hello"})
	is.True(r)

	r = Subset([]string{"hello", "foo", "bar", "hello", "bar", "hi"}, []string{"foo", "bar"} )
	is.False(r)
}
