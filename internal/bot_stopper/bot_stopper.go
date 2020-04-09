package botstopper

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	answerExpiry      = 10 * time.Minute
	letters           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	challengeIDLength = 10
	maxSavedAnswers   = 1000
)

// Challenge is sent to the front-end for the user to answer it to verify if they are a bot
type Challenge struct {
	// ID identifies the answer of this challenge when verifying
	ID string

	// Question is what the user gets to see
	Question string
}

// Response is used to verify if a user is a bot
type Response struct {
	// ID identifies the answer of this challenge
	ID string

	// Answer is the actual answer a user would type on the form
	Answer string
}

type answer struct {
	expiry time.Time
	value  string
}

// BotStopper implements custom anti bot flooding
type BotStopper struct {
	mutex sync.Mutex

	// answers are identified by challenge ID
	answers map[string]answer
}

// NewBotStopper returns an initialized Botstopper
func NewBotStopper() *BotStopper {
	return &BotStopper{
		answers: make(map[string]answer),
	}
}

// GetChallenge generates a new challenge
func (bs *BotStopper) GetChallenge() Challenge {

	answer := &answer{expiry: time.Now().Add(answerExpiry)}
	var challenge Challenge

	a := 1 + rand.Intn(9)
	b := 1 + rand.Intn(9)

	switch rand.Intn(4) {
	case 0:
		challenge.Question = fmt.Sprintf("%d+%d", a, b)
		answer.value = fmt.Sprintf("%d", a+b)
	case 1:
		if a > b {
			a, b = b, a
		}
		challenge.Question = fmt.Sprintf("%d-%d", a, b)
		answer.value = fmt.Sprintf("%d", a-b)
	case 2:
		challenge.Question = fmt.Sprintf("%d*%d", a, b)
		answer.value = fmt.Sprintf("%d", a*b)
	case 3:
		challenge.Question = fmt.Sprintf("%d/%d", a*b, b)
		answer.value = fmt.Sprintf("%d", a)
	}

	challenge.ID = bs.saveAnswer(answer)
	return challenge
}

// saveAnswer saves an answer and returns its ID for future verification
func (bs *BotStopper) saveAnswer(a *answer) string {

	ID := make([]byte, challengeIDLength)
	for i := range ID {
		ID[i] = letters[rand.Intn(len(letters))]
	}

	IDString := string(ID)

	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if len(bs.answers) >= maxSavedAnswers {
		// we hit the map size cap, remove expired answers
		now := time.Now()
		for key := range bs.answers {
			if now.After(bs.answers[key].expiry) {
				delete(bs.answers, key)
			}
		}

		// pruning old answers didn't help, so we prune everything
		// when this happens probably we are flooded by bots, so it's ok
		if len(bs.answers) >= maxSavedAnswers {
			bs.answers = make(map[string]answer, maxSavedAnswers)
		}
	}

	// ignore very rare case that IDstring was already used
	bs.answers[IDString] = *a

	return IDString
}

// Verify actually checks if a user entered the right answer for a challenge
func (bs *BotStopper) Verify(response *Response) bool {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	savedAnswer, ok := bs.answers[response.ID]
	if !ok {
		return false
	}

	delete(bs.answers, response.ID)

	return time.Now().Before(savedAnswer.expiry) && response.Answer == savedAnswer.value
}
