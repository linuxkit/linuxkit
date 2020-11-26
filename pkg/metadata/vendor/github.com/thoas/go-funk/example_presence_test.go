package funk

import "fmt"

func ExampleSome() {
	a := []string{"foo", "bar", "baz"}
	fmt.Println(Some(a, "foo", "qux"))

	b := "Mark Shaun"
	fmt.Println(Some(b, "Marc", "Sean"))

	// Output: true
	// false
}
