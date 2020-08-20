package funk

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLazyChunk(t *testing.T) {
	testCases := []struct {
		In   interface{}
		Size int
	}{
		{In: []int{0, 1, 2, 3, 4}, Size: 2},
		{In: []int{}, Size: 2},
		{In: []int{1}, Size: 2},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Chunk(tc.In, tc.Size)
			actual := LazyChain(tc.In).Chunk(tc.Size).Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyCompact(t *testing.T) {
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
		In interface{}
	}{
		{In: []interface{}{42, nil, (*int)(nil)}},
		{In: []interface{}{42, emptyFuncPtr, emptyFunc, nonEmptyFuncPtr}},
		{In: []interface{}{42, [2]int{}, map[int]int{}, []string{}, nonEmptyMapPtr, emptyMap, emptyMapPtr, nonEmptyMap, nonEmptyChan, emptyChan, emptyChanPtr, nonEmptyChanPtr}},
		{In: []interface{}{true, 0, float64(0), "", "42", emptyStringPtr, nonEmptyStringPtr, false}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Compact(tc.In)
			actual := LazyChain(tc.In).Compact().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyDrop(t *testing.T) {
	testCases := []struct {
		In interface{}
		N  int
	}{
		{In: []int{0, 1, 1, 2, 3, 0, 0, 12}, N: 3},
		// Bug: Issues from go-funk (n parameter can be greater than len(in))
		// {In: []int{0, 1}, N: 3},
		// {In: []int{}, N: 3},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Drop(tc.In, tc.N)
			actual := LazyChain(tc.In).Drop(tc.N).Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyFilter(t *testing.T) {
	testCases := []struct {
		In        interface{}
		Predicate interface{}
	}{
		{
			In:        []int{1, 2, 3, 4},
			Predicate: func(x int) bool { return x%2 == 0 },
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Filter(tc.In, tc.Predicate)
			actual := LazyChain(tc.In).Filter(tc.Predicate).Value()

			is.Equal(expected, actual)
		})
	}
}
func TestLazyFilter_SideEffect(t *testing.T) {
	is := assert.New(t)

	type foo struct {
		bar string
	}
	in := []*foo{&foo{"foo"}, &foo{"bar"}}

	LazyChain := LazyChain(in)
	is.Equal([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())

	filtered := LazyChain.Filter(func(x *foo) bool {
		x.bar = "__" + x.bar + "__"
		return x.bar == "foo"
	})
	is.Equal([]*foo{}, filtered.Value())

	// Side effect: in and LazyChain.Value modified
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, in)
}

func TestLazyFlattenDeep(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: [][]int{{1, 2}, {3, 4}},
		},
		{
			In: [][][]int{{{1, 2}, {3, 4}}, {{5, 6}}},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := FlattenDeep(tc.In)
			actual := LazyChain(tc.In).FlattenDeep().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyInitial(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{},
		},
		{
			In: []int{0},
		},
		{
			In: []int{0, 1, 2, 3},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Initial(tc.In)
			actual := LazyChain(tc.In).Initial().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyIntersect(t *testing.T) {
	testCases := []struct {
		In  interface{}
		Sec interface{}
	}{
		{
			In:  []int{1, 2, 3, 4},
			Sec: []int{2, 4, 6},
		},
		{
			In:  []string{"foo", "bar", "hello", "bar"},
			Sec: []string{"foo", "bar"},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Intersect(tc.In, tc.Sec)
			actual := LazyChain(tc.In).Intersect(tc.Sec).Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyMap(t *testing.T) {
	testCases := []struct {
		In     interface{}
		MapFnc interface{}
	}{
		{
			In:     []int{1, 2, 3, 4},
			MapFnc: func(x int) string { return "Hello" },
		},
		{
			In:     []int{1, 2, 3, 4},
			MapFnc: func(x int) (int, int) { return x, x },
		},
		{
			In:     map[int]string{1: "Florent", 2: "Gilles"},
			MapFnc: func(k int, v string) int { return k },
		},
		{
			In:     map[int]string{1: "Florent", 2: "Gilles"},
			MapFnc: func(k int, v string) (string, string) { return fmt.Sprintf("%d", k), v },
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Map(tc.In, tc.MapFnc)
			actual := LazyChain(tc.In).Map(tc.MapFnc).Value()

			if reflect.TypeOf(expected).Kind() == reflect.Map {
				is.Equal(expected, actual)
			} else {
				is.ElementsMatch(expected, actual)
			}
		})
	}
}

func TestLazyMap_SideEffect(t *testing.T) {
	is := assert.New(t)

	type foo struct {
		bar string
	}
	in := []*foo{&foo{"foo"}, &foo{"bar"}}

	LazyChain := LazyChain(in)
	is.Equal([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())

	mapped := LazyChain.Map(func(x *foo) (string, bool) {
		x.bar = "__" + x.bar + "__"
		return x.bar, x.bar == "foo"
	})
	is.Equal(map[string]bool{"__foo__": false, "__bar__": false}, mapped.Value())

	// Side effect: in and LazyChain.Value modified
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, in)
}

func TestLazyReverse(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{0, 1, 2, 3, 4},
		},
		{
			In: "abcdefg",
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Reverse(tc.In)
			actual := LazyChain(tc.In).Reverse().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyShuffle(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{0, 1, 2, 3, 4},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Shuffle(tc.In)
			actual := LazyChain(tc.In).Shuffle().Value()

			is.NotEqual(expected, actual)
			is.ElementsMatch(expected, actual)
		})
	}
}

func TestLazyTail(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{},
		},
		{
			In: []int{0},
		},
		{
			In: []int{0, 1, 2, 3},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Tail(tc.In)
			actual := LazyChain(tc.In).Tail().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyUniq(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{0, 1, 1, 2, 3, 0, 0, 12},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Uniq(tc.In)
			actual := LazyChain(tc.In).Uniq().Value()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyAll(t *testing.T) {
	testCases := []struct {
		In []interface{}
	}{
		{In: []interface{}{"foo", "bar"}},
		{In: []interface{}{"foo", ""}},
		{In: []interface{}{"", ""}},
		{In: []interface{}{}},
		{In: []interface{}{true, "foo", 6}},
		{In: []interface{}{true, "", 6}},
		{In: []interface{}{true, "foo", 0}},
		{In: []interface{}{false, "foo", 6}},
		{In: []interface{}{false, "", 0}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := All(tc.In...)
			actual := LazyChain(tc.In).All()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyAny(t *testing.T) {
	testCases := []struct {
		In []interface{}
	}{
		{In: []interface{}{"foo", "bar"}},
		{In: []interface{}{"foo", ""}},
		{In: []interface{}{"", ""}},
		{In: []interface{}{}},
		{In: []interface{}{true, "foo", 6}},
		{In: []interface{}{true, "", 6}},
		{In: []interface{}{true, "foo", 0}},
		{In: []interface{}{false, "foo", 6}},
		{In: []interface{}{false, "", 0}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Any(tc.In...)
			actual := LazyChain(tc.In).Any()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyContains(t *testing.T) {
	testCases := []struct {
		In       interface{}
		Contains interface{}
	}{
		{
			In:       []string{"foo", "bar"},
			Contains: "bar",
		},
		{
			In:       results,
			Contains: f,
		},
		{
			In:       results,
			Contains: nil,
		},
		{
			In:       results,
			Contains: b,
		},
		{
			In:       "florent",
			Contains: "rent",
		},
		{
			In:       "florent",
			Contains: "gilles",
		},
		{
			In:       map[int]*Foo{1: f, 3: c},
			Contains: 1,
		},
		{
			In:       map[int]*Foo{1: f, 3: c},
			Contains: 2,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Contains(tc.In, tc.Contains)
			actual := LazyChain(tc.In).Contains(tc.Contains)

			is.Equal(expected, actual)
		})
	}
}

func TestLazyEvery(t *testing.T) {
	testCases := []struct {
		In       interface{}
		Contains []interface{}
	}{
		{
			In:       []string{"foo", "bar", "baz"},
			Contains: []interface{}{"bar", "foo"},
		},
		{
			In:       results,
			Contains: []interface{}{f, c},
		},
		{
			In:       results,
			Contains: []interface{}{nil},
		},
		{
			In:       results,
			Contains: []interface{}{f, b},
		},
		{
			In:       "florent",
			Contains: []interface{}{"rent", "flo"},
		},
		{
			In:       "florent",
			Contains: []interface{}{"rent", "gilles"},
		},
		{
			In:       map[int]*Foo{1: f, 3: c},
			Contains: []interface{}{1, 3},
		},
		{
			In:       map[int]*Foo{1: f, 3: c},
			Contains: []interface{}{2, 3},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Every(tc.In, tc.Contains...)
			actual := LazyChain(tc.In).Every(tc.Contains...)

			is.Equal(expected, actual)
		})
	}
}

func TestLazyFind(t *testing.T) {
	testCases := []struct {
		In        interface{}
		Predicate interface{}
	}{
		{
			In:        []int{1, 2, 3, 4},
			Predicate: func(x int) bool { return x%2 == 0 },
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Find(tc.In, tc.Predicate)
			actual := LazyChain(tc.In).Find(tc.Predicate)

			is.Equal(expected, actual)
		})
	}
}

func TestLazyFind_SideEffect(t *testing.T) {
	is := assert.New(t)

	type foo struct {
		bar string
	}
	in := []*foo{&foo{"foo"}, &foo{"bar"}}

	LazyChain := LazyChain(in)
	is.Equal([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())

	result := LazyChain.Find(func(x *foo) bool {
		x.bar = "__" + x.bar + "__"
		return x.bar == "foo"
	})
	is.Nil(result)

	// Side effect: in and LazyChain.Value modified
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, in)
}

func TestLazyForEach(t *testing.T) {
	var expectedAcc, actualAcc []interface{}

	testCases := []struct {
		In                interface{}
		FunkIterator      interface{}
		LazyChainIterator interface{}
	}{
		{
			In: []int{1, 2, 3, 4},
			FunkIterator: func(x int) {
				if x%2 == 0 {
					expectedAcc = append(expectedAcc, x)
				}
			},
			LazyChainIterator: func(x int) {
				if x%2 == 0 {
					actualAcc = append(actualAcc, x)
				}
			},
		},
		{
			In:                map[int]string{1: "Florent", 2: "Gilles"},
			FunkIterator:      func(k int, v string) { expectedAcc = append(expectedAcc, fmt.Sprintf("%d:%s", k, v)) },
			LazyChainIterator: func(k int, v string) { actualAcc = append(actualAcc, fmt.Sprintf("%d:%s", k, v)) },
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)
			expectedAcc = []interface{}{}
			actualAcc = []interface{}{}

			ForEach(tc.In, tc.FunkIterator)
			LazyChain(tc.In).ForEach(tc.LazyChainIterator)

			is.ElementsMatch(expectedAcc, actualAcc)
		})
	}
}

func TestLazyForEach_SideEffect(t *testing.T) {
	is := assert.New(t)

	type foo struct {
		bar string
	}
	var out []*foo
	in := []*foo{&foo{"foo"}, &foo{"bar"}}

	LazyChain := LazyChain(in)
	is.Equal([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())

	LazyChain.ForEach(func(x *foo) {
		x.bar = "__" + x.bar + "__"
		out = append(out, x)
	})
	is.Equal([]*foo{&foo{"__foo__"}, &foo{"__bar__"}}, out)

	// Side effect: in and LazyChain.Value modified
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, in)
}

func TestLazyForEachRight(t *testing.T) {
	var expectedAcc, actualAcc []interface{}

	testCases := []struct {
		In                interface{}
		FunkIterator      interface{}
		LazyChainIterator interface{}
	}{
		{
			In: []int{1, 2, 3, 4},
			FunkIterator: func(x int) {
				if x%2 == 0 {
					expectedAcc = append(expectedAcc, x)
				}
			},
			LazyChainIterator: func(x int) {
				if x%2 == 0 {
					actualAcc = append(actualAcc, x)
				}
			},
		},
		{
			In:                map[int]string{1: "Florent", 2: "Gilles"},
			FunkIterator:      func(k int, v string) { expectedAcc = append(expectedAcc, fmt.Sprintf("%d:%s", k, v)) },
			LazyChainIterator: func(k int, v string) { actualAcc = append(actualAcc, fmt.Sprintf("%d:%s", k, v)) },
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)
			expectedAcc = []interface{}{}
			actualAcc = []interface{}{}

			ForEachRight(tc.In, tc.FunkIterator)
			LazyChain(tc.In).ForEachRight(tc.LazyChainIterator)

			is.ElementsMatch(expectedAcc, actualAcc)
		})
	}
}

func TestLazyForEachRight_SideEffect(t *testing.T) {
	is := assert.New(t)

	type foo struct {
		bar string
	}
	var out []*foo
	in := []*foo{&foo{"foo"}, &foo{"bar"}}

	LazyChain := LazyChain(in)
	is.Equal([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())

	LazyChain.ForEachRight(func(x *foo) {
		x.bar = "__" + x.bar + "__"
		out = append(out, x)
	})
	is.Equal([]*foo{&foo{"__bar__"}, &foo{"__foo__"}}, out)

	// Side effect: in and LazyChain.Value modified
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, LazyChain.Value())
	is.NotEqual([]*foo{&foo{"foo"}, &foo{"bar"}}, in)
}

func TestLazyHead(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{1, 2, 3, 4},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Head(tc.In)
			actual := LazyChain(tc.In).Head()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyKeys(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: map[string]int{"one": 1, "two": 2}},
		{In: &map[string]int{"one": 1, "two": 2}},
		{In: map[int]complex128{5: 1 + 8i, 3: 2}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Keys(tc.In)
			actual := LazyChain(tc.In).Keys()

			is.ElementsMatch(expected, actual)
		})
	}
}

func TestLazyIndexOf(t *testing.T) {
	testCases := []struct {
		In   interface{}
		Item interface{}
	}{
		{
			In:   []string{"foo", "bar"},
			Item: "bar",
		},
		{
			In:   results,
			Item: f,
		},
		{
			In:   results,
			Item: b,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := IndexOf(tc.In, tc.Item)
			actual := LazyChain(tc.In).IndexOf(tc.Item)

			is.Equal(expected, actual)
		})
	}
}

func TestLazyIsEmpty(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: ""},
		{In: [0]interface{}{}},
		{In: []interface{}(nil)},
		{In: map[interface{}]interface{}(nil)},
		{In: "s"},
		{In: [1]interface{}{1}},
		{In: []interface{}{}},
		{In: map[interface{}]interface{}{}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := IsEmpty(tc.In)
			actual := LazyChain(tc.In).IsEmpty()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyLast(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{
			In: []int{1, 2, 3, 4},
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Last(tc.In)
			actual := LazyChain(tc.In).Last()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyLastIndexOf(t *testing.T) {
	testCases := []struct {
		In   interface{}
		Item interface{}
	}{
		{
			In:   []string{"foo", "bar", "bar"},
			Item: "bar",
		},
		{
			In:   []int{1, 2, 2, 3},
			Item: 2,
		},
		{
			In:   []int{1, 2, 2, 3},
			Item: 4,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := LastIndexOf(tc.In, tc.Item)
			actual := LazyChain(tc.In).LastIndexOf(tc.Item)

			is.Equal(expected, actual)
		})
	}
}

func TestLazyNotEmpty(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: ""},
		{In: [0]interface{}{}},
		{In: []interface{}(nil)},
		{In: map[interface{}]interface{}(nil)},
		{In: "s"},
		{In: [1]interface{}{1}},
		{In: []interface{}{}},
		{In: map[interface{}]interface{}{}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := NotEmpty(tc.In)
			actual := LazyChain(tc.In).NotEmpty()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyProduct(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: []int{0, 1, 2, 3}},
		{In: &[]int{0, 1, 2, 3}},
		{In: []interface{}{1, 2, 3, 0.5}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Product(tc.In)
			actual := LazyChain(tc.In).Product()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyReduce(t *testing.T) {
	testCases := []struct {
		In         interface{}
		ReduceFunc interface{}
		Acc        interface{}
	}{
		{
			In:         []int{1, 2, 3, 4},
			ReduceFunc: func(acc, elem int) int { return acc + elem },
			Acc:        0,
		},
		{
			In:         &[]int16{1, 2, 3, 4},
			ReduceFunc: '+',
			Acc:        5,
		},
		{
			In:         []float64{1.1, 2.2, 3.3},
			ReduceFunc: '+',
			Acc:        0,
		},
		{
			In:         &[]int{1, 2, 3, 5},
			ReduceFunc: func(acc int8, elem int16) int32 { return int32(acc) * int32(elem) },
			Acc:        1,
		},
		{
			In:         []interface{}{1, 2, 3.3, 4},
			ReduceFunc: '*',
			Acc:        1,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Reduce(tc.In, tc.ReduceFunc, tc.Acc)
			actual := LazyChain(tc.In).Reduce(tc.ReduceFunc, tc.Acc)

			is.Equal(expected, actual)
		})
	}
}

func TestLazySum(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: []int{0, 1, 2, 3}},
		{In: &[]int{0, 1, 2, 3}},
		{In: []interface{}{1, 2, 3, 0.5}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Sum(tc.In)
			actual := LazyChain(tc.In).Sum()

			is.Equal(expected, actual)
		})
	}
}

func TestLazyType(t *testing.T) {
	type key string
	var x key

	testCases := []struct {
		In interface{}
	}{
		{In: []string{}},
		{In: []int{}},
		{In: []bool{}},
		{In: []interface{}{}},
		{In: &[]interface{}{}},
		{In: map[int]string{}},
		{In: map[complex128]int{}},
		{In: map[string]string{}},
		{In: map[int]interface{}{}},
		{In: map[key]interface{}{}},
		{In: &map[key]interface{}{}},
		{In: ""},
		{In: &x},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := LazyChain(tc.In).Type()

			is.Equal(reflect.TypeOf(tc.In), actual)
		})
	}
}

func TestLazyValue(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: []int{0, 1, 2, 3}},
		{In: []string{"foo", "bar"}},
		{In: &[]string{"foo", "bar"}},
		{In: map[int]string{1: "foo", 2: "bar"}},
		{In: map[string]string{"foo": "foo", "bar": "bar"}},
		{In: &map[string]string{"foo": "foo", "bar": "bar"}},
		{In: "foo"},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			actual := LazyChain(tc.In).Value()

			is.Equal(tc.In, actual)
		})
	}
}

