package internal

import (
	"github.com/stretchr/testify/mock"
	"gopkg.in/romanyx/recaptcha.v1"
)

// CaptchaVerifier checks if a request passes the gRecaptcha check
type CaptchaVerifier interface {
	Verify(gRecaptchaResponse string) (*recaptcha.Response, error)
}

var _ CaptchaVerifier = (*MockCaptchaVerifier)(nil)
var _ CaptchaVerifier = (*recaptcha.Client)(nil)

// MockCaptchaVerifier allows for mocking captcha responses
type MockCaptchaVerifier struct {
	mock.Mock
}

// Verify mocks recaptcha verification
func (verifier *MockCaptchaVerifier) Verify(gRecaptchaResponse string) (*recaptcha.Response, error) {
	args := verifier.Called(gRecaptchaResponse)
	return args.Get(0).(*recaptcha.Response), args.Error(1)
}
