package main

import (
	"database/sql"
	"log"
)

// ========== ユーザーリポジトリ ==========

// GetUser はユーザーを取得する
func GetUser(userID string) (*User, error) {
	var user User
	var primaryCircleID sql.NullInt64
	err := db.QueryRow(`
		SELECT user_id, name, COALESCE(circle, ''), primary_circle_id, step, split_event_step,
		       COALESCE(temp_event_id, 0), approval_step, COALESCE(approval_event_id, 0)
		FROM users WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Name, &user.Circle, &primaryCircleID, &user.Step,
		&user.SplitEventStep, &user.TempEventID, &user.ApprovalStep, &user.ApprovalEventID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if primaryCircleID.Valid {
		id := int(primaryCircleID.Int64)
		user.PrimaryCircleID = &id
	}

	return &user, nil
}

// SaveUser はユーザーを保存する
func SaveUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (user_id, name, circle, primary_circle_id, step, split_event_step, temp_event_id, approval_step, approval_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.UserID, user.Name, user.Circle, user.PrimaryCircleID, user.Step, user.SplitEventStep, user.TempEventID, user.ApprovalStep, user.ApprovalEventID)
	return err
}

// UpdateUser はユーザーを更新する
func UpdateUser(user *User) error {
	_, err := db.Exec(`
		UPDATE users
		SET name = $1, circle = $2, primary_circle_id = $3, step = $4, split_event_step = $5,
		    temp_event_id = $6, approval_step = $7, approval_event_id = $8, updated_at = NOW()
		WHERE user_id = $9
	`, user.Name, user.Circle, user.PrimaryCircleID, user.Step, user.SplitEventStep, user.TempEventID,
		user.ApprovalStep, user.ApprovalEventID, user.UserID)
	return err
}

// GetAllUsers は全ユーザーを取得する
func GetAllUsers() ([]User, error) {
	rows, err := db.Query(`
		SELECT user_id, name, circle, step
		FROM users
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.Name, &user.Circle, &user.Step); err != nil {
			log.Printf("ユーザースキャンエラー: %v", err)
			continue
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUsersByCircle は同じサークルのユーザーを取得する
func GetUsersByCircle(circle string, excludeUserID string) ([]User, error) {
	rows, err := db.Query(`
		SELECT user_id, name, circle
		FROM users
		WHERE circle = $1 AND step = 3 AND user_id != $2
		ORDER BY name
	`, circle, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.Name, &user.Circle); err != nil {
			log.Printf("ユーザースキャンエラー: %v", err)
			continue
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
