// Copyright 2020 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

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
}

// Key is an opaque token used to map onto Go values.
type Key struct {
	v uintptr
}

// Handle returns an opaque "pointer" value that can be used to pass to cgo.
// The returned value is pointer-sized, but should never be de-referenced.
func (k Key) Handle() unsafe.Pointer {
	return unsafe.Pointer(k.v)
}

// KeyFromPtr converts a cgo pointer to a Key.
//
// The key can be any unique pointer, but it is recommended that it be a value
// obtained from outside the view of the Go GC, in case the GC moves that memory
// around.  Typically key is a malloc-based pointer obtained from cgo.
//
// We require the key to be at least 2-bytes aligned: that is, the lower bit
// must be zero, which is a reasonable assumption for pointers returned via
// malloc.
func KeyFromPtr(ptr unsafe.Pointer) Key {
	if uintptr(ptr)&0x1 != 0 {
		panic(fmt.Errorf("mapper: ptr is unaligned: 0x%x", ptr))
	}
	return Key{uintptr(ptr)} // we assume pointer <= 64-bits!
}

// G is the global mapper... for users who don't care about lock contention.
// For those that do, it is recommended to use a separate Mapper instance.
var G Mapper

// MapPair creates a mapping between the provided Key and Go value.
func (mapper *Mapper) MapPair(key Key, goValue interface{}) {
	mapper.doMap(key, goValue)
}

// MapPtrPair is like MapPair, but maps from a cgo pointer, and returns the
// associated Key.  This method is a convenience wrapper around KeyFromPtr and
// MapPair.
func (mapper *Mapper) MapPtrPair(ptr unsafe.Pointer, goValue interface{}) Key {
	key := KeyFromPtr(ptr)
	mapper.MapPair(key, goValue)
	return key
}

// MapValue maps a new Key to the given Go value, and returns the Key.
//
// The key here is a sizeof(pointer)/2 atomic, that is simply incremented by two
// on each call.  On a 64-bit platform, this key-space is so large that is will
// unlikely ever run out during the lifetime of a program... but if it does, we
// panic.  To avoid running out of space on a 32-bit platform (where
// 2,147,483,648 mappings are possible), use MapPtrPair instead.
func (mapper *Mapper) MapValue(goValue interface{}) Key {
	key := Key{atomic.AddUintptr(&mapper.atomicKey, 2) | 0x1}
	// Crash on wrap-around
	if key.v == 0 {
		panic("mapper: key space exhausted")
	}
	mapper.doMap(key, goValue)
	return key
}

// atromicKey is a sizeof(pointer)/2 value (lower bit is reserved) that is
// incremented for each new Key "allocation".
var atomicKey uintptr

// Get retrieves the Go value from key.
func (mapper *Mapper) Get(key Key) (goValue interface{}) {
	mapper.mux.RLock()
	goValue, ok := mapper.m[key]
	if !ok {
		panic(fmt.Errorf("mapper: key not mapped: 0x%x", key))
	}
	mapper.mux.RUnlock()
	return
}

// GetPtr calls Get after first converting ptr to a Key.
func (mapper *Mapper) GetPtr(ptr unsafe.Pointer) (goValue interface{}) {
	key := Key{uintptr(ptr)}
	return mapper.Get(key)
}

// Delete an existing mapping via the key.
func (mapper *Mapper) Delete(key Key) {
	mapper.mux.Lock()
	delete(mapper.m, key)
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
