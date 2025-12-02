package main

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NewKeywordMatcher creates a matcher with dynamic category parsing
// Categories are parsed from JSON keys following the pattern: {category}_{priority}_{stage}
// Example: "donotcall_p1_s3" -> category="donotcall", priority=1, stage="s3"
// Hardcoded example: "honeypot_hardcoded_s2" -> category="honeypot", hardcoded=true, stage="s2"
func NewKeywordMatcher(rawKeywords FlexibleKeywordSets, filePath string) *KeywordMatcher {
	km := &KeywordMatcher{
		stageMap: make(map[string]*StageCategories),
		loadedAt: time.Now(),
		filePath: filePath,
		contractions: map[string]string{
			"i'm": "i am", "i've": "i have", "i'll": "i will", "i'd": "i would",
			"can't": "cannot", "won't": "will not", "don't": "do not",
			"doesn't": "does not", "didn't": "did not", "isn't": "is not",
			"aren't": "are not", "wasn't": "was not", "weren't": "were not",
			"hasn't": "has not", "haven't": "have not", "hadn't": "had not",
			"wouldn't": "would not", "shouldn't": "should not", "couldn't": "could not",
			"you're": "you are", "you've": "you have", "you'll": "you will", "you'd": "you would",
			"he's": "he is", "she's": "she is", "it's": "it is", "that's": "that is",
			"what's": "what is", "where's": "where is", "who's": "who is",
			"there's": "there is", "we're": "we are", "we've": "we have",
			"they're": "they are", "they've": "they have",
		},
	}

	// Parse all categories from JSON dynamically
	for categoryKey, value := range rawKeywords {
		info := parseCategoryName(categoryKey)
		if info == nil {
			log.Printf("Warning: Could not parse category name: %s", categoryKey)
			continue
		}

		// Convert keywords to string slice
		keywords := km.convertToStringSlice(value)
		if len(keywords) == 0 {
			continue
		}

		// Prepare keyword entries with regex
		entries := km.prepareKeywordEntries(keywords)

		// Initialize stage map if needed
		if _, exists := km.stageMap[info.Stage]; !exists {
			km.stageMap[info.Stage] = &StageCategories{
				Hardcoded:   make([]CategoryEntry, 0),
				Prioritized: make([]CategoryEntry, 0),
			}
		}

		// Add to appropriate list
		categoryEntry := CategoryEntry{
			Info:     *info,
			Keywords: entries,
		}

		if info.IsHardcoded {
			km.stageMap[info.Stage].Hardcoded = append(km.stageMap[info.Stage].Hardcoded, categoryEntry)
		} else {
			km.stageMap[info.Stage].Prioritized = append(km.stageMap[info.Stage].Prioritized, categoryEntry)
		}
	}

	// Sort prioritized categories by priority for each stage
	for stage, stageData := range km.stageMap {
		sort.Slice(stageData.Prioritized, func(i, j int) bool {
			return stageData.Prioritized[i].Info.Priority < stageData.Prioritized[j].Info.Priority
		})
		log.Printf("Loaded stage %s: %d hardcoded categories, %d prioritized categories",
			stage, len(stageData.Hardcoded), len(stageData.Prioritized))
	}

	return km
}

// parseCategoryName extracts category information from the category name
// Format: {category}_{priority}_{stage}
// Examples:
//   - "donotcall_p1_s3" -> BaseName="donotcall", Priority=1, Stage="s3"
//   - "honeypot_hardcoded_s2" -> BaseName="honeypot", IsHardcoded=true, Stage="s2"
//   - "interested_p2_s1" -> BaseName="interested", Priority=2, Stage="s1"
func parseCategoryName(name string) *CategoryInfo {
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return nil
	}

	// Find stage (last part should be s1, s2, s3, etc.)
	stage := parts[len(parts)-1]
	if !strings.HasPrefix(stage, "s") {
		return nil
	}

	// Find priority (second to last part should be p1, p2, hardcoded, etc.)
	priorityPart := parts[len(parts)-2]

	info := CategoryInfo{
		Stage: stage,
	}

	// Check if hardcoded
	if priorityPart == "hardcoded" {
		info.IsHardcoded = true
		info.Priority = 0
	} else if strings.HasPrefix(priorityPart, "p") {
		// Parse priority number
		var priority int
		_, err := fmt.Sscanf(priorityPart, "p%d", &priority)
		if err != nil {
			return nil
		}
		info.Priority = priority
		info.IsHardcoded = false
	} else {
		return nil
	}

	// Base name is everything before priority and stage
	info.BaseName = strings.Join(parts[:len(parts)-2], "_")

	// Generate return value based on category and stage
	// Format: CATEGORY_stage (e.g., "DO_NOT_CALL_s1", "HONEYPOT_s2")
	info.ReturnValue = generateReturnValue(info.BaseName, stage)

	return &info
}

