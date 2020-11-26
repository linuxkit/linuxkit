package funk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithout(t *testing.T) {
	testCases := []struct {
		Arr    interface{}
		Values []interface{}
		Expect interface{}
	}{
		{[]string{"foo", "bar"}, []interface{}{"bar"}, []string{"foo"}},
		{[]int{0, 1, 2, 3, 4}, []interface{}{3, 4}, []int{0, 1, 2}},
		{[]*Foo{f, b}, []interface{}{b, c}, []*Foo{f}},
	}

	for idx, tt := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := Without(tt.Arr, tt.Values...)
			is.Equal(tt.Expect, actual)
		})
	}
}
