package sparse_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/dnesting/sparse"
)

var (
	_ sparse.Reader     = (*sparse.Buffer)(nil)
	_ sparse.ReadFinder = (*sparse.Buffer)(nil)
	_ io.WriterAt       = (*sparse.Buffer)(nil)
	_ io.WriteSeeker    = (*sparse.Buffer)(nil)
)

func TestBufferSimple(t *testing.T) {
	var sb sparse.Buffer
	abc := []byte("ABC")

	n, err := sb.WriteAt(abc, 0)
	if n != 3 || err != nil {
		t.Errorf("WriteAt should return n=3 err=nil, got n=%d err=%v", n, err)
	}

	roff, _, err := sb.Find(0)
	if roff != 0 {
		t.Errorf("Find(0) should return off=0, got %d", roff)
	}
	if err != nil {
		t.Errorf("Find(0) should not return an error, got %v", err)
	}
	buf := make([]byte, 10)
	n, err = sb.Read(buf)
	if n != 3 {
		t.Errorf("ReadAt(0) should return n=3, got n=%d", n)
	}
	if !bytes.Equal(buf[:3], abc) {
		t.Errorf("ReadAt(0) should have read %s, got %s", string(abc), string(buf[:3]))
	}
	if sb.Size() != 3 {
		fmt.Errorf("Buffer should have 3 bytes in it, got Size of %d", sb.Size())
	}
}

func TestBufferOverlap(t *testing.T) {
	testBufferOverlapMaybeStore(t, false)
}

func TestBufferOverlapWithStore(t *testing.T) {
	testBufferOverlapMaybeStore(t, true)
}

func readBuf(sb sparse.ReadFinder, p []byte) (n int) {
	for {
		off, _, err := sb.Find(int64(n))
		if err != nil {
			break
		}
		nn, _ := sb.Read(p[off:])
		n = int(off) + nn
	}
	return
}

func testBufferOverlapMaybeStore(t *testing.T, useStore bool) {
	var sb sparse.Buffer
	expected := []string{
		//123456789012345
		"CCC..AAA.....BBB",
		".CCC.AAA.....BBB",
		"..CCCAAA.....BBB",
		"...CCCAA.....BBB",
		"....CCCA.....BBB",
		".....CCC.....BBB",
		".....ACCC....BBB",
		".....AACCC...BBB",
		".....AAACCC..BBB",
		".....AAA.CCC.BBB",
		".....AAA..CCCBBB",
		".....AAA...CCCBB",
		".....AAA....CCCB",
		".....AAA.....CCC",
		".....AAA.....BCCC",
		".....AAA.....BBCCC",
		".....AAA.....BBBCCC",
		".....AAA.....BBB.CCC",
		".....AAA.....BBB..CCC",
	}
	for i := 0; i < len(expected); i++ {
		sb.Reset()
		if useStore {
			sb.StoreAt([]byte("AAA"), 5)
			sb.StoreAt([]byte("BBB"), 13)
			sb.StoreAt([]byte("CCC"), int64(i))
		} else {
			sb.WriteAt([]byte("AAA"), 5)
			sb.WriteAt([]byte("BBB"), 13)
			sb.WriteAt([]byte("CCC"), int64(i))
		}
		var buf [24]byte
		n := readBuf(&sb, buf[:])
		actual := printable(buf[:n])
		if actual != expected[i] || n != len(expected[i]) {
			t.Errorf("Overlapping writes %2d, expected %-20s (%d), got %-20s (%d):\n  %v", i, expected[i], len(expected[i]), actual, n, sb)
		}
	}

	// And one special one
	sb.Reset()
	// 012345678
	// AA.BB.CC
	//  DDDDDD
	if useStore {
		sb.StoreAt([]byte("AA"), 0)
		sb.StoreAt([]byte("BB"), 3)
		sb.StoreAt([]byte("CC"), 6)
		sb.StoreAt([]byte("DDDDDD"), 1)
	} else {
		sb.WriteAt([]byte("AA"), 0)
		sb.WriteAt([]byte("BB"), 3)
		sb.WriteAt([]byte("CC"), 6)
		sb.WriteAt([]byte("DDDDDD"), 1)
	}
	var buf [20]byte
	n := readBuf(&sb, buf[:])
	actual := printable(buf[:n])
	if actual != "ADDDDDDC" {
		t.Errorf("Overlapping write expected ADDDDDDC, got %s (%d)", actual, n)
	}
}

