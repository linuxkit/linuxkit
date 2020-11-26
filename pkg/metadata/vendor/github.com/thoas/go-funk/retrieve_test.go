package funk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSlice(t *testing.T) {
	is := assert.New(t)

	is.Equal(Get(SliceOf(foo), "ID"), []int{1})
	is.Equal(Get(SliceOf(foo), "Bar.Name"), []string{"Test"})
	is.Equal(Get(SliceOf(foo), "Bar"), []*Bar{bar})
}

func TestGetSliceMultiLevel(t *testing.T) {
	is := assert.New(t)

	is.Equal(Get(foo, "Bar.Bars.Bar.Name"), []string{"Level2-1", "Level2-2"})
	is.Equal(Get(SliceOf(foo), "Bar.Bars.Bar.Name"), []string{"Level2-1", "Level2-2"})
}

func TestGetNull(t *testing.T) {
	is := assert.New(t)

	is.Equal(Get(foo, "EmptyValue.Int64"), int64(10))
	is.Equal(Get(SliceOf(foo), "EmptyValue.Int64"), []int64{10})
}

func TestGetNil(t *testing.T) {
	is := assert.New(t)

	is.Equal(Get(foo2, "Bar.Name"), nil)
	is.Equal(Get([]*Foo{foo, foo2}, "Bar.Name"), []string{"Test"})
}

func TestGetSimple(t *testing.T) {
	is := assert.New(t)

	is.Equal(Get(foo, "ID"), 1)

	is.Equal(Get(foo, "Bar.Name"), "Test")

	result := Get(foo, "Bar.Bars.Name")

	is.Equal(result, []string{"Level1-1", "Level1-2"})
}

func TestGetOrElse(t *testing.T) {
	is := assert.New(t)

	str := "hello world"
	is.Equal("hello world", GetOrElse(&str, "foobar"))
	is.Equal("hello world", GetOrElse(str, "foobar"))
	is.Equal("foobar", GetOrElse(nil, "foobar"))
}
