package enmime

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	//"net/mail"
	"os"
	"path/filepath"
	"testing"

	"github.com/cention-sany/net/mail"
	"github.com/stretchr/testify/assert"
)

func TestIdentifySinglePart(t *testing.T) {
	msg := readMessage("non-mime.raw")
	assert.False(t, IsMultipartMessage(msg), "Failed to identify non-multipart message")
}

func TestIdentifyMultiPart(t *testing.T) {
	msg := readMessage("html-mime-inline.raw")
	assert.True(t, IsMultipartMessage(msg), "Failed to identify multipart MIME message")
}

func TestParseNonMime(t *testing.T) {
	msg := readMessage("non-mime.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse non-MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "This is a test mailing")
	assert.Empty(t, mime.HTML, "Expected no HTML body")
}

func TestParseRussianNonMime(t *testing.T) {
	msg := readMessage("russian-non-mime.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse non-MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "Ирина  ,")
	assert.Empty(t, mime.HTML, "Expected no HTML body")
}

func TestParseNonMimeHTML(t *testing.T) {
	msg := readMessage("non-mime-html.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse non-MIME: %v", err)
	}
	assert.True(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be true")
	assert.Contains(t, mime.Text, "This is a test mailing")
	assert.Contains(t, mime.HTML, "<span>This</span>")
}

func TestParseMimeTree(t *testing.T) {
	msg := readMessage("attachment.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.NotNil(t, mime.Root, "Message should have a root node")
}

func TestParseInlineText(t *testing.T) {
	msg := readMessage("html-mime-inline.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "Test of text section", mime.Text)
}

func TestParseMultiMixedText(t *testing.T) {
	msg := readMessage("mime-mixed.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "Section one\n\n--\nSection two", mime.Text,
		"Text parts should be concatenated")
}

func TestParseMultiSignedText(t *testing.T) {
	msg := readMessage("mime-signed.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "Section one\n\n--\nSection two", mime.Text,
		"Text parts should be concatenated")
}

func TestParseQuotedPrintable(t *testing.T) {
	msg := readMessage("quoted-printable.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "Phasellus sit amet arcu")
}

func TestParseQuotedPrintableMime(t *testing.T) {
	msg := readMessage("quoted-printable-mime.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "Nullam venenatis ante")
}

func TestParseNoEndLineMime(t *testing.T) {
	msg := readMessage("no-end-line-mime.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}
	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam venenatis.", mime.Text, "Plain text is not match")
	assert.Equal(t, "<html><head></body></html>", mime.HTML, "HTML is not match")
}

func TestParseBrokenQuotedprintableUTF8(t *testing.T) {
	msg := readMessage("broken-qp-utf8.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}
	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "<o:p></o:p></p>f\xC3\r\n\x83\xC2\xB6r order", mime.Text, "Plain text is not match")
	assert.Equal(t, "<html><head>Laggon \xC3\x83 \xC2\r\n\xA4r fel</body></html>", mime.HTML, "HTML is not match")

	msg = readMessage("broken-qp-utf8.raw")
	mime, err = ParseMIMEBodyWithUTF8QPCorrection(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}
	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "<o:p></o:p></p>f\xC3\x83\xC2\xB6r order", mime.Text, "Plain text is not match")
	assert.Equal(t, "<html><head>Laggon \xC3\x83 \xC2\xA4r fel</body></html>", mime.HTML, "HTML is not match")
}

func TestParseInlineHTML(t *testing.T) {
	msg := readMessage("html-mime-inline.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Contains(t, mime.HTML, "<html>")
	assert.Contains(t, mime.HTML, "Test of HTML section")
}

func TestParseAttachment(t *testing.T) {
	msg := readMessage("attachment.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "A text section")
	assert.Equal(t, "", mime.HTML, "Html attachment is not for display")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 1, len(mime.Attachments), "Should have a single attachment")
	assert.Equal(t, "test.html", mime.Attachments[0].FileName(), "Attachment should have correct filename")
	assert.Contains(t, string(mime.Attachments[0].Content()), "<html>",
		"Attachment should have correct content")

	//for _, a := range mime.Attachments {
	//	fmt.Printf("%v %v\n", a.ContentType(), a.Disposition())
	//}
}

func TestParseUUAttachment(t *testing.T) {
	msg := readMessage("uu-attachment.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}
	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "A text section")
	assert.Equal(t, "", mime.HTML, "Html attachment is not for display")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 1, len(mime.Attachments), "Should have a single attachment")
	assert.Equal(t, "uutestname.txt", mime.Attachments[0].FileName(),
		"Attachment should has correct filename")
	assert.Contains(t, string(mime.Attachments[0].Content()),
		"I love you forever",
		"Uuencoded attachment should has correct content")
}

func TestParseAttachmentOctet(t *testing.T) {
	msg := readMessage("attachment-octet.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Contains(t, mime.Text, "A text section")
	assert.Equal(t, "", mime.HTML, "Html attachment is not for display")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 1, len(mime.Attachments), "Should have a single attachment")
	assert.Equal(t, "ATTACHMENT.EXE", mime.Attachments[0].FileName(),
		"Attachment should have correct filename")
	assert.Equal(t,
		[]byte{
			0x3, 0x17, 0xe1, 0x7e, 0xe8, 0xeb, 0xa2, 0x96, 0x9d, 0x95, 0xa7, 0x67, 0x82, 0x9,
			0xdf, 0x8e, 0xc, 0x2c, 0x6a, 0x2b, 0x9b, 0xbe, 0x79, 0xa4, 0x69, 0xd8, 0xae, 0x86,
			0xd7, 0xab, 0xa8, 0x72, 0x52, 0x15, 0xfb, 0x80, 0x8e, 0x47, 0xe1, 0xae, 0xaa, 0x5e,
			0xa2, 0xb2, 0xc0, 0x90, 0x59, 0xe3, 0x35, 0xf8, 0x60, 0xb7, 0xb1, 0x63, 0x77, 0xd7,
			0x5f, 0x92, 0x58, 0xa8, 0x75,
		}, mime.Attachments[0].Content(), "Attachment should have correct content")
}

func TestParseBase64Dot(t *testing.T) {
	msg := readMessage("base64dot.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Contains(t, mime.Text, "A text section")
	assert.Equal(t, "", mime.HTML, "Html attachment is not for display")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 1, len(mime.Attachments), "Should have a single attachment")
	assert.Equal(t, "ATTACHMENT.EXE", mime.Attachments[0].FileName(),
		"Attachment should have correct filename")
	assert.Equal(t,
		[]byte{
			0x3, 0x17, 0xe1, 0x7e, 0xe8, 0xeb, 0xa2, 0x96, 0x9d, 0x95, 0xa7, 0x67, 0x82, 0x9,
			0xdf, 0x8e, 0xc, 0x2c, 0x6a, 0x2b, 0x9b, 0xbe, 0x79, 0xa4, 0x69, 0xd8, 0xae, 0x86,
			0xd7, 0xab, 0xa8, 0x72, 0x52, 0x15, 0xfb, 0x80, 0x8e, 0x47, 0xe1, 0xae, 0xaa, 0x5e,
			0xa2, 0xb2, 0xc0, 0x90, 0x59, 0xe3, 0x35, 0xf8, 0x60, 0xb7, 0xb1, 0x63, 0x77, 0xd7,
			0x5f, 0x92, 0x58, 0xa8, 0x75,
		}, mime.Attachments[0].Content(), "Attachment should have correct content")
}

func TestParseOtherParts(t *testing.T) {
	msg := readMessage("other-parts.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Contains(t, mime.Text, "A text section")
	assert.Equal(t, "", mime.HTML, "No Html attachment available")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 0, len(mime.Attachments), "Should have no attachment")
	assert.Equal(t, 1, len(mime.OtherParts), "Should have one OtherParts")
	assert.Equal(t, "B05.gif", mime.OtherParts[0].FileName(),
		"Part should have correct filename")
	assert.Equal(t,
		[]byte{
			0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0xf, 0x0, 0xf, 0x0, 0xa2, 0x5, 0x0, 0xde, 0xeb,
			0xf3, 0x5b, 0xb0, 0xec, 0x0, 0x89, 0xe3, 0xa3, 0xd0, 0xed, 0x0, 0x46, 0x74, 0xdd,
			0xed, 0xfa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x21, 0xf9, 0x4, 0x1, 0x0, 0x0, 0x5, 0x0,
			0x2c, 0x0, 0x0, 0x0, 0x0, 0xf, 0x0, 0xf, 0x0, 0x0, 0x3, 0x40, 0x58, 0x25, 0xa4, 0x4b,
			0xb0, 0x39, 0x1, 0x46, 0xa3, 0x23, 0x5b, 0x47, 0x46, 0x68, 0x9d, 0x20, 0x6, 0x9f,
			0xd2, 0x95, 0x45, 0x44, 0x8, 0xe8, 0x29, 0x39, 0x69, 0xeb, 0xbd, 0xc, 0x41, 0x4a,
			0xae, 0x82, 0xcd, 0x1c, 0x9f, 0xce, 0xaf, 0x1f, 0xc3, 0x34, 0x18, 0xc2, 0x42, 0xb8,
			0x80, 0xf1, 0x18, 0x84, 0xc0, 0x9e, 0xd0, 0xe8, 0xf2, 0x1, 0xb5, 0x19, 0xad, 0x41,
			0x53, 0x33, 0x9b, 0x0, 0x0, 0x3b,
		}, mime.OtherParts[0].Content(), "Part should have correct content")
}

func TestParseInline(t *testing.T) {
	msg := readMessage("html-mime-inline.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Contains(t, mime.Text, "Test of text section", "Should have text section")
	assert.Contains(t, mime.HTML, ">Test of HTML section<", "Should have html section")
	assert.Equal(t, 1, len(mime.Inlines), "Should have one inline")
	assert.Equal(t, 0, len(mime.Attachments), "Should have no attachments")
	assert.Equal(t, "favicon.png", mime.Inlines[0].FileName(),
		"Inline should have correct filename")
	assert.True(t, bytes.HasPrefix(mime.Inlines[0].Content(), []byte{0x89, 'P', 'N', 'G'}),
		"Content should be PNG image")
}

func TestParseHTMLOnlyInline(t *testing.T) {
	msg := readMessage("html-only-inline.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.True(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be true")
	assert.Contains(t, mime.Text, "Test of HTML section",
		"Should have down-converted text section")
	assert.Contains(t, mime.HTML, ">Test of HTML section<", "Should have html section")
	assert.Equal(t, 1, len(mime.Inlines), "Should have one inline")
	assert.Equal(t, 0, len(mime.Attachments), "Should have no attachments")
	assert.Equal(t, "favicon.png", mime.Inlines[0].FileName(),
		"Inline should have correct filename")
	assert.True(t, bytes.HasPrefix(mime.Inlines[0].Content(), []byte{0x89, 'P', 'N', 'G'}),
		"Content should be PNG image")
}

func TestParseNestedHeaders(t *testing.T) {
	msg := readMessage("html-mime-inline.raw")
	mime, err := ParseMIMEBody(msg)

	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, 1, len(mime.Inlines), "Should have one inline")
	assert.Equal(t, "favicon.png", mime.Inlines[0].FileName(),
		"Inline should have correct filename")
	assert.Equal(t, "<8B8481A2-25CA-4886-9B5A-8EB9115DD064@skynet>",
		mime.Inlines[0].Header().Get("Content-Id"), "Inline should have a Content-Id header")
}

func TestParseEncodedSubjectAndAddress(t *testing.T) {
	// Even non-MIME messages should support encoded-words in headers
	// Also, encoded addresses should be suppored
	msg := readMessage("qp-ascii-header.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse non-MIME: %v", err)
	}
	assert.Equal(t, "Test QP Subject!", mime.GetHeader("Subject"))

	// Test UTF-8 subject line
	msg = readMessage("qp-utf8-header.raw")
	mime, err = ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}
	assert.Equal(t, "MIME UTF8 Test \u00a2 More Text", mime.GetHeader("Subject"))
	toAddresses, err := mime.AddressList("To")
	if err != nil {
		t.Fatalf("Failed to parse To list: %v", err)
	}
	assert.Equal(t, 1, len(toAddresses))
	assert.Equal(t, "Mirosław Marczak", toAddresses[0].Name)
}

// new test mail from forked version
func TestParseMultiAnyText(t *testing.T) {
	msg := readMessage("mime-any.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, "Section one\n\n--\nSection two", mime.Text,
		"Text parts should be concatenated")
}

func TestParseMimeNoHeader(t *testing.T) {
	msg := readMessage("mime-noheader.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, "first partial message\n", mime.Text, "Text parts should be parsed as plain text")
	if assert.Equal(t, 2, len(mime.OtherParts), "Should have two inlines") {
		assert.Equal(t, "second message\n", string(mime.OtherParts[0].Content()), "First attachment should has plain text")
		assert.Equal(t, "empty\n", string(mime.OtherParts[1].Content()), "Second attachment should has plain text")
	}
}

func TestParseMimeCorruptHeader(t *testing.T) {
	msg := readMessage("mime-corrupt-header.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, "first msg\n\n--\nsecond msg\n\n--\nthird msg\n", mime.Text, "Corrupted text message should be concanated as plain text")
}

func TestParseMimeCorruptHeader2(t *testing.T) {
	msg := readMessage("mime-corrupt-header2.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, "first msg\n\n--\nsecond msg\n", mime.Text, "Corrupted text message should be concanated as plain text")
	assert.Equal(t, "<b>third msg</b>\n", mime.HTML, "There should has html text")
}

func TestParseMimeMultiEmpty(t *testing.T) {
	msg := readMessage("mime-multi-empty.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Empty(t, mime.Text, "Expected empty text part")
}

func TestDetectCharacterSetInHTML(t *testing.T) {
	msg := readMessage("non-mime-missing-charset.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil {
		t.Fatalf("Failed to parse non-MIME: %v", err)
	}
	assert.False(t, strings.ContainsRune(mime.HTML, 0x80),
		"HTML body should not have contained a Windows CP1250 Euro Symbol")
	assert.True(t, strings.ContainsRune(mime.HTML, 0x20ac),
		"HTML body should have contained a Unicode Euro Symbol")
}

const otherShouldHasCorrectFilename = "Other part should have correct filename"
const otherShouldHasCorrectContent = "Other part should have correct content"

func TestAllBadMIME(t *testing.T) {
	msg := readMessage("all-bad-mime.raw")
	mime, err := ParseMIMEBody(msg)
	if err != nil && mime == nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.False(t, mime.IsTextFromHTML, "Expected text-from-HTML flag to be false")
	assert.Equal(t, "This is plain text", mime.Text, "Plain text is not match")
	assert.Equal(t, "<strong>This is html text<strong>", mime.HTML, "Html text is not match")
	assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
	assert.Equal(t, 0, len(mime.Attachments), "Should have no attachment")

	assert.Equal(t, 5, len(mime.OtherParts), "Should have one OtherParts")
	// first bad mime
	assert.Equal(t, "nomime.typ", mime.OtherParts[0].FileName(), otherShouldHasCorrectFilename)
	assert.Contains(t, string(mime.OtherParts[0].Content()), "content1",
		otherShouldHasCorrectContent)
	// second bad mime
	assert.Equal(t, "nomime2.typ", mime.OtherParts[1].FileName(), otherShouldHasCorrectFilename)
	assert.Contains(t, string(mime.OtherParts[1].Content()), "content2",
		otherShouldHasCorrectContent)
	// third bad mime
	assert.Equal(t, "slash.png", mime.OtherParts[2].FileName(), otherShouldHasCorrectFilename)
	assert.Contains(t, string(mime.OtherParts[2].Content()), "content3",
		otherShouldHasCorrectContent)
	// fourth bad mime
	assert.Equal(t, "noslash.pdf", mime.OtherParts[3].FileName(), otherShouldHasCorrectFilename)
	assert.Contains(t, string(mime.OtherParts[3].Content()), "content4",
		otherShouldHasCorrectContent)
	// fifth bad mime
	assert.Equal(t, "", mime.OtherParts[4].FileName(), "Other part should have no filename")
	assert.Contains(t, string(mime.OtherParts[4].Content()), "content5",
		otherShouldHasCorrectContent)
}

func TestLongHeader(t *testing.T) {
	msg := readMessage("broken-address-header.raw")
	addrs, err := mail.ParseAddressList(msg.Header.Get("To"))
	if err != nil {
		t.Fatalf("Failed to parse To header field: %v", err)
	} else if len(addrs) <= 0 {
		t.Fatalf("Error: To address field is zero")
	}
	expected := [...]string{
		`tester1.name@example.com`,
		`"tester2 _ name"@example.se`,
		`"tester3 s"@example.my`,
		`tester4.name@example.co.uk`,
		`tester5.name@example.com`,
		`tester6@example.org`,
		`"test e r7"@example.com`,
		`tester8@example.com`,
		`"te s ter9"@example.se`,
		`tester10@example.com.my`,
		`tester11.last@example.com`,
	}
	if len(addrs) != len(expected) {
		t.Fatalf("Error parse address length (%d) not same as expected (%d)",
			len(addrs), len(expected))
	}
	for i, a := range addrs {
		if a.Address != expected[i] {
			t.Errorf("Result #%d address %s not same as expected %s", i+1,
				a.Address, expected[i])
		}
	}
	mime, err := ParseMIMEBody(msg)
	if err != nil && mime == nil {
		t.Fatalf("Failed to parse MIME: %v", err)
	}

	assert.Equal(t, "</div>hello!</div>", mime.Text, "Plain text is not match")
	assert.Equal(t, "</div>hello!</div>", mime.HTML, "Html text is not match")
}

// readMessage is a test utility function to fetch a mail.Message object.
func readMessage(filename string) *mail.Message {
	// Open test email for parsing
	raw, err := os.Open(filepath.Join("test-data", "mail", filename))
	if err != nil {
		panic(fmt.Sprintf("Failed to open test data: %v", err))
	}

	// Parse email into a mail.Message object like we do
	reader := bufio.NewReader(raw)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		panic(fmt.Sprintf("Failed to read message: %v", err))
	}

	return msg
}
