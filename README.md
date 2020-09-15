# go-cptr

This package makes it easier to pass opaque pointers into C code, without violating the [restrictions imposed by Cgo](https://golang.org/cmd/cgo/#hdr-Passing_pointers).  It uses a `sync.Map` to map between opaque pointers and Go values.

## Usage

```
/* ... */
import "C"
import "go.jpap.org/mapper"

// cptr is a mapper between Go values and Cgo pointers.
var cptr = mapper.NewMapper()

func Usage() {
  var s myStruct = ...
  ptr := cptr.New(s)
  defer cptr.Delete(s)

  C.invokeCfuncThatCallsGoCallback(ptr)
}

//export goCallback
func goCallback(p unsafe.Pointer) {
  s := cptr.Get(p).(myStruct)

  // ... use s
}
```

## License

MIT, see the `LICENSE.md` file.
