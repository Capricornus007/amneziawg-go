package conceal

import "io"

func ReadUntil(r io.Reader, b []byte, delim byte) (n int, err error) {
	for n < len(b) {
		if _, err = r.Read(b[n : n+1]); err != nil {
			return n, err
		}
		if b[n] == delim {
			return n, nil
		}
		n++
	}
	return n, io.ErrShortBuffer
}
