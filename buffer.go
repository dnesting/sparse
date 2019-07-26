package sparse

import (
	"io"
)

// segment represents data bytes held at a specific offset.
type segment struct {
	off  int64
	data []byte
}

// ReadAt for a segment reads just this segment's portion of the address
// space of its container.  ofs is relative to the container's start, not
// e.off.
func (s *segment) ReadAt(p []byte, ofs int64) (n int, err error) {
	if s == nil {
		return 0, io.EOF
	}
	lofs := ofs - s.off
	n = copy(p, s.data[int(lofs):])
	return
}

func (s *segment) contains(ofs int64) bool {
	if s == nil {
		return false
	}
	return s.off <= ofs && ofs < s.end()
}

// end returns the offset after this segment, or s.off+len(s.data).
func (s segment) end() int64 { return s.off + int64(len(s.data)) }

// Buffer provides a sparse in-memory collection of bytes.  Data may be stored
// using the io.WriteSeeker or io.WriterAt interfaces, or with StoreAt.  Data
// may be read using the sparse.Reader or sparse.FindReader interfaces.
//
// A zero-value Buffer is ready to accept writes.
type Buffer struct {
	es      []segment
	cur     *segment
	filePos int64
	trunc   int64
}

// Span identifies the left-most and right-most segments that cover the range of
// ofs:ofs+size, including segments immediately adjacent to ofs or ofs+size.
// Returns the indices into b.es for these segments, and the offsets within the
// segments outside of ofs:ofs+size.
func (b *Buffer) span(ofs, size int64) (left, right int, keepLeft, keepRight int64, ok bool) {
	// Given segments like:
	// |0123456789012345678|
	// |....AAAAA..B..CCCCC| // A=4:9 B=11:12 C=14:19
	//
	// A request for ofs=6 size=10 would cover all three segments:
	// |0123456789012345678|
	// |....AAAAA..B..CCCCC|
	//        XXXXXXXXXX     // left=0 (A) right=2 (C)
	//      LL          RRR  // keepLeft=2 keepRight=3
	//
	// A request for ofs=5 size=2 would cover the interior of the first segment:
	// |0123456789012345678|
	// |....AAAAA..B..CCCCC|
	//        XX             // left=0 (A) right=0 (A)
	//      LL  R            // keepLeft=2 keepRight=1
	//
	// A request for ofs=0 size=5 would cover:
	// |0123456789012345678|
	// |....AAAAA..B..CCCCC|
	//  XXXXX                // left=0 (A) right=0 (A)
	//       RRRR            // keepLeft=0 keepRight=4
	//
	// A request for ofs=9 size=5 would cover the adjacent segments:
	// |0123456789012345678|
	// |....AAAAA..B..CCCCC|
	//           XXXXX       // left=0 (A) right=2 (C)
	//      LLLLL     RRRRR  // keepLeft=5 keepRight=5
	end := ofs + size
	for i, e := range b.es {
		if e.end() < ofs {
			continue
		}
		if end < e.off {
			break
		}
		if !ok && ofs <= e.end() {
			left = i
			keepLeft = ofs - e.off
			if keepLeft < 0 {
				keepLeft = 0
			}
			ok = true
		}
		if ok && e.off <= end {
			right = i
			keepRight = e.end() - end
			if keepRight < 0 {
				keepRight = 0
			}
		}
	}
	return
}

// Find moves the file pointer to the first byte of data available at or after
// off.  If off does not point within a segment of data, but a segment lies
// after it, the file position will be moved to the start of the next segment.
// Returns the starting offset and size of the data segment found. Callers may
// infer that the new file position is either off, or readerOfs if
// readerOfs>off. If no data lies at or after off, returns io.EOF.
func (b *Buffer) Find(off int64) (readerOfs, size int64, err error) {
	if b.moveTo(off, true) {
		return b.cur.off, int64(len(b.cur.data)), nil
	}
	return 0, 0, io.EOF
}

// moveTo moves the file pointer to off.  If off lies between sparse data
// segments and advance is true, the file pointer is advanced to the start
// of the next sequence.  Returns true if the file pointer ends up within
// a data segment.
func (b *Buffer) moveTo(off int64, advance bool) (ok bool) {
	b.cur = nil
	for i, e := range b.es {
		if off < e.end() {
			if off < e.off && advance {
				off = e.off
			}
			b.filePos = off
			if e.contains(off) {
				b.cur = &b.es[i]
				ok = true
			}
			break
		}
	}
	return
}

// Read reads up to len(p) bytes of data found at the current file position. If
// the file position points to a gap within the sparse data, which could be at
// the start of the buffer, returns io.EOF without reading any bytes. Callers
// should call Next() or Find(off) to move ahead to the next segment of data or
// to determine whether the actual EOF was reached.
func (b *Buffer) Read(p []byte) (n int, err error) {
	if !b.cur.contains(b.filePos) {
		b.moveTo(b.filePos, false)
	}
	n, err = b.cur.ReadAt(p, b.filePos)
	b.filePos += int64(n)
	if err == io.EOF && n != 0 {
		err = nil
	}
	return
}

// Next advances to the next sparse segment of data in the file stream.  If the
// file pointer currently points within a segment of data, it will be advanced
// beyond the end of this segment to the following one.  If there is no next
// segment of data, returns io.EOF.
func (b *Buffer) Next() (skip int64, err error) {
	start := b.filePos
	for i, e := range b.es {
		if e.off > start {
			b.filePos = e.off
			b.cur = &b.es[i]
			return e.off - start, nil
		}
	}
	size := b.Size()
	if start < size {
		b.filePos = size
		b.cur = nil
		return size - start, nil
	}
	return 0, io.EOF
}

