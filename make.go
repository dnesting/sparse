package sparse

import (
	"bufio"
	"io"
)

type maker struct {
	scan     *bufio.Scanner
	minZeros int64

	zeros int64
	data  []byte

	err error // only cleared by Next
}

// splitFunc is a bufio.SplitFunc that splits on sequences of zeros.
func (m *maker) splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for advance < len(data) && data[advance] == 0 {
		advance++
		m.zeros++
	}
	start := advance
	for advance < len(data) && data[advance] != 0 {
		advance++
	}
	if start != advance {
		token = data[start:advance]
	}
	return
}

func (m *maker) Read(p []byte) (n int, err error) {
	for n < len(p) {
		if m.err != nil {
			// Some persistent error from Scan, often io.EOF.
			err = m.err
			break
		}
		if m.zeros >= m.minZeros {
			// Stop reading with io.EOF until we're advanced with Next.
			break
		}
		for m.zeros > 0 {
			// Scan counted m.zeros < m.tooManyZeros, so emit them as data.
			p[n] = 0
			n++
			m.zeros--
		}
		nn := copy(p[n:], m.data)
		copy(m.data, m.data[nn:])
		m.data = m.data[:len(m.data)-nn]
		n += nn
		if len(m.data) == 0 {
			m.readMore()
		}
	}
	if len(p) > 0 && n == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (m *maker) Next() (skip int64, err error) {
	if m.zeros >= m.minZeros {
		skip = m.zeros
		m.zeros = 0
	}
	err = m.err
	return
}

func (m *maker) readMore() {
	wantMore := m.scan.Scan()
	m.data = m.scan.Bytes()
	if m.scan.Err() != nil {
		// We've reached some terminal condition with the input reader, so make
		// this error persistent.
		m.err = m.scan.Err()
	}
	if !wantMore && m.zeros == 0 && len(m.data) == 0 {
		// We've reached the normal end of the input reader
		m.err = io.EOF
	}
}

// Make takes the stream from r, and produces a sparse Reader that reads
// segments of bytes that lie between sequences of minZeros or more zeros.
func Make(r io.Reader, minZeros int64) Reader {
	return newMaker(r, minZeros)
}

func newMaker(r io.Reader, minZeros int64) *maker {
	m := &maker{
		scan:     bufio.NewScanner(r),
		minZeros: minZeros,
	}
	m.scan.Split(m.splitFunc)
	m.scan.Scan()
	m.data = m.scan.Bytes()
	return m
}

/*
func Scan(r io.Reader, minZeros int64) (segments []int64, err error) {
	segments = make([]int64, 1)
	z := newMaker(r, minZeros)
	var ofs int64
	for {
		_, err = z.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		segments = append(segments, ofs+int64(len(z.data)))
		ofs += int64(len(z.data))
		segments = append(segments, ofs+z.zeros)
		ofs += z.zeros
	}
	if err != nil {
		return nil, err
	}
	return segments, nil
}
*/
