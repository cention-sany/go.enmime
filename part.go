package enmime

import (
	"bufio"
	"bytes"
	"io"
	//"mime/multipart"
	//"mime/quotedprintable"
	//"net/textproto"
	"strings"

	"github.com/cention-sany/mime"
	"github.com/cention-sany/mime/multipart"
	"github.com/cention-sany/mime/quotedprintable"
	"github.com/cention-sany/net/textproto"
	"github.com/cention-sany/uuencode"
	"golang.org/x/text/transform"
)

// MIMEPart is the primary interface enmine clients will use.  Each MIMEPart represents
// a node in the MIME multipart tree.  The Content-Type, Disposition and File Name are
// parsed out of the header for easier access.
//
// TODO Content should probably be a reader so that it does not need to be stored in
// memory.
type MIMEPart interface {
	Parent() MIMEPart             // Parent of this part (can be nil)
	FirstChild() MIMEPart         // First (top most) child of this part
	NextSibling() MIMEPart        // Next sibling of this part
	Header() textproto.MIMEHeader // Header as parsed by textproto package
	ContentType() string          // Content-Type header without parameters
	Disposition() string          // Content-Disposition header without parameters
	FileName() string             // File Name from disposition or type header
	Charset() string              // Content Charset
	Content() []byte              // Decoded content of this part (can be empty)
}

// memMIMEPart is an in-memory implementation of the MIMEPart interface.  It will likely
// choke on huge attachments.
type memMIMEPart struct {
	parent      MIMEPart
	firstChild  MIMEPart
	nextSibling MIMEPart
	header      textproto.MIMEHeader
	contentType string
	disposition string
	fileName    string
	charset     string
	content     []byte
}

// NewMIMEPart creates a new memMIMEPart object.  It does not update the parents FirstChild
// attribute.
func NewMIMEPart(parent MIMEPart, contentType string) *memMIMEPart {
	return &memMIMEPart{parent: parent, contentType: contentType}
}

// Parent of this part (can be nil)
func (p *memMIMEPart) Parent() MIMEPart {
	return p.parent
}

// First (top most) child of this part
func (p *memMIMEPart) FirstChild() MIMEPart {
	return p.firstChild
}

// Next sibling of this part
func (p *memMIMEPart) NextSibling() MIMEPart {
	return p.nextSibling
}

// Header as parsed by textproto package
func (p *memMIMEPart) Header() textproto.MIMEHeader {
	return p.header
}

// Content-Type header without parameters
func (p *memMIMEPart) ContentType() string {
	return p.contentType
}

// Content-Disposition header without parameters
func (p *memMIMEPart) Disposition() string {
	return p.disposition
}

// File Name from disposition or type header
func (p *memMIMEPart) FileName() string {
	return p.fileName
}

// Content charset
func (p *memMIMEPart) Charset() string {
	return p.charset
}

// Decoded content of this part (can be empty)
func (p *memMIMEPart) Content() []byte {
	return p.content
}

// ParseMIME reads a MIME document from the provided reader and parses it into
// tree of MIMEPart objects.
func ParseMIME(reader *bufio.Reader) (MIMEPart, error) {
	tr := textproto.NewReader(reader)
	header, err := tr.ReadMIMEHeader()
	if err != nil {
		if !strings.HasPrefix(err.Error(), "malformed MIME header") {
			return nil, err
		}
	}
	mediatype, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil && mime.IsOkPMTError(err) != nil {
		return nil, err
	}
	root := &memMIMEPart{header: header, contentType: mediatype}
	correctUTF8QP := true
	if strings.HasPrefix(mediatype, "multipart/") {
		boundary := params["boundary"]
		err = parseParts(root, reader, boundary, correctUTF8QP)
		if err != nil {
			return nil, err
		}
	} else {
		// Content is text or data, decode it
		content, err := decodeSection(header.Get("Content-Transfer-Encoding"),
			params["charset"], correctUTF8QP, reader)
		if err != nil {
			return nil, err
		}
		root.content = content
	}

	return root, nil
}

const default_content_type = "text/plain; charset=US-ASCII"

