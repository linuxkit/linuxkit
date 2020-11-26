package funk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin_InnerJoin(t *testing.T) {
	testCases := []struct {
		LeftArr  interface{}
		RightArr interface{}
		Expect   interface{}
	}{
		{[]string{"foo", "bar"}, []string{"bar", "baz"}, []string{"bar"}},
		{[]string{"foo", "bar", "bar"}, []string{"bar", "baz"}, []string{"bar"}},
		{[]string{"foo", "bar"}, []string{"bar", "bar", "baz"}, []string{"bar"}},
		{[]string{"foo", "bar", "bar"}, []string{"bar", "bar", "baz"}, []string{"bar"}},
		{[]int{0, 1, 2, 3, 4}, []int{3, 4, 5, 6, 7}, []int{3, 4}},
		{[]*Foo{f, b}, []*Foo{b, c}, []*Foo{b}},
	}

	for idx, tt := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := Join(tt.LeftArr, tt.RightArr, InnerJoin)
			is.Equal(tt.Expect, actual)
		})
	}
}

func TestJoin_OuterJoin(t *testing.T) {
	testCases := []struct {
		LeftArr  interface{}
		RightArr interface{}
		Expect   interface{}
	}{
		{[]string{"foo", "bar"}, []string{"bar", "baz"}, []string{"foo", "baz"}},
		{[]int{0, 1, 2, 3, 4}, []int{3, 4, 5, 6, 7}, []int{0, 1, 2, 5, 6, 7}},
		{[]*Foo{f, b}, []*Foo{b, c}, []*Foo{f, c}},
	}

	for idx, tt := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := Join(tt.LeftArr, tt.RightArr, OuterJoin)
			is.Equal(tt.Expect, actual)
		})
	}
}

func TestJoin_LeftJoin(t *testing.T) {
	testCases := []struct {
		LeftArr  interface{}
		RightArr interface{}
		Expect   interface{}
	}{
		{[]string{"foo", "bar"}, []string{"bar", "baz"}, []string{"foo"}},
		{[]int{0, 1, 2, 3, 4}, []int{3, 4, 5, 6, 7}, []int{0, 1, 2}},
		{[]*Foo{f, b}, []*Foo{b, c}, []*Foo{f}},
	}

	for idx, tt := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := Join(tt.LeftArr, tt.RightArr, LeftJoin)
			is.Equal(tt.Expect, actual)
		})
	}
}

func TestJoin_RightJoin(t *testing.T) {
	testCases := []struct {
		LeftArr  interface{}
		RightArr interface{}
		Expect   interface{}
	}{
		{[]string{"foo", "bar"}, []string{"bar", "baz"}, []string{"baz"}},
		{[]int{0, 1, 2, 3, 4}, []int{3, 4, 5, 6, 7}, []int{5, 6, 7}},
		{[]*Foo{f, b}, []*Foo{b, c}, []*Foo{c}},
	}

	for idx, tt := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := Join(tt.LeftArr, tt.RightArr, RightJoin)
			is.Equal(tt.Expect, actual)
		})
	}
}
