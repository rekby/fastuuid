Fast uuid library, now implemented only UUIDv4 (random) generators.

Command for install:

```bash
go get github.com/rekby/fastuuid
```

Example:

```golang
package main

import "github.com/rekby/fastuuid"

func main(){
	fmt.Println(fastuuid.MustUUIDv4String())
}

```