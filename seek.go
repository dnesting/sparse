package sparse

import (
	"errors"
	"io"
)

const (
	SeekData = 3
	SeekHole = 4
)

var (
	ErrSeekEOF = errors.New("seek beyond EOF")
	errWhence  = errors.New("Seek: invalid whence")
	errOffset  = errors.New("Seek: invalid offset")
)

func resolveSeekFinder(ofs int64, whence int, endPos int64, fin Finder) (nofs int64, err error) {
	n, size, e := fin.Find(ofs)
	if e != nil {
		if e == io.EOF && whence == SeekHole {
			return ofs, nil
		}
		return 0, err
	}
	if whence == SeekData {
		if n > ofs {
			return n, nil
		}
		return ofs, nil
	}
	if ofs < n { // found data after ofs, so ofs must be inside a hole
		return ofs, nil
	}
	// found ofs to be inside a data segment, so look beyond it
	return resolveSeekFinder(n+size, whence, endPos, fin)
}

func resolveSeek(ofs int64, whence int, currentPos, endPos int64, fin Finder) (nofs int64, err error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		ofs += currentPos
	case io.SeekEnd:
		ofs += endPos
	case SeekData, SeekHole:
		if fin != nil {
			return resolveSeekFinder(ofs, whence, endPos, fin)
		}
		fallthrough
	default:
		return 0, errWhence
	}
	if ofs < 0 {
		return 0, errOffset
	}
	return ofs, nil
}
