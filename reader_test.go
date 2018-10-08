package sparse_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/dnesting/sparse"
)

var (
	_ io.ReadSeeker = (*sparse.ReadSeeker)(nil)
	_ io.Reader     = sparse.Zero
	_ io.ReaderAt   = sparse.Zero
)

func TestEmpty(t *testing.T) {
	var sbuf sparse.Buffer
	b := sparse.NewReadSeeker(&sbuf, nil)
	buf := make([]byte, 3)

	n, err := b.ReadAt(buf, 0)
	if n != 0 || err != io.EOF {
		t.Errorf("ReadAt on empty buffer should return n=0 err=EOF, got n=%d err=%v", n, err)
	}

	n, err = b.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("Read on empty buffer should return n=0 err=EOF, got n=%d err=%v", n, err)
	}

	if sbuf.Size() != 0 {
		t.Errorf("Empty buffer should have size zero, got %d", sbuf.Size())
	}
}

func TestSimple(t *testing.T) {
	var sbuf sparse.Buffer
	b := sparse.NewReadSeeker(&sbuf, nil)

	abc := []byte{'A', 'B', 'C'}

	n, err := sbuf.WriteAt(abc, 0)
	if n != 3 || err != nil {
		t.Errorf("WriteAt should return n=3 err=nil, got n=%d err=%v", n, err)
	}

	buf := make([]byte, 10)
	n, err = b.ReadAt(buf[:3], 0)
	if n != 3 || err != nil {
		t.Errorf("ReadAt(0) should return n=3 err=nil, got n=%d err=%v", n, err)
	}
	if !bytes.Equal(buf[:3], abc) {
		t.Errorf("ReadAt(0) should have read %s, got %s", string(abc), string(buf[:3]))
	}

	n, err = b.ReadAt(buf, 0)
	if n != 3 || err != nil && err != io.EOF {
		t.Errorf("ReadAt(0) should return n=3 err=nil, got n=%d err=%v\n%v", n, err, b)
	}
	if !bytes.Equal(buf[:3], abc) {
		t.Errorf("ReadAt(0) should have read %v, got %v", abc, buf[:3])
	}

	if sbuf.Size() != 3 {
		fmt.Errorf("Buffer should have 3 bytes in it, got Size of %d", sbuf.Size())
	}
}

func TestZeroFill(t *testing.T) {
	var sbuf sparse.Buffer
	b := sparse.NewReadSeeker(&sbuf, nil)
	abc := []byte{'A', 'B', 'C'}

	n, err := sbuf.WriteAt(abc, 2)
	if n != 3 || err != nil {
		t.Errorf("WriteAt should return n=3 err=nil, got n=%d err=%v", n, err)
	}

	buf := make([]byte, 10)
	n, err = b.ReadAt(buf, 0)
	if n != 5 || err != nil && err != io.EOF {
		t.Errorf("ReadAt should return n=5 err=nil, got n=%d err=%v", n, err)
	}
	if !bytes.Equal(buf[:5], []byte{0, 0, 'A', 'B', 'C'}) {
		t.Errorf("ReadAt should have read %v, got %v", abc, buf[:5])
	}

	n, err = b.Read(buf)
	if n != 5 || err != nil && err != io.EOF {
		t.Errorf("ReadAt should return n=5 err=nil, got n=%d err=%v", n, err)
	}
	if !bytes.Equal(buf[:5], []byte{0, 0, 'A', 'B', 'C'}) {
		t.Errorf("ReadAt should have read %v, got %v", abc, buf[:5])
	}

	n, err = b.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("Read after exhausting buffer should return n=0 err=EOF, got n=%d err=%v", n, err)
	}

	if sbuf.Size() != 5 {
		t.Errorf("With zeros, buffer should have Size 5, got %d", sbuf.Size())
	}
}

/*
func TestWrite(t *testing.T) {
	var sbuf sparse.Bytes
	rw := sparse.NewReadWriter(&sbuf, nil)
	var err error
	n, err := rw.Seek(5, io.SeekStart)
	if n != 5 || err != nil {
		t.Errorf("Seek should result in n=5, err=nil, got n=%d, err=%v", n, err)
	}

	nr, _ := rw.Write([]byte("AAA"))
	if nr != 3 {
		t.Errorf("%v: expected n=%d, got %d", rw, 3, nr)
	}

	buf := make([]byte, 10)
	rw.Seek(2, io.SeekStart)
	nr, _ = rw.Read(buf)
	if nr != 6 {
		t.Errorf("expected read of 6 bytes, got %d", nr)
	}
	expected := "...AAA"
	actual := printable(buf[:nr])
	if expected != actual {
		t.Errorf("%v: expected to read %s, got %s", rw, expected, actual)
	}

	n, _ = rw.Seek(-3, io.SeekCurrent)
	if n != 5 {
		t.Errorf("Seek(current-3) should give n=%d, got %d", 5, n)
	}
	nr, _ = rw.Read(buf)
	if nr != 3 {
		t.Errorf("expected read of 3 bytes, got %d", nr)
	}
	expected = "AAA"
	actual = printable(buf[:nr])
	if expected != actual {
		t.Errorf("expected to read %s, got %s", expected, actual)
	}

	n, _ = rw.Seek(-2, io.SeekEnd)
	if n != 6 {
		t.Errorf("Seek(current-3) should give n=%d, got %d", 6, n)
	}
	nr, _ = rw.Read(buf)
	if nr != 2 {
		t.Errorf("expected read of 2 bytes, got %d", nr)
	}
	expected = "AA"
	actual = printable(buf[:nr])
	if expected != actual {
		t.Errorf("expected to read %s, got %s", expected, actual)
	}
}
*/

func ExampleReader() {
	var sb sparse.Buffer
	sb.WriteAt([]byte("AAA"), 2)
	sb.WriteAt([]byte("BBB"), 7)
	b := sparse.NewReader(&sb, nil)

	got, _ := ioutil.ReadAll(b)
	fmt.Print(hex.Dump(got))
	// Output:
	// 00000000  00 00 41 41 41 00 00 42  42 42                    |..AAA..BBB|
}
