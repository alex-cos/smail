package smail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Priority - Priority Enum.
type Priority int

const (
	// MaxLineLength is the maximum line length per RFC 2045.
	MaxLineLength = 76
	// defaultContentType is the default Content-Type according to RFC 2045, section 5.2.
	defaultContentType = "text/plain; charset=UTF-8"
)

// Priority - Priority Constantes.
const (
	Default Priority = 0
	Highest Priority = 1
	High    Priority = 2
	Normal  Priority = 3
	Low     Priority = 4
	Lowest  Priority = 5
)

// Mail structure.
type Mail struct {
	From        string   `toml:"from"`
	To          []string `toml:"to"`
	CC          []string `toml:"cc"`
	BCC         []string `toml:"bcc"`
	Subject     string   `toml:"subject"`
	ContentType string   `toml:"content_type"`
	Priority    Priority `toml:"priority"`
	Body        string   `toml:"body"`
	Attachments []*Attachment
}

// Attachment is a struct representing an email attachment.
// Based on the mime/multipart.FileHeader struct, Attachment contains the name,
// MIMEHeader, and content of the attachment in question.
type Attachment struct {
	Filename string
	Header   textproto.MIMEHeader
	Content  []byte
	IsInline bool
}

// Bytes converts the Email object to a []byte representation, including all needed MIMEHeaders,
// boundaries, etc.
func (thiz *Mail) Bytes() ([]byte, error) {
	isMixed := len(thiz.Attachments) > 0

	buff := bytes.NewBuffer(make([]byte, 0, 4096))
	var writer *multipart.Writer
	if isMixed {
		writer = multipart.NewWriter(buff)
	}

	msg := thiz.makeHeader(isMixed, writer)
	_, err := io.WriteString(buff, msg)
	if err != nil {
		return nil, fmt.Errorf("IO write error: %w", err)
	}
	if len(thiz.Body) > 0 {
		mediaType := defaultContentType
		if len(thiz.ContentType) > 0 {
			mediaType = thiz.ContentType
		}
		if isMixed && writer != nil {
			header := textproto.MIMEHeader{
				"Content-Type":              {mediaType},
				"Content-Transfer-Encoding": {"quoted-printable"},
			}
			if _, err := writer.CreatePart(header); err != nil {
				return nil, fmt.Errorf("failed to create multipart section: %w", err)
			}
		}
		msg = thiz.Body + "\r\n"
		_, err := io.WriteString(buff, msg)
		if err != nil {
			return nil, fmt.Errorf("IO write error: %w", err)
		}
	}
	if writer != nil {
		if isMixed && len(thiz.Attachments) > 0 {
			// Create attachment part, if necessary
			for _, attach := range thiz.Attachments {
				ap, err := writer.CreatePart(attach.Header)
				if err != nil {
					return nil, fmt.Errorf("failed to create multipart section: %w", err)
				}
				// Write the base64Wrapped content to the part
				base64Wrap(ap, attach.Content)
			}
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("IO close error: %w", err)
		}
	}
	return buff.Bytes(), nil
}

// AttachFile is used to attach content to the email.
// It attempts to open the file referenced by filename and, if successful, creates an Attachment.
// This Attachment is then appended to the slice of Email.Attachments.
// The function will then return the Attachment for reference, as well as nil for the error, if successful.
func (thiz *Mail) AttachFile(filename string) (*Attachment, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open file '%s': %w", filename, err)
	}
	defer file.Close()

	ct := mime.TypeByExtension(filepath.Ext(filename))
	basename := filepath.Base(filename)
	return thiz.Attach(file, basename, ct, false)
}

// Attach is used to attach content from an io.Reader to the email.
// Required parameters include an io.Reader, the desired filename for the attachment, and the Content-Type
// The function will return the created Attachment for reference, as well as nil for the error, if successful.
func (thiz *Mail) Attach(r io.Reader, filename, contentType string, inline bool) (*Attachment, error) {
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, r); err != nil {
		return nil, fmt.Errorf("IO copy error: %w", err)
	}
	return thiz.AttachBytes(buffer.Bytes(), filename, contentType, inline), nil
}

