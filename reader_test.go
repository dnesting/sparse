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

func TestReaderSeek(t *testing.T) {
	var sbuf sparse.Buffer
	// 01234567890123456789
	// ..ABC...DEFGHI...JKL
	sbuf.StoreAt([]byte("ABC"), 2)
	sbuf.StoreAt([]byte("DEF"), 8)
	sbuf.StoreAt([]byte("GHI"), 11)
	sbuf.StoreAt([]byte("JKL"), 17)

	b := sparse.NewReadSeeker(&sbuf, nil)
	buf := make([]byte, 5)

	var tests = []struct {
		Pos     int64
		Whence  int
		Abs     int64
		Read    string
		ReadErr error
	}{
		{2, io.SeekStart, 2, "ABC\x00\x00", nil},    // pos now 7
		{-7, io.SeekCurrent, 0, "\x00\x00ABC", nil}, // pos now 5
		{100, io.SeekCurrent, 105, "", io.EOF},
		{-3, io.SeekEnd, 17, "JKL", io.EOF},

		{2, sparse.SeekData, 2, "ABC\x00\x00", nil},
		{2, sparse.SeekHole, 5, "\x00\x00\x00DE", nil},
		{5, sparse.SeekData, 8, "DEFGH", nil},
		{8, sparse.SeekHole, 14, "\x00\x00\x00JK", nil},
		{18, sparse.SeekHole, 20, "", io.EOF},
	}

	for _, test := range tests {
		pos, err := b.Seek(test.Pos, test.Whence)
		if pos != test.Abs || err != nil {
			t.Errorf("Seek(%d, %d) should seek to %d, with err=nil, got %d, %v\n", test.Pos, test.Whence, test.Abs, pos, err)
			continue
		}
		n, err := b.Read(buf)
		if n != len(test.Read) || err != test.ReadErr || string(buf[:n]) != test.Read {
			t.Errorf("Seek(%d, %d) then Read() should read %q/%v, got %q, %v\n", test.Pos, test.Whence, test.Read, test.ReadErr, string(buf[:n]), err)
		}
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
