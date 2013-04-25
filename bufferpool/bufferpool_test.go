package bufferpool_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kotokoko/chihaya/bufferpool"
)

func TestTakeFromEmpty(t *testing.T) {
	bp := bufferpool.New(1, 1)
	poolBuf := bp.Take()
	if !bytes.Equal(poolBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from empty bufferpool was allocated incorrectly.")
	}
}

func TestTakeFromFilled(t *testing.T) {
	bp := bufferpool.New(1, 1)
	bp.Give(bytes.NewBuffer([]byte("X")))
	reusedBuf := bp.Take()
	if !bytes.Equal(reusedBuf.Bytes(), []byte("")) {
		t.Fatalf("Buffer from filled bufferpool was recycled incorrectly.")
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
