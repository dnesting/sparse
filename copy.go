package sparse

import (
	"io"
)

// Copy copies the sparse data from r to appropriate locations within w.  Returns the
// number of bytes copied, excluding regions skipped.
func Copy(w io.WriteSeeker, r Reader) (n int64, err error) {
	for {
		var nn, skip int64
		nn, err = io.Copy(w, r)
		n += nn
		if err != nil {
			break
		}

		if skip, err = r.Next(); err != nil {
			break
		}
		if skip > 0 {
			if _, err = w.Seek(skip, io.SeekCurrent); err != nil {
				break
			}
		}
	}
	return
}
