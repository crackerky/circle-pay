package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ========== LIFF認証ミドルウェア ==========

// LIFFAuthMiddleware はLIFFトークンを検証してユーザー情報をコンテキストに保存
func LIFFAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is missing",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}

		userID, displayName, err := verifyLIFFToken(token)
		if err != nil {
			log.Printf("LIFF token verification failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Ginコンテキストにユーザー情報を保存
		c.Set("userID", userID)
		c.Set("displayName", displayName)
		c.Next()
	}
}

// GetUserID はコンテキストからユーザーIDを取得
func GetUserID(c *gin.Context) string {
	return c.GetString("userID")
}

// GetDisplayName はコンテキストから表示名を取得
func GetDisplayName(c *gin.Context) string {
	return c.GetString("displayName")
}

// verifyLIFFToken はLIFFアクセストークンを検証してユーザーIDを取得
func verifyLIFFToken(accessToken string) (string, string, error) {
	// 1. LIFFアクセストークンの検証（URLエスケープを適用）
	verifyURL := "https://api.line.me/oauth2/v2.1/verify?access_token=" + url.QueryEscape(accessToken)

	resp, err := http.Get(verifyURL)
	if err != nil {
		return "", "", fmt.Errorf("token verification failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("invalid token: %s", string(body))
	}

	// 2. プロフィール情報を取得
	profileURL := "https://api.line.me/v2/profile"
	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	profileResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("profile fetch failed: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(profileResp.Body)
		return "", "", fmt.Errorf("failed to get profile: %s", string(body))
	}

	var profile struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		return "", "", fmt.Errorf("failed to decode profile: %w", err)
	}

	return profile.UserID, profile.DisplayName, nil
}

// ========== Admin認証ミドルウェア ==========

// AdminAuthMiddleware はAPIキーを検証
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("ADMIN_API_KEY")
		if apiKey == "" {
			log.Println("WARNING: ADMIN_API_KEY not set, admin endpoints disabled")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "admin endpoints not configured",
			})
			return
		}

		// X-API-Keyヘッダーをチェック
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			// クエリパラメータにフォールバック（ブラウザテスト用）
			providedKey = c.Query("api_key")
		}

		if providedKey != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or missing API key",
			})
			return
		}

		c.Next()
	}
}

// ========== LINE Webhook署名検証ミドルウェア ==========

// LineSignatureMiddleware はLINE Webhookの署名を検証
func LineSignatureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Line-Signature")
		if signature == "" {
			log.Println("LINE webhook: signature header missing")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// ボディを読み取り
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("LINE webhook: failed to read body: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// ボディを復元（ハンドラーで再度読めるように）
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 署名を検証
		if !validateWebhookSignature(bodyBytes, signature) {
			log.Println("LINE webhook: signature validation failed")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// ボディをコンテキストに保存（必要に応じて）
		c.Set("webhookBody", bodyBytes)
		c.Next()
	}
}

// validateWebhookSignature はLINE Webhookの署名を検証
func validateWebhookSignature(body []byte, signature string) bool {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	if channelSecret == "" {
		log.Println("LINE_CHANNEL_SECRET is not set")
		return false
	}

	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ========== 共通ミドルウェア ==========

// NgrokHeadersMiddleware はngrok警告ページをスキップするヘッダーを追加
func NgrokHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("ngrok-skip-browser-warning", "true")
		c.Next()
	}
}

// RequestLoggerMiddleware はリクエストをログに記録
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[%s] %s %s %d %v",
			method,
			path,
			clientIP,
			status,
			latency,
		)
	}
}
