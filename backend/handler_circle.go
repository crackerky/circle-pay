package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ========== サークル関連ハンドラー ==========

// handleGetMyCircles は自分が所属するサークル一覧を取得する
// GET /api/liff/circles
func handleGetMyCircles(c *gin.Context) {
	userID := GetUserID(c)

	circles, err := GetUserCircles(userID)
	if err != nil {
		log.Printf("サークル取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get circles"})
		return
	}

	// 主サークル情報も取得
	primaryCircle, _ := GetPrimaryCircle(userID)
	var primaryCircleID *int
	if primaryCircle != nil {
		primaryCircleID = &primaryCircle.ID
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"circles":         circles,
		"primaryCircleId": primaryCircleID,
	})
}

// handleCreateCircle は新規サークルを作成する
// POST /api/liff/circles
func handleCreateCircle(c *gin.Context) {
	userID := GetUserID(c)

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Circle name is required"})
		return
	}

	// サークル名をサニタイズ
	name := sanitizeInput(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Circle name cannot be empty"})
		return
	}

	// 既存サークルチェック
	existing, err := GetCircleByName(name)
	if err != nil {
		log.Printf("サークル確認エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check circle"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Circle with this name already exists"})
		return
	}

	// サークル作成と参加
	circle, err := CreateCircleAndJoin(name, userID)
	if err != nil {
		log.Printf("サークル作成エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create circle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "サークルを作成しました",
		"circle":  circle,
	})
}

// handleJoinCircle は既存サークルに参加する
// POST /api/liff/circles/join
func handleJoinCircle(c *gin.Context) {
	userID := GetUserID(c)

	var req struct {
		CircleName string `json:"circleName"`
		CircleID   *int   `json:"circleId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var circle *Circle
	var err error

	if req.CircleID != nil {
		// IDで参加
		circle, err = GetCircleByID(*req.CircleID)
		if err != nil || circle == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Circle not found"})
			return
		}
		err = JoinCircle(userID, *req.CircleID)
	} else if req.CircleName != "" {
		// 名前で参加
		circle, err = JoinCircleByName(userID, req.CircleName)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Circle name or ID is required"})
		return
	}

	if err != nil {
		log.Printf("サークル参加エラー: %v", err)
		if err.Error() == "already a member of this circle" {
			c.JSON(http.StatusConflict, gin.H{"error": "既にこのサークルに参加しています"})
			return
		}
		if err.Error() == "circle not found: "+req.CircleName {
			c.JSON(http.StatusNotFound, gin.H{"error": "サークルが見つかりません"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join circle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "サークルに参加しました",
		"circle":  circle,
	})
}

// handleLeaveCircle はサークルから退出する（自分で抜ける）
// POST /api/liff/circles/:id/leave
func handleLeaveCircle(c *gin.Context) {
	userID := GetUserID(c)
	circleIDStr := c.Param("id")

	circleID, err := strconv.Atoi(circleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid circle ID"})
		return
	}

	// 自分がメンバーか確認
	isMember, err := IsCircleMember(userID, circleID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this circle"})
		return
	}

	if err := LeaveCircle(userID, circleID); err != nil {
		log.Printf("サークル退出エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave circle"})
		return
	}

	// 主サークルだった場合は解除
	user, _ := GetUser(userID)
	if user != nil && user.PrimaryCircleID != nil && *user.PrimaryCircleID == circleID {
		// 他のサークルがあれば最初のものを主サークルに
		circles, _ := GetUserCircles(userID)
		if len(circles) > 0 {
			SetPrimaryCircle(userID, circles[0].ID)
		} else {
			// サークルがなくなった場合はnull
			db.Exec(`UPDATE users SET primary_circle_id = NULL WHERE user_id = $1`, userID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "サークルから退出しました",
	})
}

// handleRemoveFromCircle はメンバーをサークルから退会させる（他人を外す）
// POST /api/liff/circles/:id/remove
func handleRemoveFromCircle(c *gin.Context) {
	userID := GetUserID(c)
	circleIDStr := c.Param("id")

	circleID, err := strconv.Atoi(circleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid circle ID"})
		return
	}

	var req struct {
		TargetUserID string `json:"targetUserId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user ID is required"})
		return
	}

	// 自分がメンバーか確認（メンバーなら誰でも外せる設計）
	isMember, err := IsCircleMember(userID, circleID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this circle"})
		return
	}

	// 自分自身を外そうとした場合はエラー
	if req.TargetUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use leave endpoint to leave by yourself"})
		return
	}

	if err := RemoveFromCircle(req.TargetUserID, circleID); err != nil {
		log.Printf("メンバー退会エラー: %v", err)
		if err.Error() == "user is not a member of this circle" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this circle"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "メンバーを退会させました",
	})
}

// handleGetCircleMembers は指定サークルのメンバー一覧を取得する
// GET /api/liff/circles/:id/members
func handleGetCircleMembersByID(c *gin.Context) {
	userID := GetUserID(c)
	circleIDStr := c.Param("id")

	circleID, err := strconv.Atoi(circleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid circle ID"})
		return
	}

	// サークル情報取得
	circle, err := GetCircleByID(circleID)
	if err != nil || circle == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circle not found"})
		return
	}

	// 自分がメンバーか確認
	isMember, err := IsCircleMember(userID, circleID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this circle"})
		return
	}

	// excludeMyself パラメータで自分を除外するか決定
	excludeMyself := c.Query("excludeMyself") == "true"

	var members []CircleMember
	if excludeMyself {
		members, err = GetCircleMembers(circleID, userID)
	} else {
		members, err = GetAllCircleMembers(circleID)
	}

	if err != nil {
		log.Printf("メンバー取得エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"circle":  circle,
		"members": members,
	})
}

// handleSearchCircles はサークルを検索する
// GET /api/liff/circles/search?q=xxx
func handleSearchCircles(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	circles, err := SearchCirclesByName(query)
	if err != nil {
		log.Printf("サークル検索エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search circles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"circles": circles,
	})
}

// handleSetPrimaryCircle は主サークルを設定する
// POST /api/liff/circles/:id/primary
func handleSetPrimaryCircle(c *gin.Context) {
	userID := GetUserID(c)
	circleIDStr := c.Param("id")

	circleID, err := strconv.Atoi(circleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid circle ID"})
		return
	}

	// 自分がメンバーか確認
	isMember, err := IsCircleMember(userID, circleID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this circle"})
		return
	}

	if err := SetPrimaryCircle(userID, circleID); err != nil {
		log.Printf("主サークル設定エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set primary circle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "主サークルを設定しました",
	})
}
