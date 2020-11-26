package funk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var f = &Foo{
	ID:        1,
	FirstName: "Dark",
	LastName:  "Vador",
	Age:       30,
	Bar: &Bar{
		Name: "Test",
	},
}

var b = &Foo{
	ID:        2,
	FirstName: "Florent",
	LastName:  "Messa",
	Age:       28,
}
var c = &Foo{
	ID:        3,
	FirstName: "Harald",
	LastName:  "Nordgren",
	Age:       27,
}

var results = []*Foo{f, c}

type Person struct {
	name string
	age  int
}

func TestContains(t *testing.T) {
	is := assert.New(t)

	is.True(Contains([]string{"foo", "bar"}, "bar"))
	is.True(Contains([...]string{"foo", "bar"}, "bar"))
	is.Panics(func() { Contains(1, 2) })

	is.True(Contains(results, f))
	is.False(Contains(results, nil))
	is.False(Contains(results, b))

	is.True(Contains("florent", "rent"))
	is.False(Contains("florent", "gilles"))

	mapping := ToMap(results, "ID")

	is.True(Contains(mapping, 1))
	is.False(Contains(mapping, 2))
}

func TestEvery(t *testing.T) {
	is := assert.New(t)

	is.True(Every([]string{"foo", "bar", "baz"}, "bar", "foo"))

	is.True(Every(results, f, c))
	is.False(Every(results, nil))
	is.False(Every(results, f, b))

	is.True(Every("florent", "rent", "flo"))
	is.False(Every("florent", "rent", "gilles"))

	mapping := ToMap(results, "ID")

	is.True(Every(mapping, 1, 3))
	is.False(Every(mapping, 2, 3))
}

func TestSome(t *testing.T) {
	is := assert.New(t)

	is.True(Some([]string{"foo", "bar", "baz"}, "foo"))
	is.True(Some([]string{"foo", "bar", "baz"}, "foo", "qux"))

	is.True(Some(results, f))
	is.False(Some(results, b))
	is.False(Some(results, nil))
	is.True(Some(results, f, b))

	is.True(Some("zeeshan", "zee", "tam"))
	is.False(Some("zeeshan", "zi", "tam"))

	persons := []Person{
		Person{
			name: "Zeeshan",
			age:  23,
		},
		Person{
			name: "Bob",
			age:  26,
		},
	}

	person := Person{"Zeeshan", 23}
	person2 := Person{"Alice", 23}
	person3 := Person{"John", 26}

	is.True(Some(persons, person, person2))
	is.False(Some(persons, person2, person3))

	mapping := ToMap(results, "ID")

	is.True(Some(mapping, 1, 2))
	is.True(Some(mapping, 4, 1))
	is.False(Some(mapping, 4, 5))
}

func TestIndexOf(t *testing.T) {
	is := assert.New(t)

	is.Equal(IndexOf([]string{"foo", "bar"}, "bar"), 1)

	is.Equal(IndexOf(results, f), 0)
	is.Equal(IndexOf(results, b), -1)
}

func TestLastIndexOf(t *testing.T) {
	is := assert.New(t)

	is.Equal(LastIndexOf([]string{"foo", "bar", "bar"}, "bar"), 2)
	is.Equal(LastIndexOf([]int{1, 2, 2, 3}, 2), 2)
	is.Equal(LastIndexOf([]int{1, 2, 2, 3}, 4), -1)
}

func TestFilter(t *testing.T) {
	is := assert.New(t)

	r := Filter([]int{1, 2, 3, 4}, func(x int) bool {
		return x%2 == 0
	})

	is.Equal(r, []int{2, 4})
}

func TestFind(t *testing.T) {
	is := assert.New(t)

	r := Find([]int{1, 2, 3, 4}, func(x int) bool {
		return x%2 == 0
	})

	is.Equal(r, 2)

}
func TestFindKey(t *testing.T) {
	is := assert.New(t)

	k, r := FindKey(map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}, func(x int) bool {
		return x == 2
	})

	is.Equal(r, 2)
	is.Equal(k, "b")

	k1, r1 := FindKey([]int{1, 2, 3, 4}, func(x int) bool {
		return x%2 == 0
	})
	is.Equal(r1, 2)
	is.Equal(k1, 1)
}
