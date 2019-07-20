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

func resolveSeek(ofs int64, whence int, currentPos, endPos int64, seg segmenter) (nofs int64, err error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		ofs += currentPos
	case io.SeekEnd:
		ofs += endPos
	case SeekData, SeekHole:
		if seg != nil {
			segs := []int64{0, endPos}
			segs, err = seg.segments()
			if err != nil {
				return 0, err
			}
			for len(segs) > 0 {
				data, hole := segs[0], segs[1]
				if whence == SeekData && data >= ofs {
					return data, nil
				} else if whence == SeekHole && hole >= ofs && hole < segs[len(segs)-1] {
					return hole, nil
				}
				segs = segs[2:]
			}
			return 0, ErrSeekEOF
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
