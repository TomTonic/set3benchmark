module github.com/TomTonic/set3benchmark

go 1.25.4

// replace github.com/TomTonic/Set3 => ../Set3

require (
	github.com/TomTonic/Set3 v0.4.2
	github.com/TomTonic/rtcompare v0.2.0
	github.com/alecthomas/kong v1.13.0
)

require (
	github.com/dolthub/maphash v0.1.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)
