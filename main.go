package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize campaign cache with file watcher
	keywordsDir := "keywords"
	var err error
	campaignCache, err = NewCampaignCache(keywordsDir)
	if err != nil {
		log.Fatalf("Failed to initialize campaign cache: %v", err)
	}
	defer campaignCache.Close()

	// Start file watcher in background
	go campaignCache.WatchFiles()

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	e.POST("/match", handleMatch)
	e.GET("/match", handleMatch)
	e.GET("/health", handleHealth)

	// Admin endpoints for manual reload
	e.POST("/admin/reload/:campaign", handleReloadCampaign)
	e.POST("/admin/reload-all", handleReloadAll)
	e.GET("/admin/cache-info", handleCacheInfo)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8050"
	}

	log.Printf("Keyword Matcher started on port %s", port)
	log.Printf("Watching keywords directory: %s", keywordsDir)
	log.Printf("Auto-reload enabled for keyword files")
	log.Printf("Using dynamic stage-priority system (s1, s2, s3, etc.)")

	e.Logger.Fatal(e.Start(":" + port))
}
