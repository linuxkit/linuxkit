package funk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFillMismatchedTypes(t *testing.T) {
	_, err := Fill([]string{"a", "b"}, 1)
	assert.EqualError(t, err, "Cannot fill '[]string' with 'int'")
}

func TestFillUnfillableTypes(t *testing.T) {
	var stringVariable string
	var uint32Variable uint32
	var boolVariable bool

	types := [](interface{}){
		stringVariable,
		uint32Variable,
		boolVariable,
	}

	for _, unfillable := range types {
		_, err := Fill(unfillable, 1)
		assert.EqualError(t, err, "Can only fill slices and arrays")
	}
}

func TestFillSlice(t *testing.T) {
	input := []int{1, 2, 3}
	result, err := Fill(input, 1)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 1, 1}, result)

	// Assert that input does not change
	assert.Equal(t, []int{1, 2, 3}, input)
}

func TestFillArray(t *testing.T) {
	input := [...]int{1, 2, 3}
	result, err := Fill(input, 2)
	assert.NoError(t, err)
	assert.Equal(t, []int{2, 2, 2}, result)

	// Assert that input does not change
	assert.Equal(t, [...]int{1, 2, 3}, input)
}
