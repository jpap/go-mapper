// Copyright 2020 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mapper

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Mapper maps between Key and Go values.
type Mapper struct {
	mux sync.RWMutex
	m   map[Key]interface{}

	// atomicKey is a sizeof(pointer)/2 value (lower bit is reserved) that is
	// incremented for each new Key "allocation".
	atomicKey uintptr
}

// Key is an opaque token used to map onto Go values.
type Key struct {
	v uintptr
}

// We use the LSB on a cgo pointer to mark it as a synthetic "counting-pointer"
// key type.  This means that real memory pointer values supplied by the package
// user and obtained from cgo (e.g. from malloc) must be at least two bytes
// aligned.
const countingPointerBit = 1

// Handle returns an opaque "pointer" value that be passed to a C function via
// cgo.  The returned value is pointer-sized, but should never be used as a
// pointer because it may NOT be a valid address in the process' address space.
//
// For the same reason, the return type here is NOT unsafe.Pointer because if
// it were, Go's GC will panic if the actual pointer value is not a valid
// address in the process' address space:
//
//   runtime: bad pointer in frame [func] at 0x[addr]: 0x[int value]
//
// When passing a returned handle to a cgo call, it must NOT be typecast to
// unsafe.Pointer.  Doing so can result in a panic as described above.
// Instead, the caller must pass the returned handle as a C.uintptr_t.  This
// means that some C APIs that have void* arguments need to be "wrapped" in
// order to perform the typecast from uintptr_t to void* in C -- unfortunately
// the Go compiler does not allow us to do that conversion in Go without using
// unsafe.Pointer which can panic in situations described above.
//
// The following issue on the Go repository tracks this topic:
// https://github.com/golang/go/issues/22906
func (k Key) Handle() uintptr {
	return k.v
}

// KeyFromPtr converts the given cgo pointer to a Key.
//
// Strictly speaking, ptr can be any pointer, but a pointer to a Go object can
// be moved (e.g. when the stack is resized), which can render the resulting
// Key invalid, and lead to a panic.  We do not recommend any pointers to Go
// objects being passed here.
//
// We require the key to be at least 2-bytes aligned: that is, the lower bit
// must be zero, which is a reasonable assumption for pointers obtained by cgo
// via malloc and friends.
func KeyFromPtr(ptr unsafe.Pointer) Key {
	if uintptr(ptr)&countingPointerBit != 0 {
		panic(fmt.Errorf("ptr is unaligned: 0x%x", ptr))
	}
	return Key{uintptr(ptr)}
}

// KeyFromHandle converts a handle to a Key.
func KeyFromHandle(handle uintptr) Key {
	return Key{handle}
}

// G is the global mapper... for users who don't care about lock contention.
// For those that do, we recommend a separate Mapper instance.
var G Mapper

// MapPair creates a mapping between the provided Key and Go values.
func (mapper *Mapper) MapPair(key Key, goValue interface{}) {
	mapper.doMap(key, goValue)
}

// MapPtrPair is like MapPair, but maps from the given cgo pointer, and returns
// the associated Key.  This method is a convenience wrapper around KeyFromPtr
// and MapPair.
func (mapper *Mapper) MapPtrPair(ptr unsafe.Pointer, goValue interface{}) Key {
	key := KeyFromPtr(ptr)
	mapper.MapPair(key, goValue)
	return key
}

// MapValue maps and returns a new Key for the given Go value.
//
// The key here is a sizeof(pointer)/2 atomic, that is simply incremented by two
// on each call.  On a 64-bit platform, this key-space is so large that is will
// unlikely ever run out during the lifetime of a program... but if it does, we
// panic.  To avoid running out of space on a 32-bit platform (where
// 2,147,483,648 mappings are possible), use MapPtrPair instead.
func (mapper *Mapper) MapValue(goValue interface{}) Key {
	key := Key{atomic.AddUintptr(&mapper.atomicKey, 2) | countingPointerBit}
	// Crash on wrap-around
	if key.v == 0 {
		panic("key space exhausted")
	}
	mapper.doMap(key, goValue)
	return key
}

// Get retrieves the Go value from the given key.
func (mapper *Mapper) Get(key Key) (goValue interface{}) {
	mapper.mux.RLock()
	goValue, ok := mapper.m[key]
	mapper.mux.RUnlock()
	if !ok {
		panic(fmt.Errorf("key not mapped: 0x%x", key))
	}
	return
}

// GetPtr calls Get after first converting the given cgo pointer to a Key.
func (mapper *Mapper) GetPtr(ptr unsafe.Pointer) (goValue interface{}) {
	// We don't use KeyFromPtr because the ptr may be a counting-pointer type.
	key := Key{uintptr(ptr)}
	return mapper.Get(key)
}

// GetHandle calls Get after first converting the given handle to a Key.
func (mapper *Mapper) GetHandle(handle uintptr) (goValue interface{}) {
	key := KeyFromHandle(handle)
	return mapper.Get(key)
}

// Delete an existing mapping via the given key.
func (mapper *Mapper) Delete(key Key) {
	mapper.mux.Lock()
	delete(mapper.m, key)
	mapper.mux.Unlock()
}

// DeletePtr deletes an existing mapping from the given cgo pointer.
func (mapper *Mapper) DeletePtr(ptr unsafe.Pointer) {
	key := KeyFromPtr(ptr)
	mapper.Delete(key)
}

// DeletePtr deletes an existing mapping from the given handle.
func (mapper *Mapper) DeleteHandle(handle uintptr) {
	// We don't use KeyFromPtr because the ptr may be a counting-pointer type.
	key := Key{handle}
	mapper.Delete(key)
}

// Clear all mappings.
func (mapper *Mapper) Clear() {
	mapper.mux.Lock()
	mapper.m = nil
	mapper.atomicKey = 0
	mapper.mux.Unlock()
}

func (mapper *Mapper) doMap(key Key, goValue interface{}) {
	mapper.mux.Lock()
	if mapper.m == nil {
		mapper.m = make(map[Key]interface{})
	}
	mapper.m[key] = goValue
	mapper.mux.Unlock()
}
