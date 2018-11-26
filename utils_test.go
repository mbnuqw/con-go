package con

import (
	"bytes"
	"fmt"
	"testing"
)

func TestUID(t *testing.T) {
	id := UID()

	// Check len
	if len(id) != 12 {
		t.Errorf("Length of uid wrong, expected: 12, got: %d", len(id))
	}

	// At least next id is different...
	if id == UID() {
		t.Error("Got two equal ids")
	}
}

func TestBID12(t *testing.T) {
	id := BID12()

	// Check len
	if len(id) != 12 {
		t.Errorf("Length of uid wrong, expected: 12, got: %d", len(id))
	}
}

func TestBinEq(t *testing.T) {
	a := []byte{0x1, 0x2, 0x3, 0x4}
	b := []byte{0x1, 0x2, 0x3, 0x4}
	c := []byte{0xff, 0x15, 0x4, 0x32}
	x := []byte{0x1, 0x2, 0x3}

	if !BinEq(a, b) {
		t.Error("Two equal slices should be treated as equal")
	}

	if !BinEq([]byte{}, []byte{}) {
		t.Error("Two empty slices should be treated as equal")
	}

	if BinEq(a, []byte{}) {
		t.Error("Filled slice and empty one should be treated as different")
	}

	if BinEq([]byte{}, b) {
		t.Error("Empty slice and filled one should be treated as different")
	}

	if BinEq(b, c) {
		t.Error("Different slices with the same length should be treated as different")
	}

	if BinEq(a, x) {
		t.Error("Slices with the different length should be treated as different")
	}
}

func TestWriteMsg(t *testing.T) {
	buf := make([]byte, 0, 50)
	consumer := bytes.NewBuffer(buf)

	writeMsg(consumer, BID12(), 0x00, "Test name", []byte("Test body"))
	msgBytes := consumer.Bytes()

	fmt.Println(" → msgBytes", msgBytes)

	if len(msgBytes) != 40 {
		t.Errorf("Wrong message len: %d", len(msgBytes))
	}

	expected := []byte{
		// Some Id
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// Meta
		0,
		// Name len
		9,
		// Name
		84, 101, 115, 116, 32, 110, 97, 109, 101,
		// Body len
		0, 0, 0, 0, 0, 0, 0, 9,
		// Body
		84, 101, 115, 116, 32, 98, 111, 100, 121,
	}
	if !BinEq(msgBytes[12:], expected[12:]) {
		t.Error("Unexpected output")
	}
}
