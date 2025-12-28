package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// ========== グローバル変数 ==========
var db *sql.DB
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
	UserName    string
	EventID     int
	EventName   string
	SplitAmount int
	CreatedAt   time.Time
}

// ========== データベース初期化 ==========

// データベース接続
func initDB() error {
	var err error
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 接続確認
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")
	return nil
}

// テーブル作成
func createTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		name TEXT,
		circle TEXT,
		step INTEGER NOT NULL,
		split_event_step INTEGER DEFAULT 0,
		temp_event_id INTEGER,
		approval_step INTEGER DEFAULT 0,
		approval_event_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	messagesTable := `
	CREATE TABLE IF NOT EXISTS messages(
		id SERIAL PRIMARY KEY,
		user_id TEXT NOT NULL,
		text TEXT NOT NULL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	eventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id SERIAL PRIMARY KEY,
		event_name TEXT NOT NULL,
		organizer_id TEXT NOT NULL,
		circle TEXT NOT NULL,
		total_amount INTEGER NOT NULL,
		split_amount INTEGER NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	participantsTable := `
	CREATE TABLE IF NOT EXISTS event_participants (
		id SERIAL PRIMARY KEY,
		event_id INTEGER NOT NULL REFERENCES events(id),
		user_id TEXT NOT NULL,
		user_name TEXT NOT NULL,
		paid BOOLEAN DEFAULT FALSE,
		reported_at TIMESTAMP,
		approved_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	indexEvents := `
	CREATE INDEX IF NOT EXISTS idx_events_organizer ON events(organizer_id);
	CREATE INDEX IF NOT EXISTS idx_events_circle ON events(circle);`

	indexParticipants := `
	CREATE INDEX IF NOT EXISTS idx_participants_event ON event_participants(event_id);
	CREATE INDEX IF NOT EXISTS idx_participants_user ON event_participants(user_id);`

	// テーブル作成
	if _, err := db.Exec(usersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.Exec(messagesTable); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	if _, err := db.Exec(eventsTable); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	if _, err := db.Exec(participantsTable); err != nil {
		return fmt.Errorf("failed to create event_participants table: %w", err)
	}

	if _, err := db.Exec(indexEvents); err != nil {
		return fmt.Errorf("failed to create events indexes: %w", err)
	}

	if _, err := db.Exec(indexParticipants); err != nil {
		return fmt.Errorf("failed to create participants indexes: %w", err)
	}

	// 既存のテーブルに新しいカラムを追加（エラーは無視）
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS split_event_step INTEGER DEFAULT 0`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS temp_event_id INTEGER`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS approval_step INTEGER DEFAULT 0`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS approval_event_id INTEGER`)
	db.Exec(`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS reported_at TIMESTAMP`)
	db.Exec(`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP`)

	log.Println("Tables created successfully")
	return nil
}

// ========== データベース操作関数 ==========

// ユーザー取得
func getUser(userID string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT user_id, name, circle, step, split_event_step,
		       COALESCE(temp_event_id, 0), approval_step, COALESCE(approval_event_id, 0)
		FROM users WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Name, &user.Circle, &user.Step,
		&user.SplitEventStep, &user.TempEventID, &user.ApprovalStep, &user.ApprovalEventID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ユーザー保存
func saveUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (user_id, name, circle, step, split_event_step, temp_event_id, approval_step, approval_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.UserID, user.Name, user.Circle, user.Step, user.SplitEventStep, user.TempEventID, user.ApprovalStep, user.ApprovalEventID)
	return err
}

// ユーザー更新
func updateUser(user *User) error {
	_, err := db.Exec(`
		UPDATE users
		SET name = $1, circle = $2, step = $3, split_event_step = $4,
		    temp_event_id = $5, approval_step = $6, approval_event_id = $7, updated_at = NOW()
		WHERE user_id = $8
	`, user.Name, user.Circle, user.Step, user.SplitEventStep, user.TempEventID,
	   user.ApprovalStep, user.ApprovalEventID, user.UserID)
	return err
}

// メッセージ保存
func saveMessage(userID, text string) error {
	_, err := db.Exec(`
		INSERT INTO messages (user_id, text) VALUES ($1, $2)
	`, userID, text)
	return err
}

