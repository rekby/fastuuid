package fastuuid_test

import (
	"fmt"
	"github.com/rekby/fastuuid"
)

func ExampleMustUUIDv4() {
	var uuid [16]byte = fastuuid.MustUUIDv4()
	fmt.Println(uuid)
}

func ExampleMustUUIDv4String() {
	fmt.Println(fastuuid.MustUUIDv4String())
}

func ExampleMustUUIDv4StringBytes() {
	buf := make([]byte, 36)
	fastuuid.MustUUIDv4StringBytes(buf)
	fmt.Println(string(buf))
}
