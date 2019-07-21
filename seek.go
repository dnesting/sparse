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

type segmenter interface {
	segments() ([]int64, error)
}

/*
func resolveSeekSegment(ofs int64, whence int, endPos int64, seg segmenter) (nofs int64, err error) {
	segs := []int64{0, endPos}
	segs, err = seg.segments()
	if err != nil {
		return 0, err
	}
	var i int
	if whence == SeekData {
		i = 1
	}
	for i < len(segs) {
		start, end := segs[i], segs[i+1]
		i += 2
		if ofs < end {
			if ofs < start {
				return start, nil
			}
			return ofs, nil
		}
	}
	return 0, ErrSeekEOF
}
*/

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
			/*
				if seg, ok := fin.(segmenter); ok {
					return resolveSeekSegment(ofs, whence, endPos, seg)
				} else {
			*/
			return resolveSeekFinder(ofs, whence, endPos, fin)
			//}
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
