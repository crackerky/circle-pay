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
		primary_circle_id INTEGER,
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

	// サークルマスターテーブル
	circlesTable := `
	CREATE TABLE IF NOT EXISTS circles (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		created_by TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// ユーザーとサークルの関係テーブル（多対多）
	userCirclesTable := `
	CREATE TABLE IF NOT EXISTS user_circles (
		id SERIAL PRIMARY KEY,
		user_id TEXT NOT NULL,
		circle_id INTEGER NOT NULL,
		status TEXT DEFAULT 'active',
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		left_at TIMESTAMP,
		UNIQUE(user_id, circle_id)
	);`

	eventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id SERIAL PRIMARY KEY,
		event_name TEXT NOT NULL,
		organizer_id TEXT NOT NULL,
		circle TEXT NOT NULL,
		circle_id INTEGER,
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
	CREATE INDEX IF NOT EXISTS idx_events_circle ON events(circle);
	CREATE INDEX IF NOT EXISTS idx_events_circle_id ON events(circle_id);`

	indexParticipants := `
	CREATE INDEX IF NOT EXISTS idx_participants_event ON event_participants(event_id);
	CREATE INDEX IF NOT EXISTS idx_participants_user ON event_participants(user_id);`

	indexUserCircles := `
	CREATE INDEX IF NOT EXISTS idx_user_circles_user ON user_circles(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_circles_circle ON user_circles(circle_id);
	CREATE INDEX IF NOT EXISTS idx_user_circles_status ON user_circles(status);`

	// テーブル作成
	tables := []struct {
		name string
		sql  string
	}{
		{"users", usersTable},
		{"messages", messagesTable},
		{"circles", circlesTable},
		{"user_circles", userCirclesTable},
		{"events", eventsTable},
		{"event_participants", participantsTable},
		{"events_indexes", indexEvents},
		{"participants_indexes", indexParticipants},
		{"user_circles_indexes", indexUserCircles},
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
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS primary_circle_id INTEGER`,
		`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS reported_at TIMESTAMP`,
		`ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP`,
		`ALTER TABLE events ADD COLUMN IF NOT EXISTS circle_id INTEGER`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			// マイグレーションエラーはログに記録するが処理を続行
			// （既存のカラムがある場合など許容されるエラーもあるため）
			log.Printf("Migration warning: %v (SQL: %s)", err, m[:min(50, len(m))])
		}
	}

	// 既存データのマイグレーション（circles, user_circlesへの移行）
	if err := migrateCircleData(); err != nil {
		log.Printf("Circle migration warning: %v", err)
	}

	log.Println("Tables created successfully")
	return nil
}

// migrateCircleData は既存のusers.circleデータを新テーブルに移行する
func migrateCircleData() error {
	// 既にマイグレーション済みかチェック
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM circles`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		// 既にデータがある場合はスキップ
		return nil
	}

	log.Println("Migrating circle data to new tables...")

	// 1. 既存のサークル名をcirclesテーブルに移行
	_, err = db.Exec(`
		INSERT INTO circles (name, created_by)
		SELECT DISTINCT circle, MIN(user_id)
		FROM users
		WHERE circle IS NOT NULL AND circle != '' AND step = 3
		GROUP BY circle
		ON CONFLICT (name) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate circles: %w", err)
	}

	// 2. user_circlesに関係を作成
	_, err = db.Exec(`
		INSERT INTO user_circles (user_id, circle_id, status)
		SELECT u.user_id, c.id, 'active'
		FROM users u
		JOIN circles c ON u.circle = c.name
		WHERE u.circle IS NOT NULL AND u.circle != '' AND u.step = 3
		ON CONFLICT (user_id, circle_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate user_circles: %w", err)
	}

	// 3. users.primary_circle_idを設定
	_, err = db.Exec(`
		UPDATE users u
		SET primary_circle_id = c.id
		FROM circles c
		WHERE u.circle = c.name AND u.primary_circle_id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to set primary_circle_id: %w", err)
	}

	// 4. events.circle_idを設定
	_, err = db.Exec(`
		UPDATE events e
		SET circle_id = c.id
		FROM circles c
		WHERE e.circle = c.name AND e.circle_id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to set events.circle_id: %w", err)
	}

	log.Println("Circle data migration completed")
	return nil
}
