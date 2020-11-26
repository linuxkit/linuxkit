package funk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestZipEmptyResult(t *testing.T) {
	map1 := map[string]int{"a": 1, "b": 2}
	array1 := []int{21, 22, 23}
	emptySlice := []int{}

	t.Run("NonSliceOrArray", func(t *testing.T) {
		expected := []Tuple{}
		result := Zip(map1, array1)
		assert.Equal(t, result, expected)
	})

	t.Run("ZerosSized", func(t *testing.T) {
		expected := []Tuple{}
		result := Zip(emptySlice, array1)
		assert.Equal(t, result, expected)
	})
}

func zipIntsAndAssert(t *testing.T, data1, data2 interface{}) {
	t.Run("FirstOneShorter", func(t *testing.T) {
		expected := []Tuple{
			{Element1: 11, Element2: 21},
			{Element1: 12, Element2: 22},
			{Element1: 13, Element2: 23},
		}
		result := Zip(data1, data2)
		assert.Equal(t, result, expected)
	})

	t.Run("SecondOneShorter", func(t *testing.T) {
		expected := []Tuple{
			{Element1: 21, Element2: 11},
			{Element1: 22, Element2: 12},
			{Element1: 23, Element2: 13},
		}
		result := Zip(data2, data1)
		assert.Equal(t, result, expected)
	})
}

func TestZipSlices(t *testing.T) {
	slice1 := []int{11, 12, 13}
	slice2 := []int{21, 22, 23, 24, 25}
	zipIntsAndAssert(t, slice1, slice2)
}

func TestZipArrays(t *testing.T) {
	array1 := [...]int{11, 12, 13}
	array2 := [...]int{21, 22, 23, 24, 25}
	zipIntsAndAssert(t, array1, array2)
}

func TestZipStructs(t *testing.T) {
	type struct1 struct {
		Member1 uint16
		Member2 string
	}
	type struct2 struct {
		Member3 bool
	}
	type struct3 struct {
		Member4 int
		Member5 struct2
	}

	slice1 := []struct1{
		{
			Member1: 11,
			Member2: "a",
		},
		{
			Member1: 12,
			Member2: "b",
		},
		{
			Member1: 13,
			Member2: "c",
		},
	}
	slice2 := []struct3{
		{
			Member4: 21,
			Member5: struct2{
				Member3: false,
			},
		},
		{
			Member4: 22,
			Member5: struct2{
				Member3: true,
			},
		},
	}

	expected := []Tuple{
		{
			Element1: struct1{
				Member1: 11,
				Member2: "a",
			},
			Element2: struct3{
				Member4: 21,
				Member5: struct2{
					Member3: false,
				},
			},
		},
		{
			Element1: struct1{
				Member1: 12,
				Member2: "b",
			},
			Element2: struct3{
				Member4: 22,
				Member5: struct2{
					Member3: true,
				},
			},
		},
	}

	result := Zip(slice1, slice2)
	assert.Equal(t, expected, result)
}
