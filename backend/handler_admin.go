package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ========== 管理用APIハンドラー ==========

// handleTestReminder は手動で催促を実行（テスト用）
func handleTestReminder(c *gin.Context) {
	log.Println("[テスト] 催促システムを手動実行します")

	go sendReminderToUnpaidUsers()

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "催促メッセージの送信を開始しました",
	})
}

// handleSend はメッセージ送信（管理画面用）
func handleSend(c *gin.Context) {
	var req struct {
		UserID  string `json:"userID" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON or missing required fields"})
		return
	}

	log.Printf("送信試行: UserID=%s, Message=%s", req.UserID, req.Message)

	if err := PushMessage(req.UserID, req.Message); err != nil {
		log.Printf("送信エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("送信成功: %s → %s", req.Message, req.UserID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleAllMessages は受信メッセージ一覧取得
func handleAllMessages(c *gin.Context) {
	c.JSON(http.StatusOK, getReceivedMessages())
}

// handleGetUsers はユーザー一覧取得
func handleGetUsers(c *gin.Context) {
	users, err := GetAllUsers()
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// ========== リッチメニュー管理ハンドラー ==========

// handleRichMenuCreate はリッチメニューのセットアップエンドポイント
func handleRichMenuCreate(c *gin.Context) {
	log.Println("[リッチメニュー] セットアップ開始...")

	result, err := CreateCirclePayRichMenu()
	if err != nil {
		log.Printf("[リッチメニュー] 作成エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create rich menu: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"richMenuId": result.RichMenuID,
		"message":    "リッチメニューを作成しました。次に画像をアップロードしてください。",
		"nextStep":   fmt.Sprintf("POST /api/admin/richmenu/%s/image", result.RichMenuID),
	})
}

// handleRichMenuImageUpload はリッチメニュー画像アップロードエンドポイント
func handleRichMenuImageUpload(c *gin.Context) {
	richMenuID := c.Param("id")
	if richMenuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rich menu ID is required"})
		return
	}

	log.Printf("[リッチメニュー] 画像アップロード開始: %s", richMenuID)

	// マルチパートフォームから画像を読み取り
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image file is required"})
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
		return
	}

	// Content-Typeを判定
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	// 画像をアップロード
	if err := UploadRichMenuImageData(richMenuID, imageData, contentType); err != nil {
		log.Printf("[リッチメニュー] 画像アップロードエラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to upload image: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"message":  "画像をアップロードしました。次にデフォルトメニューとして設定してください。",
		"nextStep": fmt.Sprintf("POST /api/admin/richmenu/%s/default", richMenuID),
	})
}

// handleRichMenuSetDefault はデフォルトリッチメニュー設定エンドポイント
func handleRichMenuSetDefault(c *gin.Context) {
	richMenuID := c.Param("id")
	if richMenuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rich menu ID is required"})
		return
	}

	log.Printf("[リッチメニュー] デフォルト設定: %s", richMenuID)

	if err := SetDefaultRichMenu(richMenuID); err != nil {
		log.Printf("[リッチメニュー] デフォルト設定エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to set default: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "デフォルトリッチメニューを設定しました！全ユーザーにリッチメニューが表示されます。",
	})
}

// handleRichMenuList はリッチメニュー一覧取得エンドポイント
func handleRichMenuList(c *gin.Context) {
	result, err := GetRichMenuList()
	if err != nil {
		log.Printf("[リッチメニュー] 一覧取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get list: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleRichMenuDelete はリッチメニュー削除エンドポイント
func handleRichMenuDelete(c *gin.Context) {
	richMenuID := c.Param("id")
	if richMenuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rich menu ID is required"})
		return
	}

	log.Printf("[リッチメニュー] 削除: %s", richMenuID)

	if err := DeleteRichMenu(richMenuID); err != nil {
		log.Printf("[リッチメニュー] 削除エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "リッチメニューを削除しました。",
	})
}
