package main

import (
	"fmt"
	"log"
)

// ========== 参加者リポジトリ ==========

// CreateParticipant はイベント参加者を追加する
func CreateParticipant(eventID int, userID, userName string) error {
	_, err := db.Exec(`
		INSERT INTO event_participants (event_id, user_id, user_name, paid)
		VALUES ($1, $2, $3, false)
	`, eventID, userID, userName)
	return err
}

// GetUnpaidParticipants は未払い参加者を取得する（催促用）
func GetUnpaidParticipants() ([]UnpaidParticipant, error) {
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unpaid participants: %w", err)
	}

	return participants, nil
}

// PendingApproval は承認待ちの支払い情報
type PendingApproval struct {
	ID              int
	EventID         int
	ParticipantID   string
	ParticipantName string
	EventName       string
	Amount          int
	ReportedAt      *string
}

// GetPendingApprovals は指定ユーザー（会計者）の承認待ち一覧を取得する
func GetPendingApprovals(organizerID string) ([]PendingApproval, error) {
	rows, err := db.Query(`
		SELECT ep.id, ep.event_id, ep.user_id, ep.user_name, e.event_name, e.split_amount, ep.reported_at
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE e.organizer_id = $1 AND ep.paid = true AND ep.approved_at IS NULL
		ORDER BY ep.reported_at DESC
	`, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []PendingApproval
	for rows.Next() {
		var a PendingApproval
		if err := rows.Scan(&a.ID, &a.EventID, &a.ParticipantID, &a.ParticipantName, &a.EventName, &a.Amount, &a.ReportedAt); err != nil {
			log.Printf("承認スキャンエラー: %v", err)
			continue
		}
		approvals = append(approvals, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return approvals, nil
}

// GetParticipantOrganizerID は参加者レコードの会計者IDを取得する
func GetParticipantOrganizerID(participantID int) (string, error) {
	var organizerID string
	err := db.QueryRow(`
		SELECT e.organizer_id
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE ep.id = $1
	`, participantID).Scan(&organizerID)
	return organizerID, err
}

// ApproveParticipant は参加者の支払いを承認する
func ApproveParticipant(participantID int) error {
	_, err := db.Exec(`
		UPDATE event_participants
		SET approved_at = NOW()
		WHERE id = $1
	`, participantID)
	return err
}

// ApprovalNotifyInfo は承認通知用の情報
type ApprovalNotifyInfo struct {
	ParticipantUserID string
	ParticipantName   string
	EventName         string
	SplitAmount       int
}

// GetApprovalNotifyInfo は承認通知に必要な情報を取得する
func GetApprovalNotifyInfo(participantID int) (*ApprovalNotifyInfo, error) {
	var info ApprovalNotifyInfo
	err := db.QueryRow(`
		SELECT ep.user_id, ep.user_name, e.event_name, e.split_amount
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE ep.id = $1
	`, participantID).Scan(&info.ParticipantUserID, &info.ParticipantName, &info.EventName, &info.SplitAmount)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ReportPayment は支払い報告を記録する
func ReportPayment(eventID int, userID string) error {
	_, err := db.Exec(`
		UPDATE event_participants
		SET paid = true, reported_at = NOW()
		WHERE event_id = $1 AND user_id = $2
	`, eventID, userID)
	return err
}
