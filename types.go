package main

import (
	"regexp"
	"time"
)

// FlexibleKeywordSets allows dynamic loading of any category from JSON
type FlexibleKeywordSets map[string]interface{}

// CategoryInfo stores parsed information from category names
// Categories follow the pattern: {category}_{priority}_{stage}
// Example: "donotcall_p1_s3" or "honeypot_hardcoded_s2"
type CategoryInfo struct {
	BaseName    string // e.g., "donotcall", "honeypot"
	Priority    int    // e.g., 1, 2, 3 (0 for hardcoded)
	Stage       string // e.g., "s1", "s2", "s3"
	IsHardcoded bool   // true if priority is "hardcoded"
	ReturnValue string // what to return when matched
}

// keywordEntry stores both the keyword and its precompiled regex
type keywordEntry struct {
	raw   string
	regex *regexp.Regexp
}

// StageCategories groups categories by stage and priority
type StageCategories struct {
	Hardcoded   []CategoryEntry // Checked first, word boundaries only
	Prioritized []CategoryEntry // Checked in priority order (p1, p2, p3...)
}

// CategoryEntry links a category to its keywords
type CategoryEntry struct {
	Info     CategoryInfo
	Keywords []keywordEntry
}

// KeywordMatcher handles keyword matching for a specific campaign
type KeywordMatcher struct {
	// Map of stage -> StageCategories
	stageMap     map[string]*StageCategories
	contractions map[string]string
	loadedAt     time.Time
	filePath     string
}

// Request/Response structures
type MatchRequest struct {
	Campaign   string `json:"campaign" form:"campaign" query:"campaign"`
	SpeechText string `json:"speech_text" form:"speech_text" query:"speech_text"`
	Stage      string `json:"stage" form:"stage" query:"stage"` // Now accepts s1, s2, s3, etc.
}

type MatchResponse struct {
	Result   string `json:"result"`
	Stage    string `json:"stage"`
	Campaign string `json:"campaign"`
}

type ReloadResponse struct {
	Message    string    `json:"message"`
	Campaign   string    `json:"campaign,omitempty"`
	ReloadedAt time.Time `json:"reloaded_at"`
}

// matchResult stores information about a keyword match
type matchResult struct {
	keyword     string
	matchType   string // "exact", "phrase", "substring"
	length      int
	category    string
	returnValue string
}