func TestBufferSetAt(t *testing.T) {
	var sb sparse.Buffer
	aaa := []byte("AAA")
	bbb := []byte("BBB")
	type T struct {
		off int64
		b   []byte
	}
	for ci, c := range []struct {
		test   []T
		expect string
		desc   string
	}{
		{[]T{{1, aaa}, {7, aaa}}, ".A!A...A!A", "should be original slice"},
		{[]T{{1, aaa}, {4, bbb}, {7, aaa}}, ".A!AB*BA!A", "should be original even when sandwiched (order 147)"},
		{[]T{{1, aaa}, {7, aaa}, {4, bbb}}, ".A!AB*BA!A", "should be original even when sandwiched (order 177)"},
		{[]T{{4, bbb}, {1, aaa}, {7, aaa}}, ".A!AB*BA!A", "should be original even when sandwiched (order 417)"},
		{[]T{{4, bbb}, {7, aaa}, {1, aaa}}, ".A!AB*BA!A", "should be original even when sandwiched (order 471)"},
		{[]T{{7, aaa}, {1, aaa}, {4, bbb}}, ".A!AB*BA!A", "should be original even when sandwiched (order 714)"},
		{[]T{{7, aaa}, {4, bbb}, {1, aaa}}, ".A!AB*BA!A", "should be original even when sandwiched (order 741)"},
	} {
		sb.Reset()
		aaa[1] = 'A'
		bbb[1] = 'B'
		buf := make([]byte, 20)
		for _, tst := range c.test {
			sb.StoreAt(tst.b, tst.off)
		}
		aaa[1] = '!'
		bbb[1] = '*'
		n := readBuf(&sb, buf)
		actual := printable(buf[:n])
		if actual != c.expect {
			t.Errorf("%d: writes %v expected %s, got %s", ci, c.test, c.expect, actual)
		}
	}
}

func TestBufferNext(t *testing.T) {
	var sb sparse.Buffer
	sb.WriteAt([]byte("AAA"), 0)
	sb.WriteAt([]byte("BBB"), 5)
	sb.Next() // skip AAA and move to BBB
	got := make([]byte, 5)
	n, _ := sb.Read(got)
	if string(got[:n]) != "BBB" {
		t.Errorf("Next should advance to permit a Read of %q, got %q", "BBB", got[:n])
	}

	sb.Find(5)
	skip, err := sb.Next()
	if skip != 3 || err != nil {
		t.Errorf("Next should advance 3, got %d, %v", skip, err)
	}

	n, err = sb.Read(got)
	if n != 0 || err != io.EOF {
		t.Errorf("Read should return 0,nil, got %d,%v", n, err)
	}

	skip, err = sb.Next()
	if skip != 0 || err != io.EOF {
		t.Errorf("Next should return io.EOF, got skip=%d err=%v", skip, err)
	}
}

func TestTruncate(t *testing.T) {
	var sb sparse.Buffer
	sb.Truncate(3)

	got := make([]byte, 5)
	_, err := sb.Read(got)
	if err != io.EOF {
		t.Errorf("Truncate first read should get io.EOF, got %v", err)
	}

	skip, err := sb.Next()
	if skip != 3 || err != nil {
		t.Errorf("Truncate then Next should get skip=3, err=nil, got %d, %v", skip, err)
	}

	_, err = sb.Read(got)
	if err != io.EOF {
		t.Errorf("Truncate second read should get io.EOF, got %v", err)
	}

	skip, err = sb.Next()
	if skip != 0 || err != io.EOF {
		t.Errorf("Truncate then Next should get skip=0, err=io.EOF, got %d, %v", skip, err)
	}
}

func printable(b []byte) string {
	return strings.Replace(string(b), "\000", ".", -1)
}

func ExampleBuffer() {
	var sb sparse.Buffer
	sb.WriteAt([]byte("AAA"), 2)
	sb.WriteAt([]byte("BBB"), 7)
	got := make([]byte, 20)

	// Using the Finder interface
	off, size, _ := sb.Find(0)
	n, _ := sb.Read(got)
	fmt.Printf("offset %d, size %d, found %q\n", off, size, string(got[:n]))

	off, size, _ = sb.Find(off + size)
	n, _ = sb.Read(got)
	fmt.Printf("offset %d, size %d, found %q\n", off, size, string(got[:n]))

	// Using the Reader interface
	sb.Seek(0, io.SeekStart)

	skipped, _ := sb.Next()
	n, _ = sb.Read(got)
	fmt.Printf("skipped %d, found %q\n", skipped, string(got[:n]))

	skipped, _ = sb.Next()
	n, _ = sb.Read(got)
	fmt.Printf("skipped %d, found %q\n", skipped, string(got[:n]))

	// Output:
	// offset 2, size 3, found "AAA"
	// offset 7, size 3, found "BBB"
	// skipped 2, found "AAA"
	// skipped 2, found "BBB"
}
