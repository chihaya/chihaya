// Copyright 2013 The Bufferpool Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bufferpool_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/pushrax/bufferpool"
)

func ExampleNew() {
	bp := bufferpool.New(10, 255)

	dogBuffer := bp.Take()
	dogBuffer.WriteString("Dog!")
	bp.Give(dogBuffer)

	catBuffer := bp.Take() // dogBuffer is reused and reset.
	catBuffer.WriteString("Cat!")

	fmt.Println(catBuffer)
	// Output:
	// Cat!
}

func TestTakeFromEmpty(t *testing.T) {
	bp := bufferpool.New(1, 1)
	poolBuf := bp.Take()
	if !bytes.Equal(poolBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from empty bufferpool was allocated incorrectly.")
	}
}

func TestTakeFromFilled(t *testing.T) {
	bp := bufferpool.New(1, 1)

	origBuf := bytes.NewBuffer([]byte("X"))
	bp.Give(origBuf)

	reusedBuf := bp.Take()
	if !bytes.Equal(reusedBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from filled bufferpool was recycled incorrectly.")
	}

	// Compare addresses of the first element in the underlying slice.
	if &origBuf.Bytes()[:1][0] != &reusedBuf.Bytes()[:1][0] {
		t.Fatalf("Recycled buffer points at different address.")
	}
}

func TestSliceSemantics(t *testing.T) {
	bp := bufferpool.New(1, 8)

	buf := bp.Take()
	buf.WriteString("abc")
	bp.Give(buf)

	buf2 := bp.TakeSlice()

	if !bytes.Equal(buf2[:3], []byte("abc")) {
		t.Fatalf("Buffer from filled bufferpool was recycled incorrectly.")
	}
}
