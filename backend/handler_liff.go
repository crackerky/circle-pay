package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ========== LIFF APIハンドラー ==========

// handleRegisterUser はユーザー情報を登録
func handleRegisterUser(c *gin.Context) {
	userID := GetUserID(c)
	displayName := GetDisplayName(c)

	var req struct {
		Circle string `json:"circle" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Circle is required"})
		return
	}

	circle := sanitizeInput(req.Circle)
	if circle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Circle is required"})
		return
	}

	existingUser, err := GetUser(userID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if existingUser != nil {
		existingUser.Name = displayName
		existingUser.Circle = circle
		existingUser.Step = 3
		if err := UpdateUser(existingUser); err != nil {
			log.Printf("ユーザー更新エラー: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
		log.Printf("ユーザー更新成功: %s (%s)", displayName, userID)
	} else {
		newUser := &User{
			UserID: userID,
			Name:   displayName,
			Circle: circle,
			Step:   3,
		}
		if err := SaveUser(newUser); err != nil {
			log.Printf("ユーザー保存エラー: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
			return
		}
		log.Printf("新規ユーザー登録成功: %s (%s)", displayName, userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"userId":      userID,
		"displayName": displayName,
		"circle":      circle,
	})
}

// handleLIFFMessage はLIFFからのメッセージを処理
func handleLIFFMessage(c *gin.Context) {
	userID := GetUserID(c)

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	message := sanitizeInput(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	if err := SaveMessage(userID, message); err != nil {
		log.Printf("メッセージ保存エラー: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"reply":  "メッセージを受け取りました",
	})
}

// handleGetMyInfo はユーザー情報を取得
func handleGetMyInfo(c *gin.Context) {
	userID := GetUserID(c)
	displayName := GetDisplayName(c)

	user, err := GetUser(userID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"userId":      userID,
			"displayName": displayName,
			"registered":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"userId":      user.UserID,
		"name":        user.Name,
		"displayName": displayName,
		"circle":      user.Circle,
		"registered":  true,
		"step":        user.Step,
	})
}

// ========== イベント管理API ==========

// handleGetEvents は自分が作成したイベント一覧を取得
func handleGetEvents(c *gin.Context) {
	userID := GetUserID(c)

	events, err := GetEventsByOrganizer(userID)
	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get events"})
		return
	}

	// レスポンス形式に変換
	var response []map[string]interface{}
	for _, e := range events {
		response = append(response, map[string]interface{}{
			"id":          e.ID,
			"name":        e.Name,
			"totalAmount": e.TotalAmount,
			"splitAmount": e.SplitAmount,
			"status":      e.Status,
			"createdAt":   e.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"events": response,
	})
}

// handleCreateEvent は新しいイベントを作成
func handleCreateEvent(c *gin.Context) {
	userID := GetUserID(c)

	var req struct {
		EventName      string   `json:"eventName" binding:"required"`
		TotalAmount    int      `json:"totalAmount" binding:"required,gt=0"`
		ParticipantIDs []string `json:"participantIds" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	organizer, err := GetUser(userID)
	if err != nil || organizer == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	splitAmount := req.TotalAmount / len(req.ParticipantIDs)

	eventID, err := CreateEvent(req.EventName, userID, organizer.Circle, req.TotalAmount, splitAmount)
	if err != nil {
		log.Printf("イベント作成エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	// 参加者を登録
	for _, participantID := range req.ParticipantIDs {
		participant, err := GetUser(participantID)
		if err != nil || participant == nil {
			log.Printf("参加者取得エラー: %v", participantID)
			continue
		}

		if err := CreateParticipant(eventID, participantID, participant.Name); err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"eventId": eventID,
		"message": "イベントを作成しました",
	})
}

// ========== 承認管理API ==========

// handleGetApprovals は承認待ちの支払い一覧を取得
func handleGetApprovals(c *gin.Context) {
	userID := GetUserID(c)

	approvals, err := GetPendingApprovals(userID)
	if err != nil {
		log.Printf("承認一覧取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get approvals"})
		return
	}

	// レスポンス形式に変換
	var response []map[string]interface{}
	for _, a := range approvals {
		response = append(response, map[string]interface{}{
			"id":              a.ID,
			"eventId":         a.EventID,
			"participantId":   a.ParticipantID,
			"participantName": a.ParticipantName,
			"eventName":       a.EventName,
			"amount":          a.Amount,
			"reportedAt":      a.ReportedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"approvals": response,
	})
}

// handleApprovePayments は支払いを承認
func handleApprovePayments(c *gin.Context) {
	userID := GetUserID(c)

	var req struct {
		ParticipantIDs []int `json:"participantIds" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No participants specified"})
		return
	}

	for _, participantID := range req.ParticipantIDs {
		// 権限確認
		organizerID, err := GetParticipantOrganizerID(participantID)
		if err != nil {
			log.Printf("承認権限確認エラー: %v", err)
			continue
		}

		if organizerID != userID {
			log.Printf("承認権限なし: %s", userID)
			continue
		}

		// 承認処理
		if err := ApproveParticipant(participantID); err != nil {
			log.Printf("承認更新エラー: %v", err)
			continue
		}

		// 承認通知を送信（非同期）
		go func(pid int, organizerUserID string) {
			info, err := GetApprovalNotifyInfo(pid)
			if err != nil {
				log.Printf("承認通知情報取得エラー: %v", err)
				return
			}

			organizer, _ := GetUser(organizerUserID)
			if organizer != nil {
				notifyText := fmt.Sprintf("【支払い承認】\n%sさんが支払いを承認しました。\n\nイベント: %s\n金額: %d円\n\nありがとうございました！",
					organizer.Name, info.EventName, info.SplitAmount)
				PushMessage(info.ParticipantUserID, notifyText)
				log.Printf("承認通知送信: %s", info.ParticipantName)
			}
		}(participantID, userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "承認しました",
	})
}

// ========== サークルメンバー取得API ==========

// handleGetCircleMembers は同じサークルのメンバー一覧を取得
func handleGetCircleMembers(c *gin.Context) {
	userID := GetUserID(c)

	user, err := GetUser(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	members, err := GetUsersByCircle(user.Circle, userID)
	if err != nil {
		log.Printf("メンバー取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get members"})
		return
	}

	// レスポンス形式に変換
	var response []map[string]interface{}
	for _, m := range members {
		response = append(response, map[string]interface{}{
			"userId": m.UserID,
			"name":   m.Name,
			"circle": m.Circle,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"members": response,
	})
}