// Size returns the apparent size of the segments, which is defined to be the offset beyond the
// farthest write, which may be artificially extended or reduced by Truncate().
func (b Buffer) Size() int64 {
	if len(b.es) > 0 {
		end := b.es[len(b.es)-1].end()
		if end > b.trunc {
			return end
		}
	}
	return b.trunc
}

// Reset empties the buffer and resets the file position to 0.
func (b *Buffer) Reset() {
	b.filePos = 0
	b.es = nil
	b.cur = nil
	b.trunc = 0
}

// WriteAt stores a copy of p at offset off within the Buffer.  The new readable portion of the
// buffer will be at least off+len(p) bytes.
func (b *Buffer) WriteAt(p []byte, off int64) (n int, err error) {
	return b.writeAt(p, off, false), nil
}

// StoreAt takes ownership of p and stores it at offset off within the Buffer.  The new readable
// portion of the buffer will be at least off+len(p) bytes.  StoreAt differs from WriteAt in that
// it may be possible to avoid a copy if the write does not overlap existing data.
func (b *Buffer) StoreAt(p []byte, off int64) {
	b.writeAt(p, off, true)
}

func (b *Buffer) writeAt(p []byte, off int64, ownP bool) (n int) {
	left, right, keepLeft, keepRight, mergeNeeded := b.span(off, int64(len(p)))

	// If we're allowed to keep p, then we can save ourselves some copies if we avoid trying
	// to merge adjacent segments.  We can spot these in the return from span by the fact that
	// span will ask us to keep their entire contents.
	if mergeNeeded && ownP {
		if len(b.es[left].data) == int(keepLeft) {
			left++
			keepLeft = 0
		}
		if len(b.es[right].data) == int(keepRight) {
			right--
			keepRight = 0
		}
		if right < left {
			// The span consisted only of adjacent segments, so we can fall back to insert behavior.
			mergeNeeded = false
		}
	}

	if mergeNeeded {
		n = b.writeMerge(p, off, left, right, int(keepLeft), int(keepRight))
	} else {
		n = b.writeInsert(p, off, ownP)
	}
	b.cur = nil
	return
}

// Truncate sets the size of the buffer to ofs.  Data at and after ofs will be deleted.  After this
// call returns, Size() is guaranteed to return at least ofs.
func (b *Buffer) Truncate(ofs int64) {
	b.trunc = ofs
	var i int
	for i = range b.es {
		if b.es[i].contains(ofs) {
			end := ofs - b.es[i].off
			if end != int64(len(b.es[i].data)) {
				nd := make([]byte, end)
				copy(nd, b.es[i].data)
				b.es[i].data = nd
			}
			break
		}
	}
	if i+1 < len(b.es) {
		b.delSegments(i+1, len(b.es)-i)
	}
}

func (b *Buffer) delSegments(idx, num int) {
	copy(b.es[idx:], b.es[idx+num:])
	trunc := len(b.es) - num
	for i := trunc; i < len(b.es); i++ {
		b.es[i] = segment{} // zero the segment to free any memory
	}
	b.es = b.es[:trunc]
}

func tryGrowByReslice(s []byte, desired int) ([]byte, bool) {
	if cap(s) >= desired {
		return s[:desired], true
	}
	return s, false
}

func (b *Buffer) writeMerge(p []byte, off int64, left, right, keepLeft, keepRight int) (n int) {
	l, r := &b.es[left], &b.es[right]

	dest := l.data
	var ok bool
	if dest, ok = tryGrowByReslice(dest, keepLeft+len(p)+keepRight); !ok {
		dest = make([]byte, keepLeft+len(p)+keepRight)
		copy(dest, l.data[:keepLeft])
	}
	copy(dest[keepLeft+len(p):], r.data[len(r.data)-keepRight:])
	copy(dest[keepLeft:], p)

	if off < l.off {
		l.off = off
	}
	l.data = dest

	if left != right {
		b.delSegments(left+1, right-left)
	}
	return len(p)
}

func (b *Buffer) fit(off, size int64) (i int, ok bool) {
	for i = 0; i < len(b.es); i++ {
		e := b.es[i]
		if off+size <= e.off {
			// next segment begins after our request, so this is a good insert spot
			break
		}
		if off < e.off+int64(len(e.data)) {
			// our request starts before this segment ends, so there's some sort of overlap
			return i, false
		}
	}
	return i, true
}

func (b *Buffer) writeInsert(p []byte, off int64, ownP bool) (n int) {
	insert, ok := b.fit(off, int64(len(p)))
	if !ok {
		panic("insert would overlap another segment")
	}

	buf := p
	n = len(p)
	if !ownP {
		buf = make([]byte, len(p))
		copy(buf, p)
	}

	b.es = append(b.es, segment{})
	copy(b.es[insert+1:], b.es[insert:])
	b.es[insert] = segment{off, buf}
	return
}

// Seek seeks the file position to ofs, relative to whence.  Seek is permitted
// to any positive file position.  Seek supports these values for whence:
//
//   io.SeekStart     seek relative to the start of the buffer
//   io.SeekCurrent   seek relative to the current position of the buffer
//   io.SeekEnd       seek relative to the end of the buffer
//   SeekData         seek relative to the start for data bytes >= ofs
//   SeekHole         seek relative to the start for a gap between data >= ofs (or EOF)
func (b *Buffer) Seek(ofs int64, whence int) (n int64, err error) {
	b.filePos, err = resolveSeek(ofs, whence, b.filePos, b.Size(), b)
	b.cur = nil
	return b.filePos, err
}

// Write writes p at the current file position.  Writes can occur at any file
// position.
func (b *Buffer) Write(p []byte) (n int, err error) {
	n, err = b.WriteAt(p, b.filePos)
	b.filePos += int64(n)
	return
}
