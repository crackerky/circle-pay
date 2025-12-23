package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// ========== LIFF認証関数 ==========

// LIFFアクセストークンを検証してユーザーIDを取得
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

// HTTPリクエストからLIFFトークンを検証し、userIDを取得
func authenticateRequest(r *http.Request) (string, string, error) {
	// Authorizationヘッダーからトークンを取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", fmt.Errorf("authorization header is missing")
	}

	// "Bearer "プレフィックスを削除
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", "", fmt.Errorf("invalid authorization header format")
	}

	// LIFFトークンを検証
	userID, displayName, err := verifyLIFFToken(token)
	if err != nil {
		return "", "", err
	}

	return userID, displayName, nil
}

// ========== LIFF APIハンドラー ==========

// ユーザー情報登録（LIFF認証付き）
func handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	// LIFFトークンを検証してuserIDを取得
	userID, displayName, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// リクエストボディを読み取り
	var req struct {
		Circle string `json:"circle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	circle := sanitizeInput(req.Circle)
	if circle == "" {
		http.Error(w, "Circle is required", http.StatusBadRequest)
		return
	}

	// 既存ユーザーをチェック
	existingUser, err := getUser(userID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		// 既存ユーザーの場合は更新
		existingUser.Name = displayName
		existingUser.Circle = circle
		existingUser.Step = 3 // 登録完了
		if err := updateUser(existingUser); err != nil {
			log.Printf("ユーザー更新エラー: %v", err)
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}
		log.Printf("ユーザー更新成功: %s (%s)", displayName, userID)
	} else {
		// 新規ユーザーの場合は作成
		newUser := &User{
			UserID: userID,
			Name:   displayName,
			Circle: circle,
			Step:   3, // 登録完了
		}
		if err := saveUser(newUser); err != nil {
			log.Printf("ユーザー保存エラー: %v", err)
			http.Error(w, "Failed to save user", http.StatusInternalServerError)
			return
		}
		log.Printf("新規ユーザー登録成功: %s (%s)", displayName, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"userId":      userID,
		"displayName": displayName,
		"circle":      circle,
	})
}

// メッセージ送信（LIFF認証付き）
func handleLIFFMessage(w http.ResponseWriter, r *http.Request) {
	// LIFFトークンを検証してuserIDを取得
	userID, _, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// リクエストボディを読み取り
	var req struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	message := sanitizeInput(req.Message)
	if message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// メッセージをDBに保存
	if err := saveMessage(userID, message); err != nil {
		log.Printf("メッセージ保存エラー: %v", err)
	}

	// TODO: メッセージ処理ロジック（replyTokenなしの場合）
	replyText := "メッセージを受け取りました"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"reply":  replyText,
	})
}

// ユーザー情報取得（LIFF認証付き）
func handleGetMyInfo(w http.ResponseWriter, r *http.Request) {
	// LIFFトークンを検証してuserIDを取得
	userID, displayName, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// DBからユーザー情報を取得
	user, err := getUser(userID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		// 未登録ユーザー
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"userId":      userID,
			"displayName": displayName,
			"registered":  false,
		})
		return
	}

	// 登録済みユーザー
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"userId":      user.UserID,
		"name":        user.Name,
		"displayName": displayName,
		"circle":      user.Circle,
		"registered":  true,
		"step":        user.Step,
	})
}

// ========== イベント管理API（新規） ==========

// イベント管理エンドポイント（GET: 一覧取得、POST: 作成）
func handleEvents(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		// 自分が作成したイベント一覧を取得
		getMyEvents(w, userID)
	case "POST":
		// 新しいイベントを作成
		createEvent(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 自分が作成したイベント一覧を取得
func getMyEvents(w http.ResponseWriter, userID string) {
	rows, err := db.Query(`
		SELECT id, event_name, total_amount, split_amount, status, created_at
		FROM events
		WHERE organizer_id = $1
		ORDER BY created_at DESC
	`, userID)

	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		http.Error(w, "Failed to get events", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"events": events,
	})
}

// 新しいイベントを作成
func createEvent(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		EventName     string   `json:"eventName"`
		TotalAmount   int      `json:"totalAmount"`
		ParticipantIDs []string `json:"participantIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// バリデーション
	if req.EventName == "" {
		http.Error(w, "Event name is required", http.StatusBadRequest)
		return
	}
	if req.TotalAmount <= 0 {
		http.Error(w, "Total amount must be positive", http.StatusBadRequest)
		return
	}
	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "At least one participant is required", http.StatusBadRequest)
		return
	}

	// ユーザー情報を取得
	organizer, err := getUser(userID)
	if err != nil || organizer == nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// 割り勘金額を計算
	splitAmount := req.TotalAmount / len(req.ParticipantIDs)

	// イベントを作成
	var eventID int
	err = db.QueryRow(`
		INSERT INTO events (event_name, organizer_id, circle, total_amount, split_amount, status)
		VALUES ($1, $2, $3, $4, $5, 'confirmed')
		RETURNING id
	`, req.EventName, userID, organizer.Circle, req.TotalAmount, splitAmount).Scan(&eventID)

	if err != nil {
		log.Printf("イベント作成エラー: %v", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"eventId": eventID,
		"message": "イベントを作成しました",
	})
}

