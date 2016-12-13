package lexl

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type LexlReader interface {
	Read(p []byte) (n int, err error)
	ReadByte() (byte, error)
	PeekByte() (byte, error)
	ReadRune() (rune, error)
	PeekRune() (rune, error)
	Eof() bool
}

///

type lexlReader struct {
	buf   []byte
	pos   int
	size  int
	in    io.Reader
	atEof bool
}

func GetLexlReader(in io.Reader, buflen int) LexlReader {
	if lr, ok := in.(LexlReader); ok {
		return lr
	}
	return &lexlReader{
		buf: make([]byte, buflen),
		in:  in,
	}
}

func (lr *lexlReader) Eof() bool {
	if lr.size > 0 {
		return false
	}
	lr.attemptFill()
	return lr.atEof
}

func (lr *lexlReader) attemptFill() error {
	if lr.size == len(lr.buf) || lr.atEof {
		return nil
	}
	for lr.size < len(lr.buf) {
		want := len(lr.buf) - lr.size
		idx := lr.pos + lr.size
		if idx >= len(lr.buf) {
			idx -= len(lr.buf)
		}
		if len(lr.buf)-idx < want {
			want = len(lr.buf) - idx
		}
		n, err := lr.in.Read(lr.buf[lr.pos : lr.pos+want])
		lr.size += n
		if err != nil {
			if err == io.EOF {
				lr.atEof = true
				return nil
			}
			return err
		}
	}
	return nil
}

func (lr *lexlReader) Read(p []byte) (n int, err error) {
	rb := 0
	for rb < len(p) {
		if lr.size == 0 {
			if lr.atEof {
				return rb, io.EOF
			}
			err := lr.attemptFill()
			if err != nil {
				return rb, err
			}
			if lr.size == 0 && lr.atEof {
				return rb, io.EOF
			}
		}
		want := len(p) - rb
		if want > lr.size {
			want = lr.size
		}
		if want > len(lr.buf)-lr.pos {
			want = len(lr.buf) - lr.pos
		}
		copy(p[rb:rb+want], lr.buf[lr.pos:lr.pos+want])
		lr.pos += want
		lr.size -= want
		if lr.pos >= len(lr.buf) {
			lr.pos -= len(lr.buf)
		}
	}
	return rb, nil
}

func (lr *lexlReader) ReadByte() (byte, error) {
	if lr.size == 0 {
		err := lr.attemptFill()
		if err != nil {
			return 0, err
		}
	}
	if lr.size == 0 {
		return 0, io.EOF
	}
	c := lr.buf[lr.pos]
	lr.pos++
	lr.size--
	if lr.pos >= len(lr.buf) {
		lr.pos -= len(lr.buf)
	}
	return c, nil
}

func (lr *lexlReader) PeekByte() (byte, error) {
	if lr.size == 0 {
		err := lr.attemptFill()
		if err != nil {
			return 0, err
		}
	}
	if lr.size == 0 {
		return 0, io.EOF
	}
	return lr.buf[lr.pos], nil
}

func (lr *lexlReader) ScanRune(read bool) (rune, error) {
	fmt.Println("SCAN RUNE")
	if lr.size < 4 {
		fmt.Println(" < 4 fill")
		err := lr.attemptFill()
		if err != nil {
			fmt.Println("SCAN ERR")
			return 0, err
		}
		fmt.Println(" < 4 fill done")
	}
	fmt.Printf("lr.size: %d\n", lr.size)
	if lr.size == 0 {
		return 0, io.EOF
	}
	if len(lr.buf)-lr.pos < 4 {
		fmt.Println("END BUFFERING")
		nbuf := make([]byte, 4)
		nlen := 4
		if nlen > lr.size {
			nlen = lr.size
		}
		npos := lr.pos
		for i := 0; i < nlen; i++ {
			nbuf[i] = lr.buf[npos]
			npos++
			if npos >= len(lr.buf) {
				npos -= len(lr.buf)
			}
		}
		r, ns := utf8.DecodeRune(nbuf)
		if r == utf8.RuneError {
			return 0, errors.New("stream does not decode a utf-8 character")
		}
		if read {
			lr.pos += ns
			lr.size -= ns
			if lr.pos >= len(lr.buf) {
				lr.pos -= len(lr.buf)
			}
		}
		return r, nil
	}
	fmt.Println("DECODING FROM BUFFER")
	r, ns := utf8.DecodeRune(lr.buf[lr.pos:])
	if r == utf8.RuneError {
		return 0, errors.New("stream does not decode a utf-8 character")
	}
	if read {
		lr.pos += ns
		lr.size -= ns
		if lr.pos >= len(lr.buf) {
			lr.pos -= len(lr.buf)
		}
	}
	return r, nil
}

func (lr *lexlReader) ReadRune() (rune, error) {
	fmt.Println("READ RUNE")
	return lr.ScanRune(true)
}

func (lr *lexlReader) PeekRune() (rune, error) {
	fmt.Println("PEEK RUNE")
	return lr.ScanRune(false)
}