func (thiz *Mail) AttachBytes(content []byte, filename, contentType string, inline bool) *Attachment {
	attach := &Attachment{
		Filename: filename,
		Header:   textproto.MIMEHeader{},
		Content:  content,
		IsInline: inline,
	}
	// Get the Content-Type to be used in the MIMEHeader
	if contentType != "" {
		attach.Header.Set("Content-Type", fmt.Sprintf("%s;\r\n name=\"%s\"", contentType, filename))
	} else {
		// If the Content-Type is blank, set the Content-Type to "application/octet-stream"
		attach.Header.Set("Content-Type", "application/octet-stream")
	}
	if inline {
		attach.Header.Set("Content-Disposition", fmt.Sprintf("inline;\r\n filename=\"%s\"", filename))
	} else {
		attach.Header.Set("Content-Disposition", fmt.Sprintf("attachment;\r\n filename=\"%s\"", filename))
	}
	attach.Header.Set("Content-ID", fmt.Sprintf("<%s>", filename))
	attach.Header.Set("X-Attachment-Id", filename)
	attach.Header.Set("Content-Transfer-Encoding", "base64")
	thiz.Attachments = append(thiz.Attachments, attach)
	return attach
}

func (thiz *Mail) makeHeader(isMixed bool, writer *multipart.Writer) string {
	now := time.Now()

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\n", thiz.From, strings.Join(thiz.To, ","))
	if len(thiz.CC) > 0 {
		msg += fmt.Sprintf("Cc: %s\r\n", strings.Join(thiz.CC, ","))
	}
	if len(thiz.BCC) > 0 {
		msg += fmt.Sprintf("Bcc: %s\r\n", strings.Join(thiz.BCC, ","))
	}
	msg += fmt.Sprintf("Date: %s\r\n", now.UTC().Format(time.RFC1123))
	if isMixed {
		msg += fmt.Sprintf("Content-Type: %s\r\n", "multipart/related;\r\n boundary="+writer.Boundary())
	} else {
		if len(thiz.ContentType) > 0 {
			msg += fmt.Sprintf("Content-Type: %s\r\n", thiz.ContentType)
		} else {
			msg += fmt.Sprintf("Content-Type: %s\r\n", defaultContentType)
		}
	}
	msg += fmt.Sprintf("MIME-Version: %s\r\n", "1.0")
	msg += fmt.Sprintf("User-Agent: %s\r\n", "Test System")
	if thiz.Priority > 0 {
		msg += fmt.Sprintf("X-Priority: %d\r\n", thiz.Priority)
	}
	msg += fmt.Sprintf("Subject: %s\r\n", thiz.Subject)
	msg += "\r\n"
	return msg
}

// base64Wrap encodes the attachment content, and wraps it according to RFC 2045 standards (every 76 chars)
// The output is then written to the specified io.Writer.
func base64Wrap(writer io.Writer, b []byte) {
	// 57 raw bytes per 76-byte base64 line.
	const maxRaw = 57
	// Buffer for each line, including trailing CRLF.
	buffer := make([]byte, MaxLineLength+len("\r\n"))
	copy(buffer[MaxLineLength:], "\r\n")
	// Process raw chunks until there's no longer enough to fill a line.
	for len(b) >= maxRaw {
		base64.StdEncoding.Encode(buffer, b[:maxRaw])
		writer.Write(buffer) //nolint:errcheck
		b = b[maxRaw:]
	}
	// Handle the last chunk of bytes.
	if len(b) > 0 {
		out := buffer[:base64.StdEncoding.EncodedLen(len(b))]
		base64.StdEncoding.Encode(out, b)
		out = append(out, "\r\n"...)
		writer.Write(out) //nolint:errcheck
	}
}
