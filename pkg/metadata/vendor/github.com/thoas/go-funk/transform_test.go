package funk

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	is := assert.New(t)

	r := Map([]int{1, 2, 3, 4}, func(x int) string {
		return "Hello"
	})

	result, ok := r.([]string)

	is.True(ok)
	is.Equal(len(result), 4)

	r = Map([]int{1, 2, 3, 4}, func(x int) (int, int) {
		return x, x
	})

	resultType := reflect.TypeOf(r)

	is.True(resultType.Kind() == reflect.Map)
	is.True(resultType.Key().Kind() == reflect.Int)
	is.True(resultType.Elem().Kind() == reflect.Int)

	mapping := map[int]string{
		1: "Florent",
		2: "Gilles",
	}

	r = Map(mapping, func(k int, v string) int {
		return k
	})

	is.True(reflect.TypeOf(r).Kind() == reflect.Slice)
	is.True(reflect.TypeOf(r).Elem().Kind() == reflect.Int)

	r = Map(mapping, func(k int, v string) (string, string) {
		return fmt.Sprintf("%d", k), v
	})

	resultType = reflect.TypeOf(r)

	is.True(resultType.Kind() == reflect.Map)
	is.True(resultType.Key().Kind() == reflect.String)
	is.True(resultType.Elem().Kind() == reflect.String)
}

func TestToMap(t *testing.T) {
	is := assert.New(t)

	f := &Foo{
		ID:        1,
		FirstName: "Dark",
		LastName:  "Vador",
		Age:       30,
		Bar: &Bar{
			Name: "Test",
		},
	}

	results := []*Foo{f}

	instanceMap := ToMap(results, "ID")

	is.True(reflect.TypeOf(instanceMap).Kind() == reflect.Map)

	mapping, ok := instanceMap.(map[int]*Foo)

	is.True(ok)

	for _, result := range results {
		item, ok := mapping[result.ID]

		is.True(ok)
		is.True(reflect.TypeOf(item).Kind() == reflect.Ptr)
		is.True(reflect.TypeOf(item).Elem().Kind() == reflect.Struct)

		is.Equal(item.ID, result.ID)
	}
}

func TestChunk(t *testing.T) {
	is := assert.New(t)

	results := Chunk([]int{0, 1, 2, 3, 4}, 2).([][]int)

	is.Len(results, 3)
	is.Len(results[0], 2)
	is.Len(results[1], 2)
	is.Len(results[2], 1)

	is.Len(Chunk([]int{}, 2), 0)
	is.Len(Chunk([]int{1}, 2), 1)
	is.Len(Chunk([]int{1, 2, 3}, 0), 3)
}

func TestFlattenDeep(t *testing.T) {
	is := assert.New(t)

	is.Equal(FlattenDeep([][]int{{1, 2}, {3, 4}}), []int{1, 2, 3, 4})
}

func TestShuffle(t *testing.T) {
	initial := []int{0, 1, 2, 3, 4}

	results := Shuffle(initial)

	is := assert.New(t)

	is.Len(results, 5)

	for _, entry := range initial {
		is.True(Contains(results, entry))
	}
}

func TestReverse(t *testing.T) {
	results := Reverse([]int{0, 1, 2, 3, 4})

	is := assert.New(t)

	is.Equal(Reverse("abcdefg"), "gfedcba")
	is.Len(results, 5)

	is.Equal(results, []int{4, 3, 2, 1, 0})
}

func TestUniq(t *testing.T) {
	is := assert.New(t)

	results := Uniq([]int{0, 1, 1, 2, 3, 0, 0, 12})
	is.Len(results, 5)
	is.Equal(results, []int{0, 1, 2, 3, 12})

	results = Uniq([]string{"foo", "bar", "foo", "bar", "bar"})
	is.Len(results, 2)
	is.Equal(results, []string{"foo", "bar"})
}

func TestConvertSlice(t *testing.T) {
	instances := []*Foo{foo, foo2}

	var raw []Model

	ConvertSlice(instances, &raw)

	is := assert.New(t)

	is.Len(raw, len(instances))
}

func TestDrop(t *testing.T) {
	results := Drop([]int{0, 1, 1, 2, 3, 0, 0, 12}, 3)

	is := assert.New(t)

	is.Len(results, 5)

	is.Equal([]int{2, 3, 0, 0, 12}, results)
}
