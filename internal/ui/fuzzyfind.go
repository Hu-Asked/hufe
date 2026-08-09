package ui

import (
	"path/filepath"
	"sort"
	"strings"
)

type FileEntry struct {
	Original string
	Lower    []rune
	BaseName []rune
}

type SearchResult struct {
	Score int
	Path  string
}

func PreProcessPaths(paths []string) []FileEntry {
	entries := make([]FileEntry, len(paths))
	for i, path := range paths {
		lowerPath := strings.ToLower(path)
		baseName := filepath.Base(lowerPath)
		
		entries[i] = FileEntry{
			Original: path,
			Lower:    []rune(lowerPath),
			BaseName: []rune(baseName),
		}
	}
	return entries
}

func getSubsequenceScore(query, path []rune) int {
	queryIndex := 0
	score := 0
	consecutiveMatch := 0

	for pathIndex, char := range path {
		if queryIndex < len(query) && char == query[queryIndex] {
			score += 10
			if consecutiveMatch > 0 {
				score += 5 * consecutiveMatch
			}
			consecutiveMatch++

			if pathIndex == 0 || path[pathIndex-1] == '/' || path[pathIndex-1] == '_' || path[pathIndex-1] == '-' || path[pathIndex-1] == '.' {
				score += 20
			}
			queryIndex++
		} else {
			consecutiveMatch = 0
		}
	}
	if queryIndex < len(query) {
		return -1
	}
	return score - len(path)
}

type Bigram [2]rune

func getBigrams(str []rune) map[Bigram]struct{} {
	bigrams := make(map[Bigram]struct{})
	for i := 0; i < len(str)-1; i++ {
		bigrams[Bigram{str[i], str[i+1]}] = struct{}{}
	}
	return bigrams
}

func getTypoScore(query, baseName []rune) int {
	if len(query) < 2 {
		return -1
	}

	minScore := 0.40

	queryBigrams := getBigrams(query)
	fileBigrams := getBigrams(baseName)

	if len(queryBigrams) == 0 || len(fileBigrams) == 0 {
		return -1
	}

	sharedBigrams := 0
	for bg := range queryBigrams {
		_, ok := fileBigrams[bg]
		if ok {
			sharedBigrams++
		}
	}

	similarity := float64(2*sharedBigrams) / float64(len(queryBigrams) + len(fileBigrams))
	if(similarity > minScore) {
		return int(similarity*100)
	}
	return -1
}

func FuzzyFind(query string, files []FileEntry) []SearchResult {
	if query == "" {
		return nil
	}

	typoScorePenalty := 30
	queryRunes := []rune(strings.ToLower(query))
	var results []SearchResult

	for _, f := range files {
		score := getSubsequenceScore(queryRunes, f.Lower)
		if score > -1 {
			results = append(results, SearchResult{Score: score, Path: f.Original})
		} else {
			typoScore := getTypoScore(queryRunes, f.Lower)
			if(typoScore > -1) {
				results = append(results, SearchResult{Score: typoScore - typoScorePenalty, Path: f.Original})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}