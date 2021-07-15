# mapper

This package makes it easier to "pass" Go values into C code via cgo, and back
out again, without violating the [cgo pointer passing
rules](https://golang.org/cmd/cgo/#hdr-Passing_pointers).

Our `Mapper` does this by associating an opaque `Key` with the Go object, and
having the caller pass the key in place of the Go object.  Later, in a Go
callback for example, the Go object can be obtained using the `Get` method on
the `Mapper` in exchange for the key.

You can create a `Mapper` object for each different category of mapping, or
reuse the same one for all, with the caveats of mapping limits described below.
A global mapper, `G` is provided for your convenience.

Internally, the mapper uses a RWLock-protected Go map to associate `Keys` with
Go values.  The following patterns are supported:

  1. You have a pointer already obtained from cgo, which is at least 2-bytes
     aligned.  (Any pointer returned from malloc satisfies this property.)

     That pointer can be mapped to a Go value using the `MapPtrPair` method.
     The returned `Key` can then be used to obtain the mapped Go value using
     the `Get` method.
  
  2. You need to create a new mapping for a Go object, without having a pointer
     previously obtained from cgo.  This might be the case with a C API that
     accepts an opaque "user defined" pointer (or "refCon" in Apple's APIs),
     but doesn't return its object until after the call.  Such a user pointer
     might be passed to a callback that makes it way back to Go, where you then
     exchange the pointer for the mapped Go object.

     In this case, you can map a Go object to a unique `Key` that is
     returned from the `MapValue` method.  To keep the implementation simple,
     the number of unique keys is limited to `sizeof(uintptr)/2`.  When the
     limit is reached, the `MapValue` call panics.  On a 64-bit system, it is
     unlikely that any long-running program will reach that limit.

     Under this pattern, you can "stretch" the map limit further on a 32-bit
     system by using multiple `Mapper`s, each for different categories of
     object mappings, instead of the global map `G`.

## Example

See [example/main.go](example/main.go).

## License

MIT, see the `LICENSE.md` file.
