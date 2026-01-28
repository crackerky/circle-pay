package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const staticDir = "../frontend/dist"

// handleStaticFiles は静的ファイルを配信（SPA対応）
func handleStaticFiles(c *gin.Context) {
	path := c.Request.URL.Path

	log.Printf("[静的ファイル] %s %s", c.Request.Method, path)

	// ルートパスはindex.htmlを返す
	if path == "/" {
		c.File(filepath.Join(staticDir, "index.html"))
		return
	}

	// 実際のファイルを探す
	filePath := filepath.Join(staticDir, path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// SPAフォールバック: クライアントサイドルーティング用にindex.htmlを返す
		c.File(filepath.Join(staticDir, "index.html"))
		return
	}

	c.File(filePath)
}
