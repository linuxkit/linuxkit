package funk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForEach(t *testing.T) {
	is := assert.New(t)

	results := []int{}

	ForEach([]int{1, 2, 3, 4}, func(x int) {
		if x%2 == 0 {
			results = append(results, x)
		}
	})

	is.Equal(results, []int{2, 4})

	mapping := map[int]string{
		1: "Florent",
		2: "Gilles",
	}

	ForEach(mapping, func(k int, v string) {
		is.Equal(v, mapping[k])
	})
}

func TestForEachRight(t *testing.T) {
	is := assert.New(t)

	results := []int{}

	ForEachRight([]int{1, 2, 3, 4}, func(x int) {
		results = append(results, x*2)
	})

	is.Equal(results, []int{8, 6, 4, 2})

	mapping := map[int]string{
		1: "Florent",
		2: "Gilles",
	}

	mapKeys := []int{}

	ForEachRight(mapping, func(k int, v string) {
		is.Equal(v, mapping[k])
		mapKeys = append(mapKeys, k)
	})

	is.Equal(len(mapKeys), 2)
	is.Contains(mapKeys, 1)
	is.Contains(mapKeys, 2)
}

func TestHead(t *testing.T) {
	is := assert.New(t)

	is.Equal(Head([]int{1, 2, 3, 4}), 1)
}

func TestLast(t *testing.T) {
	is := assert.New(t)

	is.Equal(Last([]int{1, 2, 3, 4}), 4)
}

func TestTail(t *testing.T) {
	is := assert.New(t)

	is.Equal(Tail([]int{}), []int{})
	is.Equal(Tail([]int{1}), []int{1})
	is.Equal(Tail([]int{1, 2, 3, 4}), []int{2, 3, 4})
}

func TestInitial(t *testing.T) {
	is := assert.New(t)

	is.Equal(Initial([]int{}), []int{})
	is.Equal(Initial([]int{1}), []int{1})
	is.Equal(Initial([]int{1, 2, 3, 4}), []int{1, 2, 3})
}
