# Goyp - Tool that helps you to protect your Software IP written in Golang

Goyp (pronounced go - eep) is a Golang tool that uses `go tool compile` and `go tool link` Golang commands to compile and link your application together with any library that is in Go Object code format or source code format. As well as compile your library to Go Object code to distribute to your users and protect your intellectual property.

Supporting Go Modules and libraries distributed as Go Object code, as those that are produced by `go tool compile` command

## Why?

Well Golang one time had support for creating Go Object file and distribute it to users to protect your IP, but with the Go Modules arrival, they dropped support of this feature, stating that it was very hard to maintain and allegating that "Nobody uses this feature". This disappointed many developers who do care about our intellectual property or our country does not protects our intellectual property from being violated (when distributing in source code format), we ended up using other, more considerate alternatives.

Now I want to support this removed features.

## How this is done?

The previous way of distributing Go Object file libraries was with "Binary-only packages", I'm going to take another approach and let users use `//go:linkname` compiler directive, since this is a very well supported directive with pretty well documentation.

Example:

Suppose the following two Go source code files:

```go
// closed_src/lib.go

package closed_src

import _ "unsafe"

//go:linkname algo header.Algorithm
func algo() int { ...Your closed source algorithm }
```

```go
// header/header.go

package header

import _ "closed_src" // Provides definition for "Algorithm" function

func Algorithm() int
```

From this two files, you can produce from `closed_src` package a Go Object file and distribute it together with the `header/header.go` source file, protecting the sources succesfully (This example doesn't use Go Modules).

## Some problems...

There has to be a way to determine the dependencies of a Go Object file, since Go Modules deletes dependencies from `go.mod` falsely claiming that these are unused, this tool will determine the dependencies on compile time and add it to the archive file produced by `go tool compile`, you can list the dependencies with **(TODO: command to list dependencies from Go Archive file)**.