// イベント取得
func getEvent(eventID int) (*Event, error) {
	var event Event
	err := db.QueryRow(`
		SELECT id, event_name, organizer_id, circle, total_amount, split_amount, status, created_at, updated_at
		FROM events WHERE id = $1
	`, eventID).Scan(&event.ID, &event.EventName, &event.OrganizerID, &event.Circle,
		&event.TotalAmount, &event.SplitAmount, &event.Status, &event.CreatedAt, &event.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// 未払い参加者を取得（催促用）
func getUnpaidParticipants() ([]UnpaidParticipant, error) {
	rows, err := db.Query(`
		SELECT
			ep.user_id,
			ep.user_name,
			ep.event_id,
			e.event_name,
			e.split_amount,
			ep.created_at
		FROM event_participants ep
		INNER JOIN events e ON ep.event_id = e.id
		WHERE ep.paid = FALSE
		  AND ep.approved_at IS NULL
		  AND e.status IN ('confirmed', 'selecting')
		ORDER BY ep.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query unpaid participants: %w", err)
	}
	defer rows.Close()

	var participants []UnpaidParticipant
	for rows.Next() {
		var p UnpaidParticipant
		if err := rows.Scan(&p.UserID, &p.UserName, &p.EventID, &p.EventName, &p.SplitAmount, &p.CreatedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// ========== ユーティリティ関数 ==========

// サニタイズをする関数
func sanitizeInput(input string) string {
	sanitized := html.EscapeString(input)
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// 金額フォーマット
func formatAmount(amount int) string {
	return fmt.Sprintf("%d", amount)
}

// ========== 催促システム ==========

// 未払いユーザーに催促メッセージを送信
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

		// 非同期で送信（エラーがあってもログだけ出して続行）
		go func(userID, msg string) {
			if err := PushMessage(userID, msg); err != nil {
				log.Printf("[催促システム] 送信失敗 (UserID: %s): %v", userID, err)
			} else {
				log.Printf("[催促システム] 送信成功: %s", userID)
			}
		}(p.UserID, message)

		// レート制限を避けるため少し待機
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[催促システム] 催促メッセージの送信処理を完了しました")
}

// 毎日12時に催促を実行するスケジューラー
func startReminderScheduler() {
	go func() {
		log.Println("[催促システム] スケジューラーを起動しました")

		for {
			now := time.Now()

			// 次の12時を計算
			next := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
			if now.After(next) {
				// 今日の12時を過ぎている場合は明日の12時
				next = next.Add(24 * time.Hour)
			}

			duration := next.Sub(now)
			log.Printf("[催促システム] 次回実行: %s (%s後)", next.Format("2006-01-02 15:04:05"), duration.Round(time.Second))

			// 次の12時まで待機
			time.Sleep(duration)

			// 催促メッセージを送信
			sendReminderToUnpaidUsers()
		}
	}()
}

// ========== 管理用APIハンドラー ==========

// 手動で催促を実行（テスト用）
func handleTestReminder(w http.ResponseWriter, r *http.Request) {
	log.Println("[テスト] 催促システムを手動実行します")

	go sendReminderToUnpaidUsers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "催促メッセージの送信を開始しました",
	})
}

// メッセージ送信（管理画面用）
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

// 受信メッセージ一覧取得
func handleAllMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// ユーザー一覧取得
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

// ngrok警告ページをスキップするヘッダーを追加
func addNgrokHeaders(w http.ResponseWriter) {
	w.Header().Set("ngrok-skip-browser-warning", "true")
}

// 静的ファイルハンドラー（SPA対応）
func handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	log.Printf("[静的ファイル] %s %s", r.Method, r.URL.Path)
	addNgrokHeaders(w)

	// 静的ファイルのディレクトリ
	staticDir := "../frontend/dist"

	// リクエストされたパス
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// ファイルパスを構築
	filePath := staticDir + path

	// ファイルが存在するか確認
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// ファイルが存在しない場合はindex.htmlを返す（SPA対応）
		http.ServeFile(w, r, staticDir+"/index.html")
		return
	}

	// ファイルを返す
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
