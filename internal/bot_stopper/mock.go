package botstopper

import "github.com/stretchr/testify/mock"

// Interface is the external interface of the BotStopper
type Interface interface {
	GetChallenge() Challenge
	Verify(response Response) bool
}

var _ Interface = (*BotStopper)(nil)
var _ Interface = (*MockVerifier)(nil)

// MockVerifier is a struct for external testing
type MockVerifier struct {
	mock.Mock
}

// Verify mocks the check if a user is a bot
func (mv *MockVerifier) Verify(response Response) bool {
	args := mv.Called(response)
	return args.Bool(0)
}

// GetChallenge mocks returning a challenge
func (mv *MockVerifier) GetChallenge() Challenge {
	args := mv.Called()
	return args.Get(0).(Challenge)
}
