// Copyright 2020 John Papandriopoulos.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package mapper

import (
	"fmt"
	"sync"
	"unsafe"
)

// Mapper maps between Go values and pointers suitable for passing to C via Cgo.
type Mapper struct {
	m sync.Map
}

type mapperKey struct {
	// We can't have an empty struct, otherwise allocations are not distinct.
	_ uint8
}

// NewMapper creates a new Mapper.
func NewMapper() *Mapper { return &Mapper{} }

// New creates a new pointer to pass to C via Cgo, that can be later used
// with Get to obtain the Go value v, typically from a C to Go callback func.
func (mapper *Mapper) New(v interface{}) unsafe.Pointer {
	// Create a new unique token by using the pointer value.
	//
	// This value can safely be passed to C via Cgo because it doesn't
	// contain any pointers to Go memory.
	//
	// We could've also used an atomic counter, and typecasted it to a pointer
	// value; might be a good idea to profile it vs this approach.  The advantage
	// there is that it puts less pressure on the GC.
	k := &mapperKey{}
	mapper.m.Store(k, v)
	return unsafe.Pointer(k)
}

// Get retrieves the Go value v from the Cgo pointer k.
func (mapper *Mapper) Get(k unsafe.Pointer) (v interface{}) {
	var ok bool
	v, ok = mapper.m.Load((*mapperKey)(k))
	if !ok {
		panic(fmt.Errorf("invalid cgo ptr: %p", k))
	}
	return
}

// Delete the Cgo pointer k.
func (mapper *Mapper) Delete(k unsafe.Pointer) {
	mapper.m.Delete(k)
}
