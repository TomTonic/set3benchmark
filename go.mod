module github.com/TomTonic/set3benchmark

go 1.25.5

// replace github.com/TomTonic/Set3 => ../Set3

require (
	github.com/TomTonic/Set3 v0.4.2
	github.com/TomTonic/rtcompare v0.5.0
	github.com/alecthomas/kong v1.14.0
)

require (
	github.com/dolthub/maphash v0.1.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
)
