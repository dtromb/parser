package ngen

import (
	"io"
)

type stringReader struct {
	str string
	pos int
}

func NewStringReader(str string) *stringReader {
	return &stringReader{str: str}
}

func (sr *stringReader) Read(p []byte) (n int, err error) {
	if len(p) > len(sr.str)-sr.pos {
		n = len(sr.str) - sr.pos
	} else {
		n = len(p)
	}
	if n == 0 {
		return 0, io.EOF
	}
	copy(p, []byte(sr.str)[sr.pos:sr.pos+n])
	sr.pos += n
	return n, nil
}
