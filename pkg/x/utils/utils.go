package utils

import (
	"bytes"
	"io"
)

func IsBinary(r io.Reader, size int64) (bool, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return false, err
	}
	buf = buf[:n]
	for _, b := range buf {
		if b == 0 {
			return true, nil
		}
	}
	return false, nil
}

var _ = bytes.MinRead
