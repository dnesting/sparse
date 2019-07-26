package sparse

import (
	"io"
)

// ReadSeeker implements io.ReadSeeker and io.ReaderAt from the given sparse
// ReadFinder and a fallback io.ReaderAt to fill in the gaps. If Fallback is
// nil, Zero is used. Attempts to read beyond the farthest write will result in
// io.EOF.
type ReadSeeker struct {
	src      ReadFinder
	Fallback io.ReaderAt
	filePos  int64
}

// NewReadSeeker creates a ReadSeeker from the sparse rd, falling back to
// fallback for reads outside of the regions covered by rd.
func NewReadSeeker(rd ReadFinder, fallback io.ReaderAt) *ReadSeeker {
	return &ReadSeeker{src: rd, Fallback: fallback}
}

// Read reads from the current file position. Attempts to read beyond the final
// segment will result in io.EOF. Reads within the gaps between segments will
// read instead from the fallback reader.
func (b *ReadSeeker) Read(p []byte) (n int, err error) {
	if b == nil {
		return 0, io.EOF
	}
	n, err = b.ReadAt(p, b.filePos)
	b.filePos += int64(n)
	return
}

// Seek seeks the file position to ofs, relative to whence.  The file position
// may be set beyond the size of the Buffer and writes will create a new segment
// at that location.  Seek supports these values for whence:
//
//   io.SeekStart     seek relative to the start of the buffer
//   io.SeekCurrent   seek relative to the current position of the buffer
//   io.SeekEnd       seek relative to the end of the buffer
//   SeekData         seek relative to the start for data bytes
//   SeekHole         seek relative to the start for a gap between data (or EOF)
func (b *ReadSeeker) Seek(ofs int64, whence int) (n int64, err error) {
	b.filePos, err = resolveSeek(ofs, whence, b.filePos, b.src.Size(), b.src)
	n = b.filePos
	return
}

func (b *ReadSeeker) fallbackOrZero() io.ReaderAt {
	if b.Fallback != nil {
		return b.Fallback
	}
	return Zero
}

// ReadAt reads from the buffer at ofs into p.  Attempts to read beyond the final segment will
// result in io.EOF.  Attempts to read gaps between segment will be filled using the fallback
// reader.  This method does not affect the file position used by Read, but may not
// preserve the file position of the underlying ReadFinder implementation.
func (b *ReadSeeker) ReadAt(p []byte, ofs int64) (n int, err error) {
	for n < len(p) {
		var nn int
		var dataOfs int64
		dataOfs, _, err = b.src.Find(ofs)
		if err != nil {
			break
		}

		if ofs < dataOfs {
			// We're in a gap until dataOfs
			fb := b.fallbackOrZero()
			limit := len(p)
			if int64(limit) > int64(n)+dataOfs-ofs {
				limit = int(int64(n) + dataOfs - ofs)
			}
			nn, err = fb.ReadAt(p[n:limit], ofs)
			n += nn
			ofs += int64(nn)
			if n == 0 || (err != nil && err != io.EOF) {
				break
			}
		}

		if ofs >= dataOfs {
			nn, err = b.src.Read(p[n:])
			n += nn
			ofs += int64(nn)
			if err == io.EOF {
				err = nil
				continue
			}
		}
		if err != nil {
			break
		}

	}
	return
}

type zeroType struct{}

// Zero implements Read and ReadAt that read nothing but zeros at every location.  It is the default
// fallback reader for NewReader and NewReadSeeker.
var Zero zeroType

func (z zeroType) ReadAt(p []byte, _ int64) (n int, err error) {
	return z.Read(p)
}

func (zeroType) Read(p []byte) (n int, err error) {
	for n < len(p) {
		p[n] = 0
		n++
	}
	return
}

type streamReader struct {
	src      Reader
	fallback io.Reader

	zeros int64
	err   error
}

// NewReader returns an io.Reader that reads from r (a sparse Reader) and fills
// in the gaps from fallback.  If fallback is nil, reads from Zero.  Reads from
// fallback will be continuous.  An attempt to read beyond the farthest segment
// of data in r will return io.EOF. If fallback returns io.EOF before r,
// behavior is undefined.
func NewReader(r Reader, fallback io.Reader) io.Reader {
	if fallback == nil {
		fallback = Zero
	}
	return &streamReader{src: r, fallback: fallback}
}

// Read reads into p the next available bytes from p.  If the file pointer lies
// within a gap, bytes will be read from the fallback reader instead.
func (ir *streamReader) Read(p []byte) (n int, err error) {
	if ir.err != nil {
		return 0, ir.err
	}
	for n < len(p) && err == nil {
		var nn int
		for n < len(p) && ir.zeros > 0 {
			want := int64(n) + ir.zeros
			if want > int64(len(p)) {
				want = int64(len(p))
			}
			nn, err = ir.fallback.Read(p[n:want])
			ir.zeros -= int64(nn)
			n += nn
			if err != nil {
				break
			}
		}
		if n == len(p) {
			break
		}
		nn, err = ir.src.Read(p[n:])
		n += nn
		if err == io.EOF {
			ir.zeros, ir.err = ir.src.Next()
			err = ir.err
		}
	}
	if err == nil && n == 0 {
		err = io.EOF
	}
	return
}