func TestLazyValues(t *testing.T) {
	testCases := []struct {
		In interface{}
	}{
		{In: map[string]int{"one": 1, "two": 2}},
		{In: &map[string]int{"one": 1, "two": 2}},
		{In: map[int]complex128{5: 1 + 8i, 3: 2}},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			expected := Values(tc.In)
			actual := LazyChain(tc.In).Values()

			is.ElementsMatch(expected, actual)
		})
	}
}

func TestComplexLazyChaining(t *testing.T) {
	is := assert.New(t)

	in := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	lazy := LazyChain(in)
	lazyWith := LazyChainWith(func() interface{} { return in })

	// Without builder
	fa := Filter(in, func(x int) bool { return x%2 == 0 })
	fb := Map(fa, func(x int) int { return x * 2 })
	fc := Reverse(fa)

	// With lazy chaining
	la := lazy.Filter(func(x int) bool { return x%2 == 0 })
	lb := la.Map(func(x int) int { return x * 2 })
	lc := la.Reverse()

	// With lazy chaining with generator
	lwa := lazyWith.Filter(func(x int) bool { return x%2 == 0 })
	lwb := lwa.Map(func(x int) int { return x * 2 })
	lwc := lwa.Reverse()

	is.Equal(fa, la.Value())
	is.Equal(fb, lb.Value())
	is.Equal(fc, lc.Value())
	is.Equal(fa, lwa.Value())
	is.Equal(fb, lwb.Value())
	is.Equal(fc, lwc.Value())

	is.Equal(Contains(fb, 2), lb.Contains(2))
	is.Equal(Contains(fb, 4), lb.Contains(4))
	is.Equal(Sum(fb), lb.Sum())
	is.Equal(Head(fb), lb.Head())
	is.Equal(Head(fc), lc.Head())
	is.Equal(Contains(fb, 2), lwb.Contains(2))
	is.Equal(Contains(fb, 4), lwb.Contains(4))
	is.Equal(Sum(fb), lwb.Sum())
	is.Equal(Head(fb), lwb.Head())
	is.Equal(Head(fc), lwc.Head())
}
