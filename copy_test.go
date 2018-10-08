package sparse_test

import (
	"testing"

	"github.com/dnesting/sparse"
)

func TestCopy(t *testing.T) {
	var a, b sparse.Buffer
	a.WriteAt([]byte("AAA"), 2)
	a.WriteAt([]byte("BBB"), 7)

	sparse.Copy(&b, &a)

	buf := make([]byte, 10)
	ofs, size, err := b.Find(0)
	if err != nil {
		t.Fatalf("Copy failed to copy, b.Find(0) returned %d, %d, %v", ofs, size, err)
	}
	n, _ := b.Read(buf)
	if string(buf[:n]) != "AAA" {
		t.Errorf("Failed to copy, expected AAA, got %q", buf[:n])
	}

	nofs := ofs + size
	ofs, size, err = b.Find(nofs)
	if err != nil {
		t.Fatalf("Copy failed to copy, b.Find(%d) returned %d, %d, %v", nofs, ofs, size, err)
	}
	n, _ = b.Read(buf)
	if string(buf[:n]) != "BBB" {
		t.Errorf("Failed to copy, expected AAA, got %q", buf[:n])
	}
}
