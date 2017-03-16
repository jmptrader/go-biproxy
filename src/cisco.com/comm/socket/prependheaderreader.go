package socket

import "io"

// An io.Reader that prepends the value of Header.ToBytes() to its output
type PrependHeaderReader struct {
	Header      Header
	InnerReader io.Reader
	read        int
}

func (r *PrependHeaderReader) Read(d []byte) (int, error) {
	var n int
	var err error

	if r.read < HEADER_LEN {
		hbytes := r.Header.ToBytes()
		n = copy(d, hbytes[r.read:])
	} else {
		n, err = r.InnerReader.Read(d)
	}

	r.read += n
	return n, err
}