// ========== 承認管理API（新規） ==========

// 承認管理エンドポイント
func handleApprovals(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		// 承認待ちの支払い一覧を取得
		getPendingApprovals(w, userID)
	case "POST":
		// 支払いを承認
		approvePayments(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 承認待ちの支払い一覧を取得
func getPendingApprovals(w http.ResponseWriter, userID string) {
	rows, err := db.Query(`
		SELECT ep.id, ep.event_id, ep.user_id, ep.user_name, e.event_name, e.split_amount, ep.reported_at
		FROM event_participants ep
		JOIN events e ON ep.event_id = e.id
		WHERE e.organizer_id = $1 AND ep.paid = true AND ep.approved_at IS NULL
		ORDER BY ep.reported_at DESC
	`, userID)

	if err != nil {
		log.Printf("承認一覧取得エラー: %v", err)
		http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"approvals": approvals,
	})
}

// 支払いを承認
func approvePayments(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		ParticipantIDs []int `json:"participantIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSONデコードエラー: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.ParticipantIDs) == 0 {
		http.Error(w, "No participants specified", http.StatusBadRequest)
		return
	}

	// 承認処理
	for _, participantID := range req.ParticipantIDs {
		// 承認権限を確認
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

		if organizerID != userID {
			log.Printf("承認権限なし: %s", userID)
			continue
		}

		// 承認実行
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
		go func(pid int) {
			var participantUserID, participantName, eventName string
			var splitAmount int

			db.QueryRow(`
				SELECT ep.user_id, ep.user_name, e.event_name, e.split_amount
				FROM event_participants ep
				JOIN events e ON ep.event_id = e.id
				WHERE ep.id = $1
			`, pid).Scan(&participantUserID, &participantName, &eventName, &splitAmount)

			organizer, _ := getUser(userID)
			if organizer != nil {
				notifyText := fmt.Sprintf("【支払い承認】\n%sさんが支払いを承認しました。\n\nイベント: %s\n金額: %d円\n\nありがとうございました！",
					organizer.Name, eventName, splitAmount)
				PushMessage(participantUserID, notifyText)
				log.Printf("承認通知送信: %s", participantName)
			}
		}(participantID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "承認しました",
	})
}

// ========== サークルメンバー取得API（新規） ==========

// 同じサークルのメンバー一覧を取得
func handleGetCircleMembers(w http.ResponseWriter, r *http.Request) {
	userID, _, err := authenticateRequest(r)
	if err != nil {
		log.Printf("認証エラー: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 自分のサークルを取得
	user, err := getUser(userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// 同じサークルのメンバーを取得
	rows, err := db.Query(`
		SELECT user_id, name, circle
		FROM users
		WHERE circle = $1 AND step = 3 AND user_id != $2
		ORDER BY name
	`, user.Circle, userID)

	if err != nil {
		log.Printf("メンバー取得エラー: %v", err)
		http.Error(w, "Failed to get members", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"members": members,
	})
}
