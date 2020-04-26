package dbus

import (
	"testing"
)

type TestStruct struct {
	TestInt int
	TestStr string
}

func Test_VariantOfStruct(t *testing.T) {
	tester := TestStruct{TestInt: 123, TestStr: "foobar"}
	testerDecoded := []interface{}{123, "foobar"}
	variant := MakeVariant(testerDecoded)
	input := []interface{}{variant}
	var output TestStruct
	if err := Store(input, &output); err != nil {
		t.Fatal(err)
	}
	if tester != output {
		t.Fatalf("%v != %v\n", tester, output)
	}
}
