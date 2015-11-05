/*
Rule 1: direct characters: 62 alphanumeric characters and 9 symbols: ' ( ) , - . / : ?
Rule 2: optional direct characters: all other printable characters in the range
 U+0020–U+007E except ~ \ + and space
plus sign (+) may be encoded as +- (null shift)
Plus sign (+) mean the start of 'modified Base64 encoded UTF-16'. The end of
 this block is indicated by any character not in the modified Base64.
If character after modified Base64 is a '-' then it is consumed.
 example:
 "1 + 1 = 2" is encoded as "1 +- 1 +AD0 2" //+AD0 is the '=' sign.
 "£1" is encoded as "+AKM-1" //+AKM- is the '£' sign where '-' is consumed.
A "+" character followed immediately by any character other than members
 of modified Base64 or "-" is an ill-formed sequence.
Rule 3: Space, tab, carriage return and line feed may also be represented directly
 as single ASCII bytes. Further content transfer encoding may be needed if using
 in email environment.
*/
package enmime

import (
	"encoding/base64"
	"errors"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/internal"
	"golang.org/x/text/encoding/internal/identifier"
	"golang.org/x/text/transform"
)

const (
	uRepl = '\uFFFD' // Unicode replacement code point
	u7min = 0x20     // Minimum self-representing UTF-7 value
	u7max = 0x7E     // Maximum self-representing UTF-7 value
)

var (
	UTF7 encoding.Encoding = &utf7
)

var utf7 = internal.Encoding{
	&internal.SimpleEncoding{
		utf7Decoder{ascii: true},
		utf7Encoder{},
	},
	"UTF7",
	identifier.Unicode11UTF7,
}

// ErrBadUTF7 is returned to indicate invalid modified UTF-7 encoding.
var ErrBadUTF7 = errors.New("utf7: bad utf-7 encoding")

// Base64 codec for code points outside of the 0x20-0x7E range.
const modifiedbase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var u7enc = base64.NewEncoding(modifiedbase64)

type utf7Decoder struct {
	transform.NopResetter
	ascii bool
}

func (d utf7Decoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	return nDst, nSrc, err
}

type utf7Encoder struct {
	transform.NopResetter
}

func (e utf7Encoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	return nDst, nSrc, err
}

// UTF7Encode converts a string from UTF-8 encoding to modified UTF-7. This
// encoding is used by the Mailbox International Naming Convention (RFC 3501
// section 5.1.3). Invalid UTF-8 byte sequences are replaced by the Unicode
// replacement code point (U+FFFD).
func UTF7Encode(s string) string {
	return string(UTF7EncodeBytes([]byte(s)))
}

// UTF7EncodeBytes converts a byte slice from UTF-8 encoding to modified UTF-7.
func UTF7EncodeBytes(s []byte) []byte {
	u := make([]byte, 0, len(s)*2)
	for i, n := 0, len(s); i < n; {
		if c := s[i]; u7min <= c && c <= u7max {
			i++
			if u = append(u, c); c == '&' {
				u = append(u, '-')
			}
			continue
		}
		start := i
		for i++; i < n && (s[i] < u7min || s[i] > u7max); i++ {
			// Find the next printable ASCII code point
		}
		u = append(u, utf7enc(s[start:i])...)
	}
	return u
}

// utf7enc converts string s from UTF-8 to UTF-16-BE, encodes the result as
// Base64, removes the padding, and adds UTF-7 shifts.
func utf7enc(s []byte) []byte {
	// len(s) is sufficient for UTF-8 to UTF-16 conversion if there are no
	// control code points (see table below).
	b := make([]byte, 0, len(s)+4)
	for len(s) > 0 {
		r, size := utf8.DecodeRune(s)
		if r > utf8.MaxRune {
			r, size = utf8.RuneError, 1 // Bug fix (issue 3785)
		}
		s = s[size:]
		if r1, r2 := utf16.EncodeRune(r); r1 != uRepl {
			b = append(b, byte(r1>>8), byte(r1))
			r = r2
		}
		b = append(b, byte(r>>8), byte(r))
	}

	// Encode as Base64
	n := u7enc.EncodedLen(len(b)) + 2
	b64 := make([]byte, n)
	u7enc.Encode(b64[1:], b)

	// Strip padding
	n -= 2 - (len(b)+2)%3
	b64 = b64[:n]

	// Add UTF-7 shifts
	b64[0] = '&'
	b64[n-1] = '-'
	return b64
}

