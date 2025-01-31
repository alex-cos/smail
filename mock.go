package smail

import (
	"errors"

	"github.com/stretchr/testify/mock"
)

// MockServer - A struct to Mock Mail Server.
type MockServer struct {
	mock.Mock
}

func NewMockServer() Server {
	m := &MockServer{}

	m.On("SendMail", mock.Anything).Return(nil)

	return m
}

func (thiz *MockServer) SendMail(mail *Mail) error {
	args := thiz.Called(mail)
	if mail.From == "" {
		return errors.New("missing From")
	}
	if len(mail.To) == 0 {
		return errors.New("missing To")
	}

	return args.Error(0)
}
