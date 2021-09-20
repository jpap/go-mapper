// Copyright 2021 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testing allows us to use cgo in Go tests.
package testing

/*
#include <stdlib.h>
#include <string.h>

typedef struct {
	void *user;
} object_t;

// Note the use of uintptr_t here!  If using an external API, you would need to
// write a wrapper; see (Key).handle for details.
static object_t *allocObject(uintptr_t objUserPtr) {
	object_t *obj = (object_t *)malloc(sizeof(object_t));
	obj->user = (void *)objUserPtr;
	return obj;
}

static void freeObject(object_t *obj) {
	free(obj);
}

// Note the use of uintptr_t here!  If using an external API, you would need to
// typecast the function pointer to this type.
extern void goWorkCallback(object_t *obj, uintptr_t objUserPtr, uintptr_t workUserPtr);

// Note the use of uintptr_t here!
static void objDoWork(object_t *obj, uintptr_t callUserPtr) {
	goWorkCallback(obj, (uintptr_t)obj->user, callUserPtr);
}
*/
import "C"
import (
	"testing"
	"unsafe"

	"go.jpap.org/mapper"
)

type GoObject struct {
	goCallback func()
}

func RunTestMapCgoPointer(t *testing.T) {
	// No user pointer here.
	obj := C.allocObject(0)
	if obj == nil {
		panic("obj alloc failure")
	}
	defer C.freeObject(obj)

	called := false
	goObj := GoObject{goCallback: func() {
		called = true
	}}

	// Map based on the C.object_t pointer.
	//
	// Note that [obj] is a valid pointer into the process memory space, so the
	// conversion to unsafe.Pointer is valid here.
	key := mapper.G.MapPtrPair(unsafe.Pointer(obj), goObj)
	defer mapper.G.Delete(key)

	// Pass the key as the work-user pointer.
	//
	// Note that we have to pass the handle as C.uintptr_t, hence the typecast.
	C.objDoWork(obj, C.uintptr_t(key.Handle()))
	if !called {
		t.Fatal("callback using cgo pointer did not run")
	}
}

func RunTestMapGoKey(t *testing.T) {
	called := false
	goObj := GoObject{goCallback: func() {
		called = true
	}}

	// Create a unique key
	key := mapper.G.MapValue(goObj)
	defer mapper.G.Delete(key)

	// Key is the user pointer here
	obj := C.allocObject(C.uintptr_t(key.Handle()))
	if obj == nil {
		panic("obj alloc failure")
	}
	defer C.freeObject(obj)

	C.objDoWork(obj, 0)
	if !called {
		t.Fatal("callback using handle did not run")
	}
}

//export goWorkCallback
func goWorkCallback(obj *C.object_t, objUserPtr, _ uintptr) {
	// Get the Go object from the object; if not set, use the work-user handle.
	var goObj GoObject
	if objUserPtr != 0 {
		goObj = mapper.G.GetHandle(objUserPtr).(GoObject)
	} else {
		// Note that [obj] is a valid pointer into the process' memory space, so we
		// can cast to unsafe.Pointer here.
		goObj = mapper.G.GetPtr(unsafe.Pointer(obj)).(GoObject)
	}

	// Call the Go object's callback.
	goObj.goCallback()
}
