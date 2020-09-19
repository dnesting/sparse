package sparse_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/dnesting/sparse"
)

func TestFromReader(t *testing.T) {
	type R struct {
		data    string
		skipped int64
	}
	type T struct {
		input    string
		consec   int64
		expected []R
	}
	for ci, c := range []T{
		{"abc", 3, []R{{"abc", 0}}},
		{"\000abc", 3, []R{{"\000abc", 0}}},
		{"\000\000abc", 3, []R{{"\000\000abc", 0}}},
		{"\000\000\000abc", 3, []R{{"abc", 3}}},
		{"\000\000\000\000abc", 3, []R{{"abc", 4}}},
		{"abc\000", 3, []R{{"abc\000", 0}}},
		{"abc\000\000", 3, []R{{"abc\000\000", 0}}},
		{"abc\000\000\000", 3, []R{{"abc", 0}, {"", 3}}},
		{"abc\000\000\000\000", 3, []R{{"abc", 0}, {"", 4}}},
		{"abc\000def", 3, []R{{"abc\000def", 0}}},
		{"abc\000\000def", 3, []R{{"abc\000\000def", 0}}},
		{"abc\000\000\000def", 3, []R{{"abc", 0}, {"def", 3}}},
		{"abc\000\000\000\000def", 3, []R{{"abc", 0}, {"def", 4}}},
	} {
		t.Run(fmt.Sprintf("%q", c.input), func(t *testing.T) {
			input := strings.NewReader(c.input)
			r := sparse.Make(input, c.consec)
			var i int
			for i = 0; i < 10; i++ {
				var skip int64
				var got []byte
				var err error
				got, err = ioutil.ReadAll(r)
				if len(got) == 0 {
					skip, err = r.Next()
					if err != nil {
						break
					}
					got, err = ioutil.ReadAll(r)
				}
				if i-1 > len(c.expected) {
					t.Errorf("%d: unexpected result (%d>%d): skip=%d\n%s", ci, i, len(c.expected), skip, hex.Dump(got))
					break
				}
				if i < len(c.expected) {
					if skip != c.expected[i].skipped || !bytes.Equal([]byte(c.expected[i].data), got) {
						t.Errorf("%d: data wrong, want skip=%d, got skip=%d\nwant data:\n%sgot data:\n%s", ci,
							c.expected[i].skipped, skip, hex.Dump([]byte(c.expected[i].data)), hex.Dump(got))
					}
				}
			}
			if i < len(c.expected) {
				t.Errorf("%d: wanted %d results (skip=%d %q), got %d", ci, len(c.expected), c.expected[i].skipped, c.expected[i].data, i)
			}
		})
	}
}

func ExampleFromReader() {
	// This example creates a byte slice with spans of zeros in them, and converts
	// that slice into sparse segments, showing how those spans then get iterated on.

	// Start with a sparse buffer populated with a couple of segments of data.
	var orig sparse.Buffer
	orig.WriteAt([]byte("AAA"), 0x08)
	orig.WriteAt([]byte("BBB"), 0x18)

	// sparse.Reader.Read will fill the gaps with zeros, producing a regular non-sparse stream.
	data, _ := ioutil.ReadAll(sparse.NewReader(&orig, nil))
	fmt.Print(hex.Dump(data))
	fmt.Println()

	// Make will create a sparse Reader that will search for those zeros and convert the
	// bytes back into sparse segments.
	rd := sparse.Make(bytes.NewBuffer(data), 5) // 5 contiguous zeros = skip the span

	// And just to demonstrate that:
	for i := 0; i < 10; i++ {
		skip, err := rd.Next()
		if err == io.EOF {
			break
		}
		d, _ := ioutil.ReadAll(rd)
		fmt.Printf("iter.Next %d skipped %d bytes then gave us %q\n", i+1, skip, string(d))
	}

	// From here it's pretty easy to do the full round-trip and copy back into a Buffer.
	var target sparse.Buffer
	rd = sparse.Make(bytes.NewBuffer(data), 5)
	sparse.Copy(&target, rd)

	off, _, _ := target.Find(0x16)
	data = make([]byte, 5)
	n, _ := target.Read(data)
	fmt.Printf("Found %q at 0x%X", string(data[:n]), off)

	// Output:
	// 00000000  00 00 00 00 00 00 00 00  41 41 41 00 00 00 00 00  |........AAA.....|
	// 00000010  00 00 00 00 00 00 00 00  42 42 42                 |........BBB|
	//
	// iter.Next 1 skipped 8 bytes then gave us "AAA"
	// iter.Next 2 skipped 13 bytes then gave us "BBB"
	// Found "BBB" at 0x18
}
