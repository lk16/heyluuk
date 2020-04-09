package botstopper

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBotStopperGetChallenge(t *testing.T) {

	t.Run("One", func(t *testing.T) {
		bs := NewBotStopper()
		challenge := bs.GetChallenge()

		assert.Equal(t, 1, len(bs.answers))
		assert.Contains(t, bs.answers, challenge.ID)
	})

	t.Run("Many", func(t *testing.T) {
		bs := NewBotStopper()

		count := maxSavedAnswers - 1

		challenges := make([]Challenge, count)
		for i := 0; i < count; i++ {
			challenges[i] = bs.GetChallenge()
		}

		assert.Equal(t, count, len(bs.answers))
		for _, challenge := range challenges {
			assert.Contains(t, bs.answers, challenge.ID)
		}
	})

	t.Run("ManyParallel", func(t *testing.T) {
		bs := NewBotStopper()

		count := maxSavedAnswers - 1

		challenges := make([]Challenge, count)
		ch := make(chan Challenge, count)

		for i := 0; i < count; i++ {
			go func() {
				ch <- bs.GetChallenge()
			}()
		}

		// await receiving all pending challenges
		for i := 0; i < count; i++ {
			challenges[i] = <-ch
		}

		assert.Equal(t, count, len(bs.answers))
		for _, challenge := range challenges {
			assert.Contains(t, bs.answers, challenge.ID)
		}
	})

	t.Run("HitCap", func(t *testing.T) {
		bs := NewBotStopper()

		count := maxSavedAnswers + 1

		challenges := make([]Challenge, count)
		ch := make(chan Challenge, count)

		for i := 0; i < count; i++ {
			go func() {
				ch <- bs.GetChallenge()
			}()
		}

		// await receiving all pending challenges
		for i := 0; i < count; i++ {
			challenges[i] = <-ch
		}

		assert.Equal(t, 1, len(bs.answers))
	})

	t.Run("HitCapAndPrune", func(t *testing.T) {
		// fake some expired challenges and check if they are pruned when hitting the cap
		bs := NewBotStopper()

		count := maxSavedAnswers + 1
		nonExpired := 20

		challenges := make([]Challenge, count)

		for i := 0; i < count; i++ {
			challenge := bs.GetChallenge()
			challenges[i] = challenge
			if i >= nonExpired {
				bs.answers[challenge.ID] = answer{
					expiry: time.Now().Add(-time.Second),
				}
			}
		}

		assert.Equal(t, nonExpired+1, len(bs.answers))
		for i := 0; i < nonExpired; i++ {
			assert.Contains(t, bs.answers, challenges[i].ID)
		}
	})

	t.Run("HitCapAndBeyond", func(t *testing.T) {
		// Check if we continue as usual after pruning all saved answers

		bs := NewBotStopper()

		beyond := 20
		count := maxSavedAnswers + beyond

		challenges := make([]Challenge, count)
		ch := make(chan Challenge, count)

		for i := 0; i < count; i++ {
			go func() {
				ch <- bs.GetChallenge()
			}()
		}

		// await receiving all pending challenges
		for i := 0; i < count; i++ {
			challenges[i] = <-ch
		}

		assert.Equal(t, beyond, len(bs.answers))

		found := 0
		for _, challenge := range challenges {
			if _, ok := bs.answers[challenge.ID]; ok {
				found++
			}
		}
		assert.Equal(t, beyond, found)
	})
}

func TestBotStopperVerify(t *testing.T) {

	t.Run("OK", func(t *testing.T) {
		bs := NewBotStopper()
		challenge := bs.GetChallenge()
		response := Response{
			ID:     challenge.ID,
			Answer: bs.answers[challenge.ID].value,
		}
		assert.True(t, bs.Verify(response))
	})

	t.Run("OKParallel", func(t *testing.T) {
		bs := NewBotStopper()
		for i := 0; i < maxSavedAnswers; i++ {
			go func() {
				challenge := bs.GetChallenge()

				bs.mutex.Lock()
				answer := bs.answers[challenge.ID].value
				bs.mutex.Unlock()

				response := Response{
					ID:     challenge.ID,
					Answer: answer,
				}
				assert.True(t, bs.Verify(response))
			}()
		}
	})

	t.Run("FailRandomlyParallel", func(t *testing.T) {
		bs := NewBotStopper()
		for i := 0; i < maxSavedAnswers; i++ {
			go func() {
				challenge := bs.GetChallenge()

				succeed := rand.Intn(2) == 0

				var answer string
				if succeed {
					bs.mutex.Lock()
					answer = bs.answers[challenge.ID].value
					bs.mutex.Unlock()
				}

				response := Response{
					ID:     challenge.ID,
					Answer: answer,
				}
				assert.Equal(t, succeed, bs.Verify(response))
			}()
		}
	})

	t.Run("FailChallengeUnknown", func(t *testing.T) {
		bs := NewBotStopper()
		response := Response{
			ID:     "foo",
			Answer: "bar",
		}
		assert.False(t, bs.Verify(response))
	})

	t.Run("FailChallengeExpired", func(t *testing.T) {
		bs := NewBotStopper()
		challenge := bs.GetChallenge()

		bs.answers[challenge.ID] = answer{
			expiry: time.Now().Add(-time.Second), // fake expired entry
			value:  "foo",
		}

		response := Response{
			ID:     challenge.ID,
			Answer: bs.answers[challenge.ID].value,
		}
		assert.False(t, bs.Verify(response))
	})

	t.Run("FailChallengeWrongAnswer", func(t *testing.T) {
		bs := NewBotStopper()
		challenge := bs.GetChallenge()

		response := Response{
			ID:     challenge.ID,
			Answer: bs.answers[challenge.ID].value + "wrong", // force wrong answer
		}
		assert.False(t, bs.Verify(response))
	})

}
