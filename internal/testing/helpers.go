// Copyright 2021 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package testing allows us to use cgo in Go tests.
package testing

/*
#include <stdlib.h>
#include <string.h>

typedef struct {
	void *user;
} object_t;

static object_t *allocObject(void *user) {
	object_t *obj = (object_t *)malloc(sizeof(object_t));
	obj->user = user;
	return obj;
}

static void freeObject(object_t *obj) {
	free(obj);
}

extern void goWorkCallback(object_t *obj, void *userWork);

static void objDoWork(object_t *obj, void *userWork) {
	goWorkCallback(obj, userWork);
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
	obj := C.allocObject(nil)
	if obj == nil {
		panic("obj alloc failure")
	}
	defer C.freeObject(obj)

	called := false
	goObj := GoObject{goCallback: func() {
		called = true
	}}

	// Map based on the C.object_t pointer
	key := mapper.G.MapPtrPair(unsafe.Pointer(obj), goObj)

	// Pass the key as the work-user pointer.
	C.objDoWork(obj, key.Handle())
	if !called {
		t.Fatal("callback did not run")
	}
}

func RunTestMapGoKey(t *testing.T) {
	called := false
	goObj := GoObject{goCallback: func() {
		called = true
	}}

	// Create a unique key
	key := mapper.G.MapValue(goObj)

	// Key is the user pointer here
	obj := C.allocObject(key.Handle())
	if obj == nil {
		panic("obj alloc failure")
	}
	defer C.freeObject(obj)

	C.objDoWork(obj, nil)
	if !called {
		t.Fatal("callback did not run")
	}
}

//export goWorkCallback
func goWorkCallback(obj *C.object_t, workUser unsafe.Pointer) {
	// Get the user pointer; and if not set, the work-user pointer.
	ptr := unsafe.Pointer(obj.user)
	if ptr == nil {
		ptr = workUser
	}

	// Exchange the user pointer for the Go object.
	goObj := mapper.G.GetPtr(ptr).(GoObject)

	// Call the Go object's callback.
	goObj.goCallback()
}
