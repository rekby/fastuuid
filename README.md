[![PkgGoDev](https://pkg.go.dev/badge/github.com/rekby/fastuuid)](https://pkg.go.dev/github.com/rekby/fastuuid)
[![Go Report Card](https://goreportcard.com/badge/github.com/rekby/fastuuid)](https://goreportcard.com/report/github.com/rekby/fastuuid)
[![codecov](https://codecov.io/gh/github.com/rekby/fastuuid/branch/master/graph/badge.svg?precision=2)](https://github.com/rekby/fastuuid)

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