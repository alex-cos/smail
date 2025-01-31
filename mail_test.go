package smail_test

import (
	"bytes"
	"html/template"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/alex-cos/smail"
	"github.com/stretchr/testify/assert"
)

var (
	hostname  = os.Getenv("TEST_SMTP_HOST")
	port      = os.Getenv("TEST_SMTP_PORT") // 465 (SSL required) or 587 (TLS required)
	username  = os.Getenv("TEST_SMTP_USERNAME")
	password  = os.Getenv("TEST_SMTP_PASSWORD")
	recipient = os.Getenv("TEST_SMTP_RECIPIENT")
)

func TestRawMail(t *testing.T) {
	t.Parallel()

	port, err := strconv.Atoi(port)
	assert.NoError(t, err)
	server := &smail.ServerImpl{
		Hostname:  hostname,
		Port:      port,
		Username:  username,
		Password:  password,
		EnableTLS: true,
		EnableSSL: false,
	}

	mail := &smail.Mail{
		From:        username,
		To:          []string{recipient},
		CC:          nil,
		BCC:         nil,
		ContentType: "text/plain; charset=UTF-8",
		Subject:     "Ceci est un test",
		Body:        "this is the body",
		Priority:    smail.Highest,
		Attachments: nil,
	}

	err = server.SendMail(mail)
	assert.NoError(t, err)
}

func TestHTMLMail(t *testing.T) {
	t.Parallel()

	testfile := filepath.Join("testdata", "test.html")
	imgfile := filepath.Join("testdata", "logo.jpg")

	data, err := os.ReadFile(testfile)
	assert.NoError(t, err)

	port, err := strconv.Atoi(port)
	assert.NoError(t, err)

	server := &smail.ServerImpl{
		Hostname:  hostname,
		Port:      port,
		Username:  username,
		Password:  password,
		EnableTLS: false,
		EnableSSL: false,
	}

	mail := &smail.Mail{
		From:        username,
		To:          []string{recipient},
		CC:          nil,
		BCC:         nil,
		ContentType: "text/html; charset=UTF-8",
		Subject:     "Ceci est un test",
		Body:        string(data),
		Priority:    smail.Highest,
		Attachments: nil,
	}

	f, err := os.Open(imgfile)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = mail.AttachFile(imgfile)
	assert.NoError(t, err)

	err = server.SendMail(mail)
	assert.NoError(t, err)
}

func TestTemplateMail(t *testing.T) {
	t.Parallel()

	testfile := filepath.Join("testdata", "test.html")
	imgfile := filepath.Join("testdata", "logo.jpg")

	template, err := template.ParseFiles(testfile)
	assert.NoError(t, err)

	port, err := strconv.Atoi(port)
	assert.NoError(t, err)

	server := &smail.ServerImpl{
		Hostname:  hostname,
		Port:      port,
		Username:  username,
		Password:  password,
		EnableTLS: false,
		EnableSSL: false,
	}

	var buff bytes.Buffer
	err = template.Execute(&buff, struct {
		Lastname     string
		Firstname    string
		PhoneNumber  string
		EmailAddress string
		FreeText     string
	}{
		Lastname:     "Doo",
		Firstname:    "John",
		PhoneNumber:  "001 22 33 44 55 66",
		EmailAddress: "john.doo@hello.com",
		FreeText:     "my remarks here",
	})
	assert.NoError(t, err)

	mail := &smail.Mail{
		From:        username,
		To:          []string{recipient},
		CC:          nil,
		BCC:         nil,
		ContentType: "text/html; charset=UTF-8",
		Subject:     "Ceci est un test",
		Body:        buff.String(),
		Priority:    smail.Highest,
		Attachments: nil,
	}

	file, err := os.Open(imgfile)
	if err != nil {
		return
	}
	defer file.Close()

	ct := mime.TypeByExtension(filepath.Ext(imgfile))
	basename := filepath.Base(imgfile)
	_, err = mail.Attach(file, basename, ct, true)
	assert.NoError(t, err)

	err = server.SendMail(mail)
	assert.NoError(t, err)
}