// UTF7Decode converts a string from modified UTF-7 encoding to UTF-8.
func UTF7Decode(u string) (s string, err error) {
	b, err := UTF7DecodeBytes([]byte(u))
	s = string(b)
	return
}

func isModifiedBase64(r byte) bool {
	bs := []byte(modifiedbase64)
	for _, b := range bs {
		if b == r {
			return true
		}
	}
	return false
}

// UTF7DecodeBytes converts a byte slice from modified UTF-7 encoding to UTF-8.
func UTF7DecodeBytes(u []byte) (s []byte, err error) {
	s = make([]byte, 0, len(u))
	//ascii := true
	padding := false
	for i, n := 0, len(u); i < n; i++ {
		if c := u[i]; ((c < u7min || c > u7max) &&
			c != '\t' && c != '\r' && c != '\n') ||
			c == '~' || c == '\\' {
			return nil, ErrBadUTF7 // Illegal code point in ASCII mode
		} else if c != '+' {
			s = append(s, c)
			//ascii = true
			continue
		}
		start := i + 1
		// Find the end of the Base64 or "+-" segment
		padding = false
		for i++; i < n && u[i] != '-'; i++ {
			if !isModifiedBase64(u[i]) {
				if i == start {
					return nil, ErrBadUTF7 // '+' next char must modified base64
				} else if u[i] == '=' {
					padding = true
				}
			} else {
				if padding {
					return nil, ErrBadUTF7 // base64 padding at the end only
				}
			}
		}
		if i == start {
			s = append(s, '+') // Escape sequence "+-"
			//ascii = true
		} else if b := utf7dec(u[start:i]); len(b) > 0 {
			s = append(s, b...) // Control or non-ASCII code points in Base64
			//ascii = false
		} else {
			return nil, ErrBadUTF7 // Null shift ("+...-+...-") or bad encoding
		}
	}
	return
}

// utf7dec extracts UTF-16-BE bytes from Base64 data and converts them to UTF-8.
// A nil slice is returned if the encoding is invalid.
func utf7dec(b64 []byte) []byte {
	var b []byte

	// Allocate a single block of memory large enough to store the Base64 data
	// (if padding is required), UTF-16-BE bytes, and decoded UTF-8 bytes.
	// Since a 2-byte UTF-16 sequence may expand into a 3-byte UTF-8 sequence,
	// double the space allocation for UTF-8.
	if n := len(b64); b64[n-1] == '=' {
		return nil
	} else if n&3 == 0 {
		b = make([]byte, u7enc.DecodedLen(n)*3)
	} else {
		n += 4 - n&3
		b = make([]byte, n+u7enc.DecodedLen(n)*3)
		copy(b[copy(b, b64):n], []byte("=="))
		b64, b = b[:n], b[n:]
	}

	// Decode Base64 into the first 1/3rd of b
	n, err := u7enc.Decode(b, b64)
	if err != nil || n&1 == 1 {
		return nil
	}

	// Decode UTF-16-BE into the remaining 2/3rds of b
	b, s := b[:n], b[n:]
	j := 0
	for i := 0; i < n; i += 2 {
		r := rune(b[i])<<8 | rune(b[i+1])
		if utf16.IsSurrogate(r) {
			if i += 2; i == n {
				return nil
			}
			r2 := rune(b[i])<<8 | rune(b[i+1])
			if r = utf16.DecodeRune(r, r2); r == uRepl {
				return nil
			}
		} else if u7min <= r && r <= u7max {
			return nil
		}
		j += utf8.EncodeRune(s[j:], r)
	}
	return s[:j]
}

/*
The following table shows the number of bytes required to encode each code point
in the specified range using UTF-8 and UTF-16 representations:

+-----------------+-------+--------+
| Code points     | UTF-8 | UTF-16 |
+-----------------+-------+--------+
| 000000 - 00007F |   1   |   2    |
| 000080 - 0007FF |   2   |   2    |
| 000800 - 00FFFF |   3   |   2    |
| 010000 - 10FFFF |   4   |   4    |
+-----------------+-------+--------+

Source: http://en.wikipedia.org/wiki/Comparison_of_Unicode_encodings
*/
