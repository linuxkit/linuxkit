package funk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMaxWithArrayNumericInput(t *testing.T) {
	//Test Data
	d1 := []int{8, 3, 4, 44, 0}
	n1 := []int{}
	d2 := []int8{3, 3, 5, 9, 1}
	n2 := []int8{}
	d3 := []int16{4, 5, 4, 33, 2}
	n3 := []int16{}
	d4 := []int32{5, 3, 21, 15, 3}
	n4 := []int32{}
	d5 := []int64{9, 3, 9, 1, 2}
	n5 := []int64{}
	//Calls
	r1 := MaxInt(d1)
	c1 := MaxInt(n1)
	r2 := MaxInt8(d2)
	c2 := MaxInt8(n2)
	r3 := MaxInt16(d3)
	c3 := MaxInt16(n3)
	r4 := MaxInt32(d4)
	c4 := MaxInt32(n4)
	r5 := MaxInt64(d5)
	c5 := MaxInt64(n5)
	// Assertions
	assert.Equal(t, int(44), r1, "It should return the max value in array")
	assert.Equal(t, nil, c1, "It should return nil")
	assert.Equal(t, int8(9), r2, "It should return the max value in array")
	assert.Equal(t, nil, c2, "It should return nil")
	assert.Equal(t, int16(33), r3, "It should return the max value in array")
	assert.Equal(t, nil, c3, "It should return nil")
	assert.Equal(t, int32(21), r4, "It should return the max value in array")
	assert.Equal(t, nil, c4, "It should return nil")
	assert.Equal(t, int64(9), r5, "It should return the max value in array")
	assert.Equal(t, nil, c5, "It should return nil")

}

func TestMaxWithArrayFloatInput(t *testing.T) {
	//Test Data
	d1 := []float64{2, 38.3, 4, 4.4, 4}
	n1 := []float64{}
	d2 := []float32{2.9, 1.3, 4.23, 4.4, 7.7}
	n2 := []float32{}
	//Calls
	r1 := MaxFloat64(d1)
	c1 := MaxFloat64(n1)
	r2 := MaxFloat32(d2)
	c2 := MaxFloat32(n2)
	// Assertions
	assert.Equal(t, float64(38.3), r1, "It should return the max value in array")
	assert.Equal(t, nil, c1, "It should return nil")
	assert.Equal(t, float32(7.7), r2, "It should return the max value in array")
	assert.Equal(t, nil, c2, "It should return nil")
}

func TestMaxWithArrayInputWithStrings(t *testing.T) {
	//Test Data
	d1 := []string{"abc", "abd", "cbd"}
	d2 := []string{"abc", "abd", "abe"}
	d3 := []string{"abc", "foo", " "}
	d4 := []string{"abc", "abc", "aaa"}
	n1 := []string{}
	//Calls
	r1 := MaxString(d1)
	r2 := MaxString(d2)
	r3 := MaxString(d3)
	r4 := MaxString(d4)
	c1 := MaxString(n1)
	// Assertions
	assert.Equal(t, "cbd", r1, "It should print cbd because its first char is max in the list")
	assert.Equal(t, "abe", r2, "It should print abe because its first different char is max in the list")
	assert.Equal(t, "foo", r3, "It should print foo because its first different char is max in the list")
	assert.Equal(t, "abc", r4, "It should print abc because its first different char is max in the list")
	assert.Equal(t, nil, c1, "It should return nil")
}

