package funk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompact(t *testing.T) {
	var emptyFunc func() bool
	emptyFuncPtr := &emptyFunc

	nonEmptyFunc := func() bool { return true }
	nonEmptyFuncPtr := &nonEmptyFunc

	nonEmptyMap := map[int]int{1: 2}
	nonEmptyMapPtr := &nonEmptyMap

	var emptyMap map[int]int
	emptyMapPtr := &emptyMap

	var emptyChan chan bool
	nonEmptyChan := make(chan bool, 1)
	nonEmptyChan <- true

	emptyChanPtr := &emptyChan
	nonEmptyChanPtr := &nonEmptyChan

	var emptyString string
	emptyStringPtr := &emptyString

	nonEmptyString := "42"
	nonEmptyStringPtr := &nonEmptyString

	testCases := []struct {
		Arr    interface{}
		Result interface{}
	}{
		// Check with nils
		{
			[]interface{}{42, nil, (*int)(nil)},
			[]interface{}{42},
		},

		// Check with functions
		{
			[]interface{}{42, emptyFuncPtr, emptyFunc, nonEmptyFuncPtr},
			[]interface{}{42, nonEmptyFuncPtr},
		},

		// Check with slices, maps, arrays and channels
		{
			[]interface{}{
				42, [2]int{}, map[int]int{}, []string{}, nonEmptyMapPtr, emptyMap,
				emptyMapPtr, nonEmptyMap, nonEmptyChan, emptyChan, emptyChanPtr, nonEmptyChanPtr,
			},
			[]interface{}{42, nonEmptyMapPtr, nonEmptyMap, nonEmptyChan, nonEmptyChanPtr},
		},

		// Check with strings, numbers and booleans
		{
			[]interface{}{true, 0, float64(0), "", "42", emptyStringPtr, nonEmptyStringPtr, false},
			[]interface{}{true, "42", nonEmptyStringPtr},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)
			result := Compact(tc.Arr)

			if !is.Equal(result, tc.Result) {
				t.Errorf("%#v doesn't equal to %#v", result, tc.Result)
			}
		})
	}
}
