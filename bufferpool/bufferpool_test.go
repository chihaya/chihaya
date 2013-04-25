package bufferpool_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kotokoko/chihaya/bufferpool"
)

func TestTakeFromEmpty(t *testing.T) {
	poolSize := 1
	bufSize := 1
	bp := bufferpool.New(poolSize, bufSize)
	poolBuf := bp.Take()
	newBuf := bytes.NewBuffer(make([]byte, 0, bufSize))
	if !bytes.Equal(poolBuf.Bytes(), newBuf.Bytes()) {
		t.Fatalf("Buffer from empty bufferpool was allocated incorrectly.")
	}
}

func TestTakeFromFilled(t *testing.T) {
	poolSize := 1
	bufSize := 1
	bp := bufferpool.New(poolSize, bufSize)
	bp.Give(bytes.NewBuffer([]byte("X")))
	reusedBuf := bp.Take()
	if !bytes.Equal(reusedBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from empty bufferpool was recycled incorrectly.")
	}
}

func ExampleNew() {
	catBuffer := bytes.NewBuffer([]byte("cat"))
	bp := bufferpool.New(10, catBuffer.Len())
	bp.Give(catBuffer) // An error is returned, but not neccessary to check
	reusedBuffer := bp.Take()
	reusedBuffer.Write([]byte("dog"))
	fmt.Println(reusedBuffer)
	// Output:
	// dog
}
