Fast uuid library, now implemented only UUIDv4 (random) generators.

Example:

```golang
package main

import "github.com/rekby/fastuuid"

func main(){
	fmt.Println(fastuuid.MustUUIDv4String())
}

```