// parseParts recursively parses a mime multipart document.
func parseParts(parent *memMIMEPart, reader io.Reader, boundary string, correctUTF8QP bool) error {
	var (
		prevSibling *memMIMEPart
		mr          *multipart.Reader
	)
	// Loop over MIME parts
	if !correctUTF8QP {
		mr = multipart.NewReader(reader, boundary)
	} else {
		mr = multipart.NewCorrectUTF8QPReader(reader, boundary)
	}
	for {
		// mrp is golang's built in mime-part
		mrp, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				// This is a clean end-of-message signal
				break
			} else if strings.HasPrefix(err.Error(), "malformed MIME header") {
				// ignore this type of error and continue to process and valid MIME header
				//log.Println("debug: malformed MIME header - ignore it", len(mrp.Header))
			} else if strings.HasSuffix(err.Error(), "EOF") {
				//log.Println("debug: type of EOF failure:", err)
				if mrp == nil {
					//log.Println("debug: next part is empty")
					break
				}
			} else {
				return err
			}
		}
		if len(mrp.Header) == 0 {
			// // Empty header probably means the part didn't using the correct trailing "--"
			// // syntax to close its boundary.  We will let this slide if this this the
			// // last MIME part.
			// if _, err := mr.NextPart(); err != nil {
			// 	if err == io.EOF || strings.HasSuffix(err.Error(), "EOF") {
			// 		// This is what we were hoping for
			// 		break
			// 	} else {
			// 		return fmt.Errorf("Error at boundary %v: %v", boundary, err)
			// 	}
			// }
			// return fmt.Errorf("Empty header at boundary %v", boundary)

			if errEOF := mr.CheckNextPart(); errEOF != nil {
				if errEOF == io.EOF || strings.HasSuffix(errEOF.Error(), "EOF") {
					// This is what we were hoping for. And to remain the ability
					// to detect the empty MIME header caused by improper boundary
					// ending.
					break
				}
			}
			// empty header field inside mime part body should not treat as error as
			// MIME is allowed to have empty header and straight to the body content.
			mrp.Header.Add("Content-Type", default_content_type)
		}
		ctype := mrp.Header.Get("Content-Type")
		if ctype == "" {
			//return fmt.Errorf("Missing Content-Type at boundary %v", boundary)

			// can not find Content-Type header does not mean error
			//log.Println("debug: can not found content-type - use default")
			mrp.Header.Add("Content-Type", default_content_type)
			ctype = mrp.Header.Get("Content-Type")
		}
		mediatype, mparams, err := mime.ParseMediaType(ctype)
		if err != nil && mime.IsOkPMTError(err) != nil {
			//log.Println("debug: parse parts media type error")
			return err
		}

		// Insert ourselves into tree, p is enmime's mime-part
		p := NewMIMEPart(parent, mediatype)
		p.header = mrp.Header
		if prevSibling != nil {
			prevSibling.nextSibling = p
		} else {
			parent.firstChild = p
		}
		prevSibling = p

		// Figure out our disposition, filename
		disposition, dparams, err := mime.ParseMediaType(mrp.Header.Get("Content-Disposition"))
		if err == nil || mime.IsOkPMTError(err) == nil {
			// Disposition is optional
			p.disposition = disposition
			p.fileName = DecodeHeader(dparams["filename"])
		}
		if p.fileName == "" && mparams["name"] != "" {
			p.fileName = DecodeHeader(mparams["name"])
		}
		if p.fileName == "" && mparams["file"] != "" {
			p.fileName = DecodeHeader(mparams["file"])
		}
		if p.charset == "" {
			p.charset = mparams["charset"]
		}

		boundary := mparams["boundary"]
		isText := strings.HasPrefix(mediatype, "text/")
		if boundary != "" && !isText {
			// Content is another multipart
			err = parseParts(p, mrp, boundary, correctUTF8QP)
			if err != nil {
				return err
			}
		} else {
			// Content is text or data, decode it
			d := mrp.Header.Get("Content-Transfer-Encoding")
			if mediatype == "message/rfc822" {
				switch strings.ToLower(d) {
				case "7bit", "8bit", "binary":
				default:
					d = "" // force no decoding
				}
			}
			var txtCharset string
			if isText {
				txtCharset = p.charset
			}
			data, err := decodeSection(d, txtCharset, correctUTF8QP, mrp)
			if err != nil {
				return err
			}
			p.content = data
		}
	}

	return nil
}

// decodeSection attempts to decode the data from reader using the algorithm listed in
// the Content-Transfer-Encoding header, returning the raw data if it does not known
// the encoding type.
func decodeSection(encoding, txtCharset string, correctUTF8QP bool, reader io.Reader) ([]byte, error) {
	// Default is to just read input into bytes
	decoder := reader
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		if correctUTF8QP {
			txtCharset = strings.ToLower(txtCharset)
		}
		if correctUTF8QP && (txtCharset == "utf8" || txtCharset == "utf-8") {
			decoder = quotedprintable.NewUTF8Reader(reader)
		} else {
			decoder = quotedprintable.NewReader(reader)
		}
	case "base64":
		// cleaner := NewBase64Cleaner(reader)
		// decoder = base64.NewDecoder(base64.StdEncoding, cleaner)
		decoder = NewB64SoftCombiner(reader)
	case "uuencode":
		decoder = transform.NewReader(reader, uuencode.NewDecFirstOne())
	}

	// Read bytes into buffer
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(decoder)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
