package funk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	testCases := []struct {
		In    interface{}
		Panic string
	}{
		// Check with array types
		{In: []int{0, 1, 2}},
		{In: []string{"aaa", "bbb", "ccc"}},
		{In: []interface{}{0, false, "___"}},

		// Check with map types
		{In: map[int]string{0: "aaa", 1: "bbb", 2: "ccc"}},
		{In: map[string]string{"0": "aaa", "1": "bbb", "2": "ccc"}},
		{In: map[int]interface{}{0: 0, 1: false, 2: "___"}},

		// Check with invalid types
		{false, "Type bool is not supported by Chain"},
		{0, "Type int is not supported by Chain"},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			if tc.Panic != "" {
				is.PanicsWithValue(tc.Panic, func() {
					Chain(tc.In)
				})
				return
			}

			chain := Chain(tc.In)
			collection := chain.(*chainBuilder).collection

			is.Equal(collection, tc.In)
		})
	}
}

func TestLazyChain(t *testing.T) {
	testCases := []struct {
		In    interface{}
		Panic string
	}{
		// Check with array types
		{In: []int{0, 1, 2}},
		{In: []string{"aaa", "bbb", "ccc"}},
		{In: []interface{}{0, false, "___"}},

		// Check with map types
		{In: map[int]string{0: "aaa", 1: "bbb", 2: "ccc"}},
		{In: map[string]string{"0": "aaa", "1": "bbb", "2": "ccc"}},
		{In: map[int]interface{}{0: 0, 1: false, 2: "___"}},

		// Check with invalid types
		{false, "Type bool is not supported by LazyChain"},
		{0, "Type int is not supported by LazyChain"},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			if tc.Panic != "" {
				is.PanicsWithValue(tc.Panic, func() {
					LazyChain(tc.In)
				})
				return
			}

			chain := LazyChain(tc.In)
			collection := chain.(*lazyBuilder).exec()

			is.Equal(collection, tc.In)
		})
	}
}

func TestLazyChainWith(t *testing.T) {
	testCases := []struct {
		In    func() interface{}
		Panic string
	}{
		// Check with array types
		{In: func() interface{} { return []int{0, 1, 2} }},
		{In: func() interface{} { return []string{"aaa", "bbb", "ccc"} }},
		{In: func() interface{} { return []interface{}{0, false, "___"} }},

		// Check with map types
		{In: func() interface{} { return map[int]string{0: "aaa", 1: "bbb", 2: "ccc"} }},
		{In: func() interface{} { return map[string]string{"0": "aaa", "1": "bbb", "2": "ccc"} }},
		{In: func() interface{} { return map[int]interface{}{0: 0, 1: false, 2: "___"} }},

		// Check with invalid types
		{
			In:    func() interface{} { return false },
			Panic: "Type bool is not supported by LazyChainWith generator",
		},
		{
			In:    func() interface{} { return 0 },
			Panic: "Type int is not supported by LazyChainWith generator",
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test case #%d", idx+1), func(t *testing.T) {
			is := assert.New(t)

			if tc.Panic != "" {
				is.PanicsWithValue(tc.Panic, func() {
					LazyChainWith(tc.In).(*lazyBuilder).exec()
				})
				return
			}

			chain := LazyChainWith(tc.In)
			collection := chain.(*lazyBuilder).exec()

			is.Equal(collection, tc.In())
		})
	}
}

func ExampleChain() {
	v := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	chain := Chain(v)
	lazy := LazyChain(v)

	// Without builder
	a := Filter(v, func(x int) bool { return x%2 == 0 })
	b := Map(a, func(x int) int { return x * 2 })
	c := Reverse(a)
	fmt.Printf("funk.Contains(b, 2): %v\n", Contains(b, 2)) // false
	fmt.Printf("funk.Contains(b, 4): %v\n", Contains(b, 4)) // true
	fmt.Printf("funk.Sum(b): %v\n", Sum(b))                 // 40
	fmt.Printf("funk.Head(b): %v\n", Head(b))               // 4
	fmt.Printf("funk.Head(c): %v\n\n", Head(c))             // 8

	// With simple chain builder
	ca := chain.Filter(func(x int) bool { return x%2 == 0 })
	cb := ca.Map(func(x int) int { return x * 2 })
	cc := ca.Reverse()
	fmt.Printf("chainB.Contains(2): %v\n", cb.Contains(2)) // false
	fmt.Printf("chainB.Contains(4): %v\n", cb.Contains(4)) // true
	fmt.Printf("chainB.Sum(): %v\n", cb.Sum())             // 40
	fmt.Printf("chainB.Head(): %v\n", cb.Head())           // 4
	fmt.Printf("chainC.Head(): %v\n\n", cc.Head())         // 8

	// With lazy chain builder
	la := lazy.Filter(func(x int) bool { return x%2 == 0 })
	lb := la.Map(func(x int) int { return x * 2 })
	lc := la.Reverse()
	fmt.Printf("lazyChainB.Contains(2): %v\n", lb.Contains(2)) // false
	fmt.Printf("lazyChainB.Contains(4): %v\n", lb.Contains(4)) // true
	fmt.Printf("lazyChainB.Sum(): %v\n", lb.Sum())             // 40
	fmt.Printf("lazyChainB.Head(): %v\n", lb.Head())           // 4
	fmt.Printf("lazyChainC.Head(): %v\n", lc.Head())           // 8
}

type updatingStruct struct {
	x []int
}

func (us *updatingStruct) Values() interface{} {
	return us.x
}

func ExampleLazyChain() {
	us := updatingStruct{}
	chain := Chain(us.x).
		Map(func(x int) float64 { return float64(x) * 2.5 })
	lazy := LazyChain(us.x).
		Map(func(x int) float64 { return float64(x) * 2.5 })
	lazyWith := LazyChainWith(us.Values).
		Map(func(x int) float64 { return float64(x) * 2.5 })

	fmt.Printf("chain.Sum(): %v\n", chain.Sum())         // 0
	fmt.Printf("lazy.Sum(): %v\n", lazy.Sum())           // 0
	fmt.Printf("lazyWith.Sum(): %v\n\n", lazyWith.Sum()) // 0

	us.x = append(us.x, 2)
	fmt.Printf("chain.Sum(): %v\n", chain.Sum())         // 0
	fmt.Printf("lazy.Sum(): %v\n", lazy.Sum())           // 0
	fmt.Printf("lazyWith.Sum(): %v\n\n", lazyWith.Sum()) // 5

	us.x = append(us.x, 10)
	fmt.Printf("chain.Sum(): %v\n", chain.Sum())         // 0
	fmt.Printf("lazy.Sum(): %v\n", lazy.Sum())           // 0
	fmt.Printf("lazyWith.Sum(): %v\n\n", lazyWith.Sum()) // 30
}
