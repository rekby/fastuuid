module github.com/rekby/fastuuid/benchmarks

go 1.20

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/google/uuid v1.3.0
	github.com/jakehl/goid v1.1.0
	github.com/rekby/fastuuid v0.0.0-00010101000000-000000000000
	github.com/rogpeppe/fastuuid v1.2.0
	github.com/satori/go.uuid v1.2.0
	gitlab.com/rwxrob/uniq v0.0.0-20200325203910-f771e6779384
)

require (
	github.com/valyala/fastrand v1.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/rekby/fastuuid => ../
