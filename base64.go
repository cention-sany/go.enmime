package enmime

import (
	"encoding/base64"
	"io"

	//third parties
	"github.com/glycerine/rbuf"
)

// Base64Cleaner helps work around bugs in Go's built-in base64 decoder by stripping out
// whitespace that would cause Go to lose count of things and issue an "illegal base64 data at
// input byte..." error
type Base64Cleaner struct {
	in  io.Reader
	buf [1024]byte
	//count int64
}

// NewBase64Cleaner returns a Base64Cleaner object for the specified reader.  Base64Cleaner
// implements the io.Reader interface.
func NewBase64Cleaner(r io.Reader) *Base64Cleaner {
	return &Base64Cleaner{in: r}
}

// Read method for io.Reader interface.
func (qp *Base64Cleaner) Read(p []byte) (n int, err error) {
	// Size our slice to theirs
	size := len(qp.buf)
	if len(p) < size {
		size = len(p)
	}
	buf := qp.buf[:size]
	bn, err := qp.in.Read(buf)
	for i := 0; i < bn; i++ {
		switch buf[i] {
		case ' ', '\t', '\r', '\n', '!', '.', '\x00':
			// Strip these
		default:
			p[n] = buf[i]
			n++
		}
	}
	// Count may be useful if I need to pad to even quads
	//qp.count += int64(n)
	return n, err
}

const paddingBufferSize = 1024

// Base64PadReader helps read io.Reader and stop at the last padding byte
type base64PadReader struct {
	foundEqual, eof bool
	err             error
	in              io.Reader
	rb              *rbuf.FixedSizeRingBuf
}

// BASE64 rfc2045 specified that:
// All line breaks or other characters not found in Table 1 must be ignored by
// decoding software. Other white space probably indicate a transmission error,
// about which a warning message or even a message rejection might be
// appropriate under some circumstances.
func newBase64PadReader(r io.Reader) *base64PadReader {
	return &base64PadReader{
		in: r,
		rb: rbuf.NewFixedSizeRingBuf(paddingBufferSize),
	}
}

// Read method for io.Reader interface.
func (b *base64PadReader) Read(p []byte) (n int, err error) {
	var buf [paddingBufferSize]byte
	var pbuf []byte
	var size int

	if b.err != nil {
		return 0, b.err
	}
	if !b.eof {
		// Size our slice to theirs
		size = b.rb.N - b.rb.Readable
		if len(p) < size {
			size = len(p)
		}
		pbuf = buf[:size]
		readn, err := b.in.Read(pbuf)
		if err != nil {
			if err == io.EOF {
				b.eof = true
			} else {
				//log.Println("error in read io buffer:", err)
				b.err = err
				return 0, b.err
			}
		}
		pbuf = pbuf[:readn]
		if _, err := b.rb.Write(pbuf); err != nil {
			if err != nil {
				//log.Println("error in ring buffer write:", err)
				b.err = err
				return 0, b.err
			}
		}
	}
	size = b.rb.Readable
	if size == 0 {
		return 0, io.EOF
	}
	if len(p) < size {
		size = len(p)
	}
	pbuf = buf[:size]
	peekn, err := b.rb.ReadWithoutAdvance(pbuf)
	if err != nil {
		//log.Println("error in read ring buffer:", err)
		b.err = err
		return 0, b.err
	}
	pbuf = pbuf[:peekn]
	for i := 0; i < peekn; i++ {
		if pbuf[i] == '=' {
			b.foundEqual = true
		} else if b.foundEqual {
			b.foundEqual = false
			err = io.EOF
			b.err = err
			break
		}
		p[n] = pbuf[i]
		n++
	}
	b.rb.Advance(n)
	return n, b.err
}

// nextPadding reset the internal state so the next padding can be searched.
// Return true meaning it can continue. False mean no more data to be
// searched and should be terminated.
func (b *base64PadReader) nextPadding() bool {
	if b.eof && b.rb.Readable == 0 {
		return false
	}
	b.foundEqual = false
	b.err = nil
	return true
}

// Base64Combiner help to work around bug where split base64-ed data by line
// break can cause "illegal base64 data at input byte..." error when the
// base64-ed data has padding inside it.
type Base64Combiner struct {
	pad     *base64PadReader
	cleaner *Base64Cleaner
	decoder io.Reader
	buf     [1024]byte
}

// NewBase64Combiner get data from base64-ed source and produce the
// original data from it no matter how the base64-ed source splited
// by line break or carriage return.
func NewBase64Combiner(r io.Reader) *Base64Combiner {
	padReader := newBase64PadReader(r)
	c := NewBase64Cleaner(padReader)
	return &Base64Combiner{
		pad:     padReader,
		cleaner: c,
		decoder: base64.NewDecoder(base64.StdEncoding, c),
	}
}

// Read method for io.Reader interface.
func (b *Base64Combiner) Read(p []byte) (int, error) {
	// Size our slice to theirs
	size := len(b.buf)
	if len(p) < size {
		size = len(p)
	}
	buf := b.buf[:size]
	bn, err := b.decoder.Read(buf)
	copy(p, buf[:bn])
	if err == io.EOF {
		if b.pad.nextPadding() {
			b.decoder = base64.NewDecoder(base64.StdEncoding, b.cleaner)
			err = nil
		}
	}
	return bn, err
}
