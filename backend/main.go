package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ========== グローバル変数 ==========

var messages []ReceivedMessage

// ========== データ構造 ==========

type ReceivedMessage struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userID"`
	Text      string    `json:"text"`
}

// ユーザー情報を管理する構造体
type User struct {
	UserID          string
	Name            string
	Circle          string
	Step            int // 0:未登録 1:名前待ち 2:サークル名待ち 3:完了
	SplitEventStep  int // 0:なし 1:イベント名待ち 2:金額待ち 3:参加者選択待ち
	TempEventID     int // 作成中のイベントID
	ApprovalStep    int // 0:なし 1:承認番号待ち
	ApprovalEventID int // 承認中のイベントID
}

// 割り勘イベント情報を管理する構造体
type Event struct {
	ID          int
	EventName   string
	OrganizerID string
	Circle      string
	TotalAmount int
	SplitAmount int
	Status      string // 'selecting' / 'confirmed' / 'completed'
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// イベント参加者情報を管理する構造体
type Participant struct {
	ID         int
	EventID    int
	UserID     string
	UserName   string
	Paid       bool
	ReportedAt *time.Time
	ApprovedAt *time.Time
	CreatedAt  time.Time
}

// 未払い参加者情報（催促用）
type UnpaidParticipant struct {
	UserID      string
	EventID     int
	EventName   string
	UserName    string
	SplitAmount int
	CreatedAt   time.Time
}

// ========== ユーティリティ関数 ==========

// sanitizeInput はサニタイズをする関数
func sanitizeInput(input string) string {
	sanitized := html.EscapeString(input)
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// formatAmount は金額フォーマット
func formatAmount(amount int) string {
	return fmt.Sprintf("%d", amount)
}

// ========== 催促システム ==========

// sendReminderToUnpaidUsers は未払いユーザーに催促メッセージを送信
func sendReminderToUnpaidUsers() {
	log.Println("[催促システム] 未払いユーザーの確認を開始...")

	participants, err := getUnpaidParticipants()
	if err != nil {
		log.Printf("[催促システム] エラー: %v", err)
		return
	}

	if len(participants) == 0 {
		log.Println("[催促システム] 未払いユーザーはいません")
		return
	}

	log.Printf("[催促システム] %d人の未払いユーザーに催促を送信します", len(participants))

	for _, p := range participants {
		message := fmt.Sprintf(
			"⏰ お支払いの催促\n\n"+
				"【イベント】%s\n"+
				"【金額】%s円\n\n"+
				"まだお支払いが確認できていません。\n"+
				"お支払い済みの場合は「💰 支払いました」ボタンから報告してください。",
			p.EventName,
			formatAmount(p.SplitAmount),
		)

		go func(userID, msg string) {
			if err := PushMessage(userID, msg); err != nil {
				log.Printf("[催促システム] 送信失敗 (UserID: %s): %v", userID, err)
			} else {
				log.Printf("[催促システム] 送信成功: %s", userID)
			}
		}(p.UserID, message)

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[催促システム] 催促メッセージの送信処理を完了しました")
}

// startReminderScheduler は毎日12時に催促を実行するスケジューラー
func startReminderScheduler() {
	go func() {
		log.Println("[催促システム] スケジューラーを起動しました")

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			duration := next.Sub(now)
			log.Printf("[催促システム] 次回実行: %s (%s後)", next.Format("2006-01-02 15:04:05"), duration.Round(time.Second))

			time.Sleep(duration)
			sendReminderToUnpaidUsers()
		}
	}()
}

// ========== 管理用APIハンドラー ==========

// handleTestReminder は手動で催促を実行（テスト用）
func handleTestReminder(w http.ResponseWriter, r *http.Request) {
	log.Println("[テスト] 催促システムを手動実行します")

	go sendReminderToUnpaidUsers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "催促メッセージの送信を開始しました",
	})
}

// handleSend はメッセージ送信（管理画面用）
func handleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"userID"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		log.Printf("エラー: User IDが空です")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		log.Printf("エラー: メッセージが空です")
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	log.Printf("送信試行: UserID=%s, Message=%s", req.UserID, req.Message)

	if err := PushMessage(req.UserID, req.Message); err != nil {
		log.Printf("送信エラー: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("送信成功: %s → %s", req.Message, req.UserID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAllMessages は受信メッセージ一覧取得
func handleAllMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleGetUsers はユーザー一覧取得
func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT user_id, name, circle, step
		FROM users
		ORDER BY updated_at DESC
	`)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var userList []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.Name, &user.Circle, &user.Step); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		userList = append(userList, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userList)
}

// ========== 静的ファイルハンドラー ==========

// addNgrokHeaders はngrok警告ページをスキップするヘッダーを追加
func addNgrokHeaders(w http.ResponseWriter) {
	w.Header().Set("ngrok-skip-browser-warning", "true")
}

// handleStaticFiles は静的ファイルハンドラー（SPA対応）
func handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	log.Printf("[静的ファイル] %s %s", r.Method, r.URL.Path)
	addNgrokHeaders(w)

	staticDir := "../frontend/dist"
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	filePath := staticDir + path

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.ServeFile(w, r, staticDir+"/index.html")
		return
	}

	http.ServeFile(w, r, filePath)
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

	if os.Getenv("LINE_CHANNEL_ACCESS_TOKEN") == "" {
		log.Fatal("LINE_CHANNEL_ACCESS_TOKEN is not set")
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

	// ========== ルーティング設定 ==========

	// LINE Bot関連（bot.go）
	http.HandleFunc("/webhook", handleWebhook)

	// LIFF関連（liff.go）- 認証付きハンドラー
	http.HandleFunc("/api/liff/register", WithAuth(handleRegisterUser))
	http.HandleFunc("/api/liff/message", WithAuth(handleLIFFMessage))
	http.HandleFunc("/api/liff/me", WithAuth(handleGetMyInfo))
	http.HandleFunc("/api/liff/events", WithAuth(handleEvents))
	http.HandleFunc("/api/liff/approvals", WithAuth(handleApprovals))
	http.HandleFunc("/api/liff/circle/members", WithAuth(handleGetCircleMembers))

	// 管理用API
	http.HandleFunc("/api/users", handleGetUsers)
	http.HandleFunc("/messages", handleAllMessages)
	http.HandleFunc("/send", handleSend)

	// テスト用API
	http.HandleFunc("/api/test/send-reminders", handleTestReminder)

	// 静的ファイル
	http.HandleFunc("/", handleStaticFiles)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Printf("LINE Bot webhook: /webhook")
	log.Printf("LIFF endpoints: /api/liff/*")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
