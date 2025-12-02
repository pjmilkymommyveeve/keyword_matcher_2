package main

import (
	"strings"
)

// ProcessStage is the generic stage processor for any stage (s1, s2, s3, etc.)
// It follows this matching order:
// 1. Check hardcoded keywords first (with word boundaries only)
// 2. Check prioritized categories in order (p1, p2, p3, etc.)
// 3. Within each priority level, use the standard matching algorithm:
//   - Exact match (entire text matches keyword)
//   - Phrase match (tokenized n-grams match keyword)
//   - Substring match (keyword found with word boundaries)
//
// 4. Return "UNKNOWN_{stage}" if no match found
func (km *KeywordMatcher) ProcessStage(text, stage string) string {
	// Get stage data
	stageData, exists := km.stageMap[stage]
	if !exists {
		return "UNKNOWN_" + stage
	}

	normalized := km.normalizeText(text)

	// Step 1: Check hardcoded keywords first (word boundaries only)
	if len(stageData.Hardcoded) > 0 {
		result := km.findBestMatch(text, stageData.Hardcoded)
		if result != nil {
			return result.returnValue
		}
	}

	// Step 2: Check prioritized categories in order (p1, p2, p3, etc.)
	// Categories are already sorted by priority in NewKeywordMatcher
	currentPriority := -1
	var currentPriorityCategories []CategoryEntry

	for i, catEntry := range stageData.Prioritized {
		// Group categories by priority level
		if catEntry.Info.Priority != currentPriority {
			// Check previous priority level if we have accumulated categories
			if len(currentPriorityCategories) > 0 {
				result := km.findBestMatch(text, currentPriorityCategories)
				if result != nil {
					return result.returnValue
				}
			}

			// Start new priority level
			currentPriority = catEntry.Info.Priority
			currentPriorityCategories = []CategoryEntry{catEntry}
		} else {
			// Add to current priority level
			currentPriorityCategories = append(currentPriorityCategories, catEntry)
		}

		// Check last priority level if we're at the end
		if i == len(stageData.Prioritized)-1 && len(currentPriorityCategories) > 0 {
			result := km.findBestMatch(text, currentPriorityCategories)
			if result != nil {
				return result.returnValue
			}
		}
	}

	// Special case handling for common patterns (optional, can be removed if not needed)
	// These maintain backward compatibility with old behavior
	if stage == "s3" || stage == "s4" { // pitch/rebuttal stages
		if normalized == "no" || strings.HasPrefix(normalized, "no ") {
			// Check if DNQ category exists for this stage
			for _, catEntry := range stageData.Prioritized {
				if catEntry.Info.BaseName == "dnq" {
					return catEntry.Info.ReturnValue
				}
			}
		}
	}

	// Step 3: No match found
	return "UNKNOWN_" + stage
}
