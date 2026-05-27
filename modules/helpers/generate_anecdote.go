package helpers

import (
	"fmt"
	"math/rand"
	"zaglyt-tg/constants"
	"zaglyt-tg/modules/z3abp"
)

func GenerateAnecdote(db []string) (string, error) {
	markovText := z3abp.GenerateRandomMarkov(db, 2, 10)
	if markovText == "" {
		return "", fmt.Errorf("failed to generate anecdote phrase")
	}

	randomTemplate := constants.Anecdotes[rand.Intn(len(constants.Anecdotes))]

	return fmt.Sprintf(randomTemplate, markovText), nil
}
