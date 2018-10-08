// Package sparse contains types for treating small, sparse amounts of data as
// though it were a lot of data.
//
// Reader is used by types advancing forward through sparse data. The Read()
// method is used to read bytes within a segment of data, and Next() will
// advance ahead to the next segment.
//
// Finder is used by types representing addressable sparse data, where callers
// can locate segments of data and read them in a random access fashion.  It
// implements Find, which is the sparse equivalent of Seek.
//
// Buffer is a concrete type implementing Reader, Finder, io.WriterAt and
// io.WriteSeeker.  It can be used similarly to bytes.Buffer but does not
// directly implement io.Reader.
package sparse

import (
	"io"
)

// Reader is implemented by types wanting to stream sparse data.  When the
// stream position reaches a gap in the data (which could be at the start of
// the stream), Read returns io.EOF. Callers can use Next() to advance the
// stream position to the next segment of data, and can then call Read to
// retrieve bytes at that position. Segments may be zero-length. That is, Read
// may return io.EOF even after a successful call to Next. The true end of the
// sparse stream is reached when Next returns io.EOF.
type Reader interface {
	io.Reader
	Next() (skip int64, err error)
}

// Finder allows the discovery and reading of sparse data.  Use Find to locate
// sparse data at or after an offset, and Size() to determine the maximum extent
// of sparse data.
type Finder interface {
	// Find moves the file position to the sparse data at or after ofs.  This may
	// be within a segment of sparse data, or it may be the start of sparse data
	// after ofs.  Returns the absolute offset of the start of this segment of
	// sparse data, and the size of the segment.  If there is no more data in the
	// file, returns io.EOF.  If the type also implements Seek, callers should
	// assume both use the same file position, unless documented otherwise.
	Find(ofs int64) (readerOfs int64, size int64, err error)

	// Size returns the total apparent size of this data.
	Size() int64
}

// ReadFinder allows reading and discovery of sparse data.  It is is the
// sparse-friendly equivalent of ReadSeeker.
type ReadFinder interface {
	io.Reader
	Finder
}
