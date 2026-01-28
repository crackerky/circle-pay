package main

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// ========== グローバル変数 ==========

// receivedMessages は受信メッセージを一時保存（管理画面用）
// 注意: 複数goroutineからアクセスされるため、receivedMessagesMutexで保護する
var (
	receivedMessages      []ReceivedMessage
	receivedMessagesMutex sync.RWMutex
)

// addReceivedMessage はメッセージを安全に追加する
func addReceivedMessage(msg ReceivedMessage) {
	receivedMessagesMutex.Lock()
	defer receivedMessagesMutex.Unlock()

	// メモリリーク防止: 最大1000件に制限
	const maxMessages = 1000
	if len(receivedMessages) >= maxMessages {
		receivedMessages = receivedMessages[1:]
	}
	receivedMessages = append(receivedMessages, msg)
}

// getReceivedMessages はメッセージを安全に取得する
func getReceivedMessages() []ReceivedMessage {
	receivedMessagesMutex.RLock()
	defer receivedMessagesMutex.RUnlock()

	// コピーを返す（元のスライスを外部で変更されないように）
	result := make([]ReceivedMessage, len(receivedMessages))
	copy(result, receivedMessages)
	return result
}

// ========== メイン関数 ==========

func main() {
	// .envファイルを読み込み
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file not found in parent directory, trying current directory")
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: .env file not found, using system environment variables")
		}
	}

	// 必須環境変数のチェック
	requiredEnvs := []string{"LINE_CHANNEL_ACCESS_TOKEN", "LINE_CHANNEL_SECRET"}
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			log.Fatalf("%s is not set", env)
		}
	}

	// Admin認証の警告
	if os.Getenv("ADMIN_API_KEY") == "" {
		log.Println("WARNING: ADMIN_API_KEY not set - admin endpoints will be disabled")
	}

	// データベース初期化
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database: ", err)
	}
	defer db.Close()

	// テーブル作成
	if err := createTables(); err != nil {
		log.Fatal("Failed to create tables: ", err)
	}

	// 催促システムの起動
	startReminderScheduler()

	// Ginルーターをセットアップ
	router := setupRouter()

	// サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Printf("LINE Bot webhook: POST /webhook")
	log.Printf("LIFF endpoints: /api/liff/*")
	log.Printf("Admin endpoints: /api/admin/* (API key required)")

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
