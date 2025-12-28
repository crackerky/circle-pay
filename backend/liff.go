package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// ========== LIFFトークン検証 ==========

// verifyLIFFToken はLIFFアクセストークンを検証してユーザーIDを取得
func verifyLIFFToken(accessToken string) (string, string, error) {
	// 1. LIFFアクセストークンの検証
	verifyURL := "https://api.line.me/oauth2/v2.1/verify?access_token=" + accessToken

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
	req, _ := http.NewRequest("GET", profileURL, nil)
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

// ========== LIFF APIハンドラー ==========

// handleRegisterUser はユーザー情報を登録
func handleRegisterUser(ctx *APIContext) {
	var req struct {
		Circle string `json:"circle"`
	}

	if !ctx.DecodeJSON(&req) {
		return
	}

	circle := sanitizeInput(req.Circle)
	if circle == "" {
		ctx.Error("Circle is required", http.StatusBadRequest)
		return
	}

	existingUser, err := getUser(ctx.UserID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		ctx.Error("Internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		existingUser.Name = ctx.DisplayName
		existingUser.Circle = circle
		existingUser.Step = 3
		if err := updateUser(existingUser); err != nil {
			log.Printf("ユーザー更新エラー: %v", err)
			ctx.Error("Failed to update user", http.StatusInternalServerError)
			return
		}
		log.Printf("ユーザー更新成功: %s (%s)", ctx.DisplayName, ctx.UserID)
	} else {
		newUser := &User{
			UserID: ctx.UserID,
			Name:   ctx.DisplayName,
			Circle: circle,
			Step:   3,
		}
		if err := saveUser(newUser); err != nil {
			log.Printf("ユーザー保存エラー: %v", err)
			ctx.Error("Failed to save user", http.StatusInternalServerError)
			return
		}
		log.Printf("新規ユーザー登録成功: %s (%s)", ctx.DisplayName, ctx.UserID)
	}

	ctx.Success(map[string]interface{}{
		"userId":      ctx.UserID,
		"displayName": ctx.DisplayName,
		"circle":      circle,
	})
}

// handleLIFFMessage はLIFFからのメッセージを処理
func handleLIFFMessage(ctx *APIContext) {
	var req struct {
		Message string `json:"message"`
	}

	if !ctx.DecodeJSON(&req) {
		return
	}

	message := sanitizeInput(req.Message)
	if message == "" {
		ctx.Error("Message is required", http.StatusBadRequest)
		return
	}

	if err := saveMessage(ctx.UserID, message); err != nil {
		log.Printf("メッセージ保存エラー: %v", err)
	}

	ctx.Success(map[string]interface{}{
		"reply": "メッセージを受け取りました",
	})
}

// handleGetMyInfo はユーザー情報を取得
func handleGetMyInfo(ctx *APIContext) {
	user, err := getUser(ctx.UserID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		ctx.Error("Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		ctx.Success(map[string]interface{}{
			"userId":      ctx.UserID,
			"displayName": ctx.DisplayName,
			"registered":  false,
		})
		return
	}

	ctx.Success(map[string]interface{}{
		"userId":      user.UserID,
		"name":        user.Name,
		"displayName": ctx.DisplayName,
		"circle":      user.Circle,
		"registered":  true,
		"step":        user.Step,
	})
}

// ========== イベント管理API ==========

// handleEvents はイベント管理エンドポイント
func handleEvents(ctx *APIContext) {
	switch ctx.Request.Method {
	case "GET":
		getMyEvents(ctx)
	case "POST":
		createEvent(ctx)
	default:
		ctx.Error("Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getMyEvents は自分が作成したイベント一覧を取得
func getMyEvents(ctx *APIContext) {
	rows, err := db.Query(`
		SELECT id, event_name, total_amount, split_amount, status, created_at
		FROM events
		WHERE organizer_id = $1
		ORDER BY created_at DESC
	`, ctx.UserID)

	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		ctx.Error("Failed to get events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		var totalAmount, splitAmount int
		var status string
		var createdAt string

		if err := rows.Scan(&id, &name, &totalAmount, &splitAmount, &status, &createdAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}

		events = append(events, map[string]interface{}{
			"id":          id,
			"name":        name,
			"totalAmount": totalAmount,
			"splitAmount": splitAmount,
			"status":      status,
			"createdAt":   createdAt,
		})
	}

	ctx.Success(map[string]interface{}{
		"events": events,
	})
}

// createEvent は新しいイベントを作成
func createEvent(ctx *APIContext) {
	var req struct {
		EventName      string   `json:"eventName"`
		TotalAmount    int      `json:"totalAmount"`
		ParticipantIDs []string `json:"participantIds"`
	}

	if !ctx.DecodeJSON(&req) {
		return
	}

	// バリデーション
	if req.EventName == "" {
		ctx.Error("Event name is required", http.StatusBadRequest)
		return
	}
	if req.TotalAmount <= 0 {
		ctx.Error("Total amount must be positive", http.StatusBadRequest)
		return
	}
	if len(req.ParticipantIDs) == 0 {
		ctx.Error("At least one participant is required", http.StatusBadRequest)
		return
	}

	organizer, err := getUser(ctx.UserID)
	if err != nil || organizer == nil {
		ctx.Error("User not found", http.StatusBadRequest)
		return
	}

	splitAmount := req.TotalAmount / len(req.ParticipantIDs)

	var eventID int
	err = db.QueryRow(`
		INSERT INTO events (event_name, organizer_id, circle, total_amount, split_amount, status)
		VALUES ($1, $2, $3, $4, $5, 'confirmed')
		RETURNING id
	`, req.EventName, ctx.UserID, organizer.Circle, req.TotalAmount, splitAmount).Scan(&eventID)

	if err != nil {
		log.Printf("イベント作成エラー: %v", err)
		ctx.Error("Failed to create event", http.StatusInternalServerError)
		return
	}

	// 参加者を登録
	for _, participantID := range req.ParticipantIDs {
		participant, err := getUser(participantID)
		if err != nil || participant == nil {
			log.Printf("参加者取得エラー: %v", participantID)
			continue
		}

		_, err = db.Exec(`
			INSERT INTO event_participants (event_id, user_id, user_name, paid)
			VALUES ($1, $2, $3, false)
		`, eventID, participantID, participant.Name)

		if err != nil {
			log.Printf("参加者登録エラー: %v", err)
		}
	}

	// 参加者に通知を送信（非同期）
	go func() {
		for _, participantID := range req.ParticipantIDs {
			notifyText := fmt.Sprintf("【割り勘のお知らせ】\n%sさんが割り勘イベントを作成しました。\n\nイベント: %s\nあなたの支払額: %d円\n支払先: %s\n\n支払いが完了したら「支払いました」と送信してください。",
				organizer.Name, req.EventName, splitAmount, organizer.Name)

			if err := PushMessage(participantID, notifyText); err != nil {
				log.Printf("通知エラー (%s): %v", participantID, err)
			} else {
				log.Printf("通知成功: %s", participantID)
			}
		}
	}()

	log.Printf("イベント作成成功: %s (ID: %d)", req.EventName, eventID)

	ctx.Success(map[string]interface{}{
		"eventId": eventID,
		"message": "イベントを作成しました",
	})
}

// ========== 承認管理API ==========

// handleApprovals は承認管理エンドポイント
func handleApprovals(ctx *APIContext) {
	switch ctx.Request.Method {
	case "GET":
		getPendingApprovals(ctx)
	case "POST":
		approvePayments(ctx)
	default:
		ctx.Error("Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getPendingApprovals は承認待ちの支払い一覧を取得
func getPendingApprovals(ctx *APIContext) {
	rows, err := db.Query(`
		SELECT ep.id, ep.event_id, ep.user_id, ep.user_name, e.event_name, e.split_amount, ep.reported_at
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE e.organizer_id = $1 AND ep.paid = true AND ep.approved_at IS NULL
		ORDER BY ep.reported_at DESC
	`, ctx.UserID)

	if err != nil {
		log.Printf("承認一覧取得エラー: %v", err)
		ctx.Error("Failed to get approvals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var approvals []map[string]interface{}
	for rows.Next() {
		var id, eventID int
		var participantID, participantName, eventName string
		var splitAmount int
		var reportedAt *string

		if err := rows.Scan(&id, &eventID, &participantID, &participantName, &eventName, &splitAmount, &reportedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}

		approvals = append(approvals, map[string]interface{}{
			"id":              id,
			"eventId":         eventID,
			"participantId":   participantID,
			"participantName": participantName,
			"eventName":       eventName,
			"amount":          splitAmount,
			"reportedAt":      reportedAt,
		})
	}

	ctx.Success(map[string]interface{}{
		"approvals": approvals,
	})
}

// approvePayments は支払いを承認
func approvePayments(ctx *APIContext) {
	var req struct {
		ParticipantIDs []int `json:"participantIds"`
	}

	if !ctx.DecodeJSON(&req) {
		return
	}

	if len(req.ParticipantIDs) == 0 {
		ctx.Error("No participants specified", http.StatusBadRequest)
		return
	}

	for _, participantID := range req.ParticipantIDs {
		var organizerID string
		err := db.QueryRow(`
			SELECT e.organizer_id
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.id
			WHERE ep.id = $1
		`, participantID).Scan(&organizerID)

		if err != nil {
			log.Printf("承認権限確認エラー: %v", err)
			continue
		}

		if organizerID != ctx.UserID {
			log.Printf("承認権限なし: %s", ctx.UserID)
			continue
		}

		_, err = db.Exec(`
			UPDATE event_participants
			SET approved_at = NOW()
			WHERE id = $1
		`, participantID)

		if err != nil {
			log.Printf("承認更新エラー: %v", err)
			continue
		}

		// 承認通知を送信（非同期）
		go func(pid int, organizerUserID string) {
			var participantUserID, participantName, eventName string
			var splitAmount int

			db.QueryRow(`
				SELECT ep.user_id, ep.user_name, e.event_name, e.split_amount
				FROM event_participants ep
				JOIN events e ON ep.event_id = e.id
				WHERE ep.id = $1
			`, pid).Scan(&participantUserID, &participantName, &eventName, &splitAmount)

			organizer, _ := getUser(organizerUserID)
			if organizer != nil {
				notifyText := fmt.Sprintf("【支払い承認】\n%sさんが支払いを承認しました。\n\nイベント: %s\n金額: %d円\n\nありがとうございました！",
					organizer.Name, eventName, splitAmount)
				PushMessage(participantUserID, notifyText)
				log.Printf("承認通知送信: %s", participantName)
			}
		}(participantID, ctx.UserID)
	}

	ctx.Success(map[string]interface{}{
		"message": "承認しました",
	})
}

// ========== サークルメンバー取得API ==========

// handleGetCircleMembers は同じサークルのメンバー一覧を取得
func handleGetCircleMembers(ctx *APIContext) {
	user, err := getUser(ctx.UserID)
	if err != nil || user == nil {
		ctx.Error("User not found", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT user_id, name, circle
		FROM users
		WHERE circle = $1 AND step = 3 AND user_id != $2
		ORDER BY name
	`, user.Circle, ctx.UserID)

	if err != nil {
		log.Printf("メンバー取得エラー: %v", err)
		ctx.Error("Failed to get members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var id, name, circle string
		if err := rows.Scan(&id, &name, &circle); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}

		members = append(members, map[string]interface{}{
			"userId": id,
			"name":   name,
			"circle": circle,
		})
	}

	ctx.Success(map[string]interface{}{
		"members": members,
	})
}
