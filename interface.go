package smail

// Server abstract Server interface.
type Server interface {
	SendMail(mail *Mail) error
}
