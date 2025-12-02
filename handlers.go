package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"timestamp":   time.Now(),
		"auto_reload": "enabled",
	})
}

func handleCacheInfo(c echo.Context) error {
	campaignCache.RLock()
	defer campaignCache.RUnlock()

	info := make(map[string]interface{})
	campaigns := make([]map[string]interface{}, 0)

	for campaign, matcher := range campaignCache.matchers {
		stageInfo := make(map[string]interface{})
		for stage, stageData := range matcher.stageMap {
			stageInfo[stage] = map[string]interface{}{
				"hardcoded_categories":   len(stageData.Hardcoded),
				"prioritized_categories": len(stageData.Prioritized),
			}
		}

		campaigns = append(campaigns, map[string]interface{}{
			"campaign":  campaign,
			"loaded_at": matcher.loadedAt,
			"file_path": matcher.filePath,
			"stages":    stageInfo,
		})
	}

	info["cached_campaigns"] = len(campaignCache.matchers)
	info["campaigns"] = campaigns
	info["timestamp"] = time.Now()

	return c.JSON(http.StatusOK, info)
}

func handleReloadCampaign(c echo.Context) error {
	campaign := c.Param("campaign")

	campaignCache.Lock()
	delete(campaignCache.matchers, campaign)
	delete(campaignCache.fileModTimes, campaign)
	campaignCache.Unlock()

	return c.JSON(http.StatusOK, ReloadResponse{
		Message:    fmt.Sprintf("Campaign '%s' cache cleared and will reload on next request", campaign),
		Campaign:   campaign,
		ReloadedAt: time.Now(),
	})
}

func handleReloadAll(c echo.Context) error {
	campaignCache.Lock()
	count := len(campaignCache.matchers)
	campaignCache.matchers = make(map[string]*KeywordMatcher)
	campaignCache.fileModTimes = make(map[string]time.Time)
	campaignCache.Unlock()

	return c.JSON(http.StatusOK, ReloadResponse{
		Message:    fmt.Sprintf("All %d campaign caches cleared and will reload on next request", count),
		ReloadedAt: time.Now(),
	})
}

func handleMatch(c echo.Context) error {
	var req MatchRequest

	// Bind request (works for both POST JSON and GET query params)
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate required fields
	if req.Campaign == "" || req.SpeechText == "" || req.Stage == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "campaign, speech_text, and stage are required",
		})
	}

	// Validate stage format (must be s1, s2, s3, etc.)
	if !strings.HasPrefix(req.Stage, "s") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid stage format. Must be s1, s2, s3, etc.",
		})
	}

	// Get or load matcher for campaign
	matcher, err := getMatcher(req.Campaign)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Campaign not found: %s", req.Campaign),
		})
	}

	// Process using generic stage processor
	result := matcher.ProcessStage(req.SpeechText, req.Stage)

	return c.JSON(http.StatusOK, MatchResponse{
		Result:   result,
		Stage:    req.Stage,
		Campaign: req.Campaign,
	})
}
