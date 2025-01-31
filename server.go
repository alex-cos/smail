package smail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

// ServerImpl holds connection information for SMTP.
type ServerImpl struct {
	Hostname  string `toml:"host"`
	Port      int    `toml:"port"`
	Username  string `toml:"username"`
	Password  string `toml:"password"`
	EnableTLS bool   `toml:"enableTLS"`
	EnableSSL bool   `toml:"EnableSSL"`
}

// SendMail - Send the given mail.
func (thiz *ServerImpl) SendMail(mail *Mail) error {
	return thiz.sendMailSimple(mail)
}

//nolint:unused
func (thiz *ServerImpl) sendMailSimple(mail *Mail) error {
	var auth smtp.Auth

	if thiz.Username != "" {
		auth = smtp.PlainAuth("", thiz.Username, thiz.Password, thiz.Hostname)
	}
	connstr := fmt.Sprintf("%s:%d", thiz.Hostname, thiz.Port)

	raw, err := mail.Bytes()
	if err != nil {
		return err
	}

	err = smtp.SendMail(connstr, auth, mail.From, mail.To, raw)
	if err != nil {
		return err
	}
	return nil
}

//nolint:unused
func (thiz *ServerImpl) sendMailComplex(mail *Mail) error {
	var (
		auth   smtp.Auth
		client *smtp.Client
		err    error
	)
	if thiz.Username != "" {
		auth = smtp.PlainAuth("", thiz.Username, thiz.Password, thiz.Hostname)
	}
	connstr := fmt.Sprintf("%s:%d", thiz.Hostname, thiz.Port)
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         thiz.Hostname,
		ClientAuth:         tls.NoClientCert,
	}

	raw, err := mail.Bytes()
	if err != nil {
		return err
	}
	if thiz.EnableSSL {
		conn, err := tls.Dial("tcp", connstr, tlsconfig)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, thiz.Hostname)
		if err != nil {
			return err
		}
	} else {
		client, err = smtp.Dial(connstr)
		if err != nil {
			return err
		}
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok || thiz.EnableTLS {
		client.StartTLS(tlsconfig)
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(mail.From); err != nil {
		return err
	}
	for _, addr := range mail.To {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = client.Quit()
	if err != nil {
		return err
	}
	return nil
}
