package z3abp

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/bbalet/stopwords"
	"github.com/crawlab-team/bm25"
	"github.com/kljensen/snowball"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Config struct {
	BM25K1      float64
	BM25B       float64
	TopKForGen  int
	MinGenWords int
	MaxGenWords int
}

func DefaultConfig() Config {
	return Config{
		BM25K1:      1.5,
		BM25B:       0.75,
		TopKForGen:  5,
		MinGenWords: 2,
		MaxGenWords: 10,
	}
}

type DocScore struct {
	Document string
	Score    float64
}

func Tokenize(s string) []string {
	s = strings.ToLower(s)
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func StemWords(words []string) []string {
	stemmed := make([]string, 0, len(words))
	for _, word := range words {
		stem, _ := snowball.Stem(word, "russian", true)
		stemmed = append(stemmed, stem)
	}
	return stemmed
}

func ContainsQuery(line, query string) bool {
	line = strings.ToLower(strings.TrimSpace(line))
	query = strings.ToLower(strings.TrimSpace(query))
	return line == query
}

func FilterLines(db []string, stems []string, query string) []string {
	var foundLines []string
	for _, line := range db {
		if ContainsQuery(line, query) {
			continue
		}

		words := Tokenize(line)
		lineStems := StemWords(words)
		matched := false

		for _, wordStem := range lineStems {
			for _, queryStem := range stems {
				if wordStem == queryStem {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		if matched {
			foundLines = append(foundLines, line)
		}
	}
	return foundLines
}

func RankDocuments(docs []string, query []string, k1, b float64) ([]DocScore, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	bm, err := bm25.NewBM25Okapi(docs, Tokenize, k1, b, nil)
	if err != nil {
		return nil, err
	}

	scores, err := bm.GetScores(query)
	if err != nil {
		return nil, err
	}

	rankedResults := make([]DocScore, len(scores))
	for i, score := range scores {
		rankedResults[i] = DocScore{
			Document: docs[i],
			Score:    score,
		}
	}

	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].Score > rankedResults[j].Score
	})

	return rankedResults, nil
}

type MarkovModel struct {
	Starts2      [][2]string
	Starts1      []string
	Transitions2 map[[2]string][]string
	Transitions1 map[string][]string
}

func normWord(w string) string {
	return strings.ToLower(strings.Trim(w, ".,!?:;\"'()[]{}«»\t\n\r "))
}

func buildMarkovChain(docs []string) MarkovModel {
	model := MarkovModel{
		Transitions2: make(map[[2]string][]string),
		Transitions1: make(map[string][]string),
	}

	for _, doc := range docs {
		words := strings.Fields(doc)
		if len(words) == 0 {
			continue
		}

		if len(words) == 1 {
			model.Starts1 = append(model.Starts1, words[0])
			model.Transitions1[normWord(words[0])] = append(model.Transitions1[normWord(words[0])], "")
			continue
		}

		model.Starts2 = append(model.Starts2, [2]string{words[0], words[1]})
		model.Starts1 = append(model.Starts1, words[0])

		model.Transitions1[normWord(words[0])] = append(model.Transitions1[normWord(words[0])], words[1])

		for i := 0; i < len(words)-2; i++ {
			w1, w2, w3 := words[i], words[i+1], words[i+2]

			norm2 := [2]string{normWord(w1), normWord(w2)}
			norm1 := normWord(w2)

			model.Transitions2[norm2] = append(model.Transitions2[norm2], w3)
			model.Transitions1[norm1] = append(model.Transitions1[norm1], w3)
		}

		lastWord := words[len(words)-1]
		prevWord := words[len(words)-2]

		normLast2 := [2]string{normWord(prevWord), normWord(lastWord)}
		normLast1 := normWord(lastWord)

		model.Transitions2[normLast2] = append(model.Transitions2[normLast2], "")
		model.Transitions1[normLast1] = append(model.Transitions1[normLast1], "")
	}

	return model
}

func generateMarkovText(model MarkovModel, minWords, maxWords int) string {
	if len(model.Starts1) == 0 && len(model.Starts2) == 0 {
		return ""
	}

	for attempt := 0; attempt < 50; attempt++ {
		var result []string
		var currNorm1 string
		var currNorm2 [2]string
		useOrder2 := false

		if len(model.Starts2) > 0 && rand.Float32() < 0.8 {
			start := model.Starts2[rand.Intn(len(model.Starts2))]
			result = append(result, start[0], start[1])
			currNorm2 = [2]string{normWord(start[0]), normWord(start[1])}
			currNorm1 = normWord(start[1])
			useOrder2 = true
		} else if len(model.Starts1) > 0 {
			start := model.Starts1[rand.Intn(len(model.Starts1))]
			result = append(result, start)
			currNorm1 = normWord(start)
		} else {
			continue
		}

		for len(result) < maxWords {
			var nextWords []string
			var exists bool

			if useOrder2 {
				nextWords, exists = model.Transitions2[currNorm2]
			}

			if !exists || len(nextWords) == 0 || rand.Float32() < 0.15 {
				altWords, altExists := model.Transitions1[currNorm1]
				if altExists && len(altWords) > 0 {
					nextWords = altWords
					exists = true
				}
			}

			if !exists || len(nextWords) == 0 {
				break
			}

			nextWord := nextWords[rand.Intn(len(nextWords))]
			if nextWord == "" {
				break
			}

			result = append(result, nextWord)

			if len(result) >= 2 {
				currNorm2 = [2]string{normWord(result[len(result)-2]), normWord(nextWord)}
				useOrder2 = true
			}
			currNorm1 = normWord(nextWord)
		}

		if len(result) >= minWords && len(result) <= maxWords {
			sentence := strings.Join(result, " ")
			cleaned := cleanTrailingPunctuation(sentence)
			if cleaned != "" {
				return cleaned
			}
		}
	}

	return ""
}

func cleanTrailingPunctuation(s string) string {
	s = strings.TrimRight(s, " ,-;:«»'\"")
	words := strings.Fields(s)

	for len(words) > 0 {
		lastWord := strings.ToLower(words[len(words)-1])
		lastWordNorm := strings.Trim(lastWord, ".,!?:;\"'()[]{}«»")

		cleaned := stopwords.CleanString(lastWordNorm, "ru", false)

		if strings.TrimSpace(cleaned) == "" {
			words = words[:len(words)-1]
		} else {
			break
		}
	}

	if len(words) == 0 {
		return ""
	}

	res := strings.Join(words, " ")
	return strings.TrimRight(res, " ,-;:«»'\"")
}

func GenerateBestResponse(queryStr string, db []string, cfg Config) (string, error) {
	queryWords := Tokenize(queryStr)
	stemmedQuery := StemWords(queryWords)

	matchedLines := FilterLines(
		db,
		stemmedQuery,
		queryStr,
	)

	if len(matchedLines) == 0 {
		return "", fmt.Errorf("matched lines not found")
	}

	rankedDb, err := RankDocuments(
		matchedLines,
		queryWords,
		cfg.BM25K1,
		cfg.BM25B,
	)

	if err != nil {
		return "", fmt.Errorf("sort error: %w", err)
	}

	topCount := cfg.TopKForGen
	if len(rankedDb) < topCount {
		topCount = len(rankedDb)
	}

	var topDocs []string
	for _, doc := range rankedDb[:topCount] {
		topDocs = append(topDocs, doc.Document)
	}

	model := buildMarkovChain(topDocs)

	generatedText := generateMarkovText(model, cfg.MinGenWords, cfg.MaxGenWords)

	if generatedText == "" {
		bestDoc := rankedDb[0].Document
		words := strings.Fields(bestDoc)
		limit := cfg.MaxGenWords
		if len(words) < limit {
			limit = len(words)
		}
		if limit == 0 {
			return "", fmt.Errorf("best document is empty")
		}
		generatedText = cleanTrailingPunctuation(strings.Join(words[:limit], " "))
	}

	return generatedText, nil
}

func GenerateRandomMarkov(db []string, minWords, maxWords int) string {
	if len(db) == 0 {
		return ""
	}
	model := buildMarkovChain(db)
	return generateMarkovText(model, minWords, maxWords)
}
