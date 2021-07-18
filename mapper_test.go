// Copyright 2021 John Papandriopoulos.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mapper_test

import (
	"testing"

	itest "go.jpap.org/mapper/internal/testing"
)

func TestMapCgoPointer(t *testing.T) {
	itest.RunTestMapCgoPointer(t)
}

func TestMapGoKey(t *testing.T) {
	itest.RunTestMapGoKey(t)
}
