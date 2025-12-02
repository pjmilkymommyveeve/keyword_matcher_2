package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// CampaignCache caches loaded campaigns with file watching
type CampaignCache struct {
	sync.RWMutex
	matchers     map[string]*KeywordMatcher
	fileModTimes map[string]time.Time
	watcher      *fsnotify.Watcher
	keywordsDir  string
}

var campaignCache *CampaignCache

func NewCampaignCache(keywordsDir string) (*CampaignCache, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add keywords directory to watcher
	err = watcher.Add(keywordsDir)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch keywords directory: %w", err)
	}

	cache := &CampaignCache{
		matchers:     make(map[string]*KeywordMatcher),
		fileModTimes: make(map[string]time.Time),
		watcher:      watcher,
		keywordsDir:  keywordsDir,
	}

	log.Printf("File watcher initialized for: %s", keywordsDir)
	return cache, nil
}

func (cc *CampaignCache) Close() {
	if cc.watcher != nil {
		cc.watcher.Close()
	}
}

func (cc *CampaignCache) WatchFiles() {
	log.Println("File watcher started")

	for {
		select {
		case event, ok := <-cc.watcher.Events:
			if !ok {
				return
			}

			// Only process Write and Create events for .json files
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if strings.HasSuffix(event.Name, ".json") {
					// Extract campaign name from filename
					campaign := strings.TrimSuffix(filepath.Base(event.Name), ".json")

					// Small delay to ensure file write is complete
					time.Sleep(100 * time.Millisecond)

					log.Printf("File changed: %s, reloading campaign: %s", event.Name, campaign)

					// Reload the campaign
					cc.Lock()
					delete(cc.matchers, campaign)
					delete(cc.fileModTimes, campaign)
					cc.Unlock()

					log.Printf("Campaign '%s' cache cleared, will reload on next request", campaign)
				}
			}

		case err, ok := <-cc.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (cc *CampaignCache) isFileModified(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	modTime := info.ModTime()

	cc.RLock()
	lastModTime, exists := cc.fileModTimes[filePath]
	cc.RUnlock()

	if !exists || modTime.After(lastModTime) {
		cc.Lock()
		cc.fileModTimes[filePath] = modTime
		cc.Unlock()
		return true, nil
	}

	return false, nil
}

func getMatcher(campaign string) (*KeywordMatcher, error) {
	filePath := filepath.Join(campaignCache.keywordsDir, campaign+".json")

	// Check if file has been modified
	modified, err := campaignCache.isFileModified(filePath)
	if err == nil && modified {
		// File was modified, clear cache
		campaignCache.Lock()
		delete(campaignCache.matchers, campaign)
		campaignCache.Unlock()
		log.Printf("Detected modification for %s, reloading...", campaign)
	}

	campaignCache.RLock()
	matcher, exists := campaignCache.matchers[campaign]
	campaignCache.RUnlock()

	if exists {
		return matcher, nil
	}

	// Load campaign keywords
	campaignCache.Lock()
	defer campaignCache.Unlock()

	// Double-check after acquiring write lock
	if matcher, exists := campaignCache.matchers[campaign]; exists {
		return matcher, nil
	}

	// Load from file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load campaign keywords: %w", err)
	}

	var rawKeywords FlexibleKeywordSets
	if err := json.Unmarshal(data, &rawKeywords); err != nil {
		return nil, fmt.Errorf("failed to parse campaign keywords: %w", err)
	}

	matcher = NewKeywordMatcher(rawKeywords, filePath)
	campaignCache.matchers[campaign] = matcher

	log.Printf("Loaded campaign: %s", campaign)

	return matcher, nil
}
