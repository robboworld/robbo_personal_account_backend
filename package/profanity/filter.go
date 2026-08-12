package profanity

import (
	"bufio"
	"embed"
	"strings"
	"sync"
)

//go:embed words_ru.txt words_en.txt
var wordFiles embed.FS

var (
	loadOnce sync.Once
	words    []string
)

func loadWords() {
	loadOnce.Do(func() {
		for _, name := range []string{"words_ru.txt", "words_en.txt"} {
			data, err := wordFiles.ReadFile(name)
			if err != nil {
				continue
			}
			sc := bufio.NewScanner(strings.NewReader(string(data)))
			for sc.Scan() {
				w := strings.TrimSpace(sc.Text())
				if w == "" || strings.HasPrefix(w, "#") {
					continue
				}
				// Skip very short roots to reduce false positives.
				if len([]rune(w)) < 3 {
					continue
				}
				words = append(words, Normalize(w))
			}
		}
	})
}

// ContainsProfanity reports whether text matches any dictionary word
// after normalization (substring match on normalized forms).
func ContainsProfanity(text string) bool {
	loadWords()
	norm := Normalize(text)
	if norm == "" {
		return false
	}
	for _, w := range words {
		if w == "" {
			continue
		}
		if strings.Contains(norm, w) {
			return true
		}
	}
	return false
}
