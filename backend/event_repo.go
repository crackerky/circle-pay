package main

import (
	"database/sql"
	"log"
)

// ========== イベントリポジトリ ==========

// GetEvent はイベントを取得する
func GetEvent(eventID int) (*Event, error) {
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

// CreateEvent は新しいイベントを作成する
func CreateEvent(eventName, organizerID, circle string, totalAmount, splitAmount int) (int, error) {
	var eventID int
	err := db.QueryRow(`
		INSERT INTO events (event_name, organizer_id, circle, total_amount, split_amount, status)
		VALUES ($1, $2, $3, $4, $5, 'confirmed')
		RETURNING id
	`, eventName, organizerID, circle, totalAmount, splitAmount).Scan(&eventID)

	if err != nil {
		return 0, err
	}
	return eventID, nil
}

// EventSummary はイベント一覧用のサマリー情報
type EventSummary struct {
	ID          int
	Name        string
	TotalAmount int
	SplitAmount int
	Status      string
	CreatedAt   string
}

// GetEventsByOrganizer は指定ユーザーが作成したイベント一覧を取得する
func GetEventsByOrganizer(organizerID string) ([]EventSummary, error) {
	rows, err := db.Query(`
		SELECT id, event_name, total_amount, split_amount, status, created_at
		FROM events
		WHERE organizer_id = $1
		ORDER BY created_at DESC
	`, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventSummary
	for rows.Next() {
		var e EventSummary
		if err := rows.Scan(&e.ID, &e.Name, &e.TotalAmount, &e.SplitAmount, &e.Status, &e.CreatedAt); err != nil {
			log.Printf("イベントスキャンエラー: %v", err)
			continue
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UnpaidEventInfo は未払いイベント情報
type UnpaidEventInfo struct {
	ID     int
	Name   string
	Amount int
}

// GetUnpaidEventsForUser はユーザーの未払いイベントを取得する
func GetUnpaidEventsForUser(userID string) ([]UnpaidEventInfo, error) {
	rows, err := db.Query(`
		SELECT e.id, e.event_name, e.split_amount
		FROM events e
		JOIN event_participants ep ON e.id = ep.event_id
		WHERE ep.user_id = $1 AND ep.paid = false AND e.status = 'confirmed'
		ORDER BY e.created_at DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []UnpaidEventInfo
	for rows.Next() {
		var e UnpaidEventInfo
		if err := rows.Scan(&e.ID, &e.Name, &e.Amount); err != nil {
			log.Printf("イベントスキャンエラー: %v", err)
			continue
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UserPaymentStatus はユーザーの支払い状況
type UserPaymentStatus struct {
	EventName string
	Amount    int
	Paid      bool
}

// GetUserPaymentStatus はユーザーの支払い状況一覧を取得する
func GetUserPaymentStatus(userID string) ([]UserPaymentStatus, error) {
	rows, err := db.Query(`
		SELECT e.event_name, e.split_amount, ep.paid
		FROM events e
		JOIN event_participants ep ON e.id = ep.event_id
		WHERE ep.user_id = $1
		ORDER BY e.created_at DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []UserPaymentStatus
	for rows.Next() {
		var s UserPaymentStatus
		if err := rows.Scan(&s.EventName, &s.Amount, &s.Paid); err != nil {
			log.Printf("ステータススキャンエラー: %v", err)
			continue
		}
		statuses = append(statuses, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return statuses, nil
}
