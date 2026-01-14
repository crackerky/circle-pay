package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// ========== グローバル変数 ==========

var db *sql.DB

// ========== データベース初期化 ==========

// initDB はデータベースに接続する
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

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")
	return nil
}

// createTables はテーブルを作成する
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
	tables := []struct {
		name string
		sql  string
	}{
		{"users", usersTable},
		{"messages", messagesTable},
		{"events", eventsTable},
		{"event_participants", participantsTable},
		{"events_indexes", indexEvents},
		{"participants_indexes", indexParticipants},
	}

	for _, t := range tables {
		if _, err := db.Exec(t.sql); err != nil {
			return fmt.Errorf("failed to create %s: %w", t.name, err)
		}
	}

	// 既存のテーブルに新しいカラムを追加（エラーは無視）
	migrations := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS split_event_step INTEGER DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS temp_event_id INTEGER`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS approval_step INTEGER DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS approval_event_id INTEGER`,
		`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS reported_at TIMESTAMP`,
		`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP`,
	}

	for _, m := range migrations {
		db.Exec(m)
	}

	log.Println("Tables created successfully")
	return nil
}

// ========== ユーザー操作 ==========

// getUser はユーザーを取得する
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

// saveUser はユーザーを保存する
func saveUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (user_id, name, circle, step, split_event_step, temp_event_id, approval_step, approval_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.UserID, user.Name, user.Circle, user.Step, user.SplitEventStep, user.TempEventID, user.ApprovalStep, user.ApprovalEventID)
	return err
}

// updateUser はユーザーを更新する
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

// ========== メッセージ操作 ==========

// saveMessage はメッセージを保存する
func saveMessage(userID, text string) error {
	_, err := db.Exec(`
		INSERT INTO messages (user_id, text) VALUES ($1, $2)
	`, userID, text)
	return err
}

// ========== イベント操作 ==========

// getEvent はイベントを取得する
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

// ========== 参加者操作 ==========

// getUnpaidParticipants は未払い参加者を取得する（催促用）
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