// generateReturnValue creates the return value for a category match
// Returns only the category name without priority or stage
func generateReturnValue(baseName, stage string) string {
	// Convert common category names to standard return values
	switch baseName {
	case "donotcall", "do_not_call":
		return "DO_NOT_CALL"
	case "honeypot":
		return "HONEYPOT"
	case "answermachine", "answer_machine":
		return "ANSWER_MACHINE"
	case "interested":
		return "INTERESTED"
	case "notinterested", "not_interested":
		return "NOT_INTERESTED"
	case "dnq":
		return "DNQ"
	case "busy":
		return "BUSY"
	case "already":
		return "ALREADY"
	case "rebuttal":
		return "REBUTTAL"
	case "neutral":
		return "NEUTRAL"
	case "repeatpitch", "repeat_pitch":
		return "REPEAT_PITCH"
	case "greetingresponse", "greeting_response":
		return "GREETING_RESPONSE"
	case "notfeelinggood", "not_feeling_good":
		return "NOT_FEELING_GOOD"
	case "donttransfer", "dont_transfer":
		return "DONT_TRANSFER"
	default:
		// Generic: uppercase the base name
		return strings.ToUpper(baseName)
	}
}

// convertToStringSlice converts various JSON value types to string slice
func (km *KeywordMatcher) convertToStringSlice(value interface{}) []string {
	var result []string

	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	case []string:
		result = v
	case map[string]interface{}:
		// If it's a dict, use the values
		for _, val := range v {
			if str, ok := val.(string); ok {
				result = append(result, str)
			}
		}
	case string:
		// Single string value
		result = append(result, v)
	}

	return result
}

// prepareKeywordEntries normalizes keywords and creates regex patterns
func (km *KeywordMatcher) prepareKeywordEntries(keywords []string) []keywordEntry {
	entries := make([]keywordEntry, 0, len(keywords))

	for _, kw := range keywords {
		normalized := km.normalizeText(kw)
		if normalized != "" {
			// Precompile regex pattern with word boundaries
			pattern := `\b` + regexp.QuoteMeta(normalized) + `\b`
			re := regexp.MustCompile(pattern)

			entries = append(entries, keywordEntry{
				raw:   normalized,
				regex: re,
			})
		}
	}

	return entries
}

// normalizeText performs text normalization (same as before)
// - Unicode normalization
// - Lowercase conversion
// - Contraction expansion
// - Whitespace normalization
func (km *KeywordMatcher) normalizeText(text string) string {
	// Normalize unicode
	text = norm.NFKD.String(text)

	// Replace unicode spaces with regular spaces
	text = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, text)

	// Convert to lowercase
	text = strings.ToLower(strings.TrimSpace(text))

	// Expand contractions
	words := strings.Fields(text)
	for i, word := range words {
		if expansion, ok := km.contractions[word]; ok {
			words[i] = expansion
		}
	}
	text = strings.Join(words, " ")

	// Normalize multiple spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

// tokenize creates n-grams from text (same as before)
// Generates: unigrams, bigrams, trigrams, and 4-5 word phrases
func (km *KeywordMatcher) tokenize(text string) []string {
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words)*3)

	// Individual words
	tokens = append(tokens, words...)

	// Bigrams
	for i := 0; i < len(words)-1; i++ {
		tokens = append(tokens, words[i]+" "+words[i+1])
	}

	// Trigrams
	for i := 0; i < len(words)-2; i++ {
		tokens = append(tokens, words[i]+" "+words[i+1]+" "+words[i+2])
	}

	// Longer phrases (4-5 words)
	for length := 4; length <= 5 && length <= len(words); length++ {
		for i := 0; i <= len(words)-length; i++ {
			tokens = append(tokens, strings.Join(words[i:i+length], " "))
		}
	}

	return tokens
}

// findBestMatch finds the best keyword match from a list of category entries
// Matching priority: exact match > phrase match > substring match (with word boundaries)
// Returns the longest match found
func (km *KeywordMatcher) findBestMatch(text string, categories []CategoryEntry) *matchResult {
	normalized := km.normalizeText(text)

	// First: Check for exact matches across all categories
	for _, catEntry := range categories {
		for _, entry := range catEntry.Keywords {
			if entry.raw == normalized {
				return &matchResult{
					keyword:     normalized,
					matchType:   "exact",
					length:      len(normalized),
					category:    catEntry.Info.BaseName,
					returnValue: catEntry.Info.ReturnValue,
				}
			}
		}
	}

	// Second: Find best partial match (phrase or substring)
	var bestMatch *matchResult

	for _, catEntry := range categories {
		// Check phrase matches using tokenization
		tokens := km.tokenize(normalized)
		for _, entry := range catEntry.Keywords {
			for _, token := range tokens {
				if entry.raw == token {
					if bestMatch == nil || len(token) > bestMatch.length {
						bestMatch = &matchResult{
							keyword:     token,
							matchType:   "phrase",
							length:      len(token),
							category:    catEntry.Info.BaseName,
							returnValue: catEntry.Info.ReturnValue,
						}
					}
				}
			}
		}

		// Check substring matches with precompiled regex (word boundaries)
		for _, entry := range catEntry.Keywords {
			if entry.regex.MatchString(normalized) {
				if bestMatch == nil || len(entry.raw) > bestMatch.length {
					bestMatch = &matchResult{
						keyword:     entry.raw,
						matchType:   "substring",
						length:      len(entry.raw),
						category:    catEntry.Info.BaseName,
						returnValue: catEntry.Info.ReturnValue,
					}
				}
			}
		}
	}

	return bestMatch
}
