// Copyright 2020 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mapper makes it easy to "pass" Go values by way of an opaque
// pointer into C code via cgo, without violating the cgo pointer passing
// rules, outlined at https://golang.org/cmd/cgo/#hdr-Passing_pointers.  When
// that external code calls back into Go via a callback, you can use the passed
// opaque pointer to retrieve the Go value again.
//
// Our `Mapper` does this by associating an opaque `Key` with the Go object,
// and having the caller pass the key's handle to the C code.  Later, in a Go
// callback, the Go object can be obtained using the `Get` method on the
// `Mapper` in exchange for the opaque pointer (key).
//
// You can create a `Mapper` object for each different category of mapping, or
// reuse the same one for all, with the caveats of mapping limits described
// below.  A global mapper, `G` is provided for your convenience.
//
// Internally, the mapper uses a RWLock-protected Go map to associate `Keys`
// with Go values.  The following patterns are supported.
//
// Mapping a Go Object with an Existing Cgo Pointer
//
// You have a pointer already obtained from cgo, which is at least 2-bytes
// aligned.  (Any pointer returned from malloc satisfies this property.)
//
// That pointer can be mapped to a Go value using the `MapPtrPair` method.  The
// returned `Key` can then be used to obtain the mapped Go value using the
// `Get` method.
//
// Mapping a Go Object without an Existing Cgo Pointer
//
// You need to create a new mapping for a Go object, without having a pointer
// previously obtained from cgo.  This might be the case with a C API that
// accepts an opaque "user defined" pointer (or "refCon" in Apple's APIs), but
// doesn't return its object until after the call.  Such a user pointer might
// be passed to a callback that makes it way back to Go, where you then
// exchange the pointer for the mapped Go object.
//
// In this case, you can map a Go object to a unique `Key` that is returned
// from the `MapValue` method.  To keep the implementation simple, the number
// of unique keys is limited to `sizeof(uintptr)/2`.  When the limit is
// reached, the `MapValue` call panics.  On a 64-bit system, it is unlikely
// that any long-running program will reach that limit.
//
// Under this pattern, you can "stretch" the map limit further on a 32-bit
// system by using multiple `Mapper`s, each for different categories of object
// mappings, instead of the global map `G`.
//
// Relation to Go 1.17 and Up
//
// Go 1.17 introduced a new Handle type that is similar to the functionality
// provided here; see https://pkg.go.dev/runtime/cgo@master#Handle.  The main
// difference between this package and the new runtime/cgo Handle is that this
// package allows you to use an existing C pointer for the mapping, which turns
// out to be available quite often.
//
package mapper // go.jpap.org/mapper

// To install: `go install go.jpap.org/godoc-readme-gen`
//
//go:generate godoc-readme-gen -f -title "Map Between Go Values and Cgo Pointers"
