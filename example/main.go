// Copyright 2021 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

/*
#include <stdint.h>

extern void goCallback(void *user);

static void invokeCfuncThatCallsGoCallback(uintptr_t user) {
	goCallback(user);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"go.jpap.org/mapper"
)

type myStruct struct {
	msg string
}

func main() {
	s := myStruct{"hello world"}

	key := mapper.G.MapValue(s)
	defer mapper.G.Delete(key)

	C.invokeCfuncThatCallsGoCallback(C.uintptr_t(key.Handle()))
}

//export goCallback
func goCallback(user unsafe.Pointer) {
	s := mapper.G.GetPtr(user).(myStruct)
	fmt.Printf("goCallback got s: %#v\n", s)
}
