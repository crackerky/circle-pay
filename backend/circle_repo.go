package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ========== サークル操作 ==========

// CreateCircle はサークルを作成する
func CreateCircle(name, createdBy string) (*Circle, error) {
	var circle Circle
	err := db.QueryRow(`
		INSERT INTO circles (name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at
	`, name, createdBy).Scan(&circle.ID, &circle.Name, &circle.CreatedBy, &circle.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create circle: %w", err)
	}

	log.Printf("[サークル] 作成: %s (ID: %d, 作成者: %s)", name, circle.ID, createdBy)
	return &circle, nil
}

// GetCircleByID はIDでサークルを取得する
func GetCircleByID(circleID int) (*Circle, error) {
	var circle Circle
	err := db.QueryRow(`
		SELECT id, name, created_by, created_at
		FROM circles
		WHERE id = $1
	`, circleID).Scan(&circle.ID, &circle.Name, &circle.CreatedBy, &circle.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &circle, nil
}

// GetCircleByName は名前でサークルを取得する
func GetCircleByName(name string) (*Circle, error) {
	var circle Circle
	err := db.QueryRow(`
		SELECT id, name, created_by, created_at
		FROM circles
		WHERE name = $1
	`, name).Scan(&circle.ID, &circle.Name, &circle.CreatedBy, &circle.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &circle, nil
}

// SearchCirclesByName はサークル名で部分一致検索する
func SearchCirclesByName(query string) ([]Circle, error) {
	rows, err := db.Query(`
		SELECT id, name, created_by, created_at
		FROM circles
		WHERE name ILIKE $1
		ORDER BY name
		LIMIT 20
	`, "%"+query+"%")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var circles []Circle
	for rows.Next() {
		var c Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedBy, &c.CreatedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		circles = append(circles, c)
	}
	return circles, nil
}

// ========== ユーザーサークル関係操作 ==========

// JoinCircle はユーザーをサークルに参加させる
func JoinCircle(userID string, circleID int) error {
	// 既に参加しているか確認
	var existingID int
	var status string
	err := db.QueryRow(`
		SELECT id, status FROM user_circles
		WHERE user_id = $1 AND circle_id = $2
	`, userID, circleID).Scan(&existingID, &status)

	if err == nil {
		// 既存レコードがある場合
		if status == "active" {
			return fmt.Errorf("already a member of this circle")
		}
		// 以前退出/退会していた場合は再参加
		_, err = db.Exec(`
			UPDATE user_circles
			SET status = 'active', joined_at = NOW(), left_at = NULL
			WHERE id = $1
		`, existingID)
		if err != nil {
			return fmt.Errorf("failed to rejoin circle: %w", err)
		}
		log.Printf("[サークル] 再参加: user=%s, circle=%d", userID, circleID)
		return nil
	}

	if err != sql.ErrNoRows {
		return err
	}

	// 新規参加
	_, err = db.Exec(`
		INSERT INTO user_circles (user_id, circle_id, status)
		VALUES ($1, $2, 'active')
	`, userID, circleID)

	if err != nil {
		return fmt.Errorf("failed to join circle: %w", err)
	}

	log.Printf("[サークル] 参加: user=%s, circle=%d", userID, circleID)
	return nil
}

// LeaveCircle はユーザーがサークルから退出する（自分で抜ける）
func LeaveCircle(userID string, circleID int) error {
	result, err := db.Exec(`
		UPDATE user_circles
		SET status = 'left', left_at = NOW()
		WHERE user_id = $1 AND circle_id = $2 AND status = 'active'
	`, userID, circleID)

	if err != nil {
		return fmt.Errorf("failed to leave circle: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("not a member of this circle")
	}

	log.Printf("[サークル] 退出: user=%s, circle=%d", userID, circleID)
	return nil
}

// RemoveFromCircle はユーザーをサークルから退会させる（他人を外す）
func RemoveFromCircle(targetUserID string, circleID int) error {
	result, err := db.Exec(`
		UPDATE user_circles
		SET status = 'removed', left_at = NOW()
		WHERE user_id = $1 AND circle_id = $2 AND status = 'active'
	`, targetUserID, circleID)

	if err != nil {
		return fmt.Errorf("failed to remove from circle: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user is not a member of this circle")
	}

	log.Printf("[サークル] 退会: user=%s, circle=%d", targetUserID, circleID)
	return nil
}

// GetUserCircles はユーザーが所属するサークル一覧を取得する
func GetUserCircles(userID string) ([]Circle, error) {
	rows, err := db.Query(`
		SELECT c.id, c.name, c.created_by, c.created_at
		FROM circles c
		JOIN user_circles uc ON c.id = uc.circle_id
		WHERE uc.user_id = $1 AND uc.status = 'active'
		ORDER BY uc.joined_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var circles []Circle
	for rows.Next() {
		var c Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedBy, &c.CreatedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		circles = append(circles, c)
	}
	return circles, nil
}

// GetCircleMembers はサークルのメンバー一覧を取得する
func GetCircleMembers(circleID int, excludeUserID string) ([]CircleMember, error) {
	rows, err := db.Query(`
		SELECT u.user_id, u.name, uc.joined_at
		FROM users u
		JOIN user_circles uc ON u.user_id = uc.user_id
		WHERE uc.circle_id = $1 AND uc.status = 'active' AND u.step = 3 AND u.user_id != $2
		ORDER BY u.name
	`, circleID, excludeUserID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []CircleMember
	for rows.Next() {
		var m CircleMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.JoinedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		members = append(members, m)
	}
	return members, nil
}

// GetAllCircleMembers はサークルの全メンバー一覧を取得する（自分を含む）
func GetAllCircleMembers(circleID int) ([]CircleMember, error) {
	rows, err := db.Query(`
		SELECT u.user_id, u.name, uc.joined_at
		FROM users u
		JOIN user_circles uc ON u.user_id = uc.user_id
		WHERE uc.circle_id = $1 AND uc.status = 'active' AND u.step = 3
		ORDER BY u.name
	`, circleID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []CircleMember
	for rows.Next() {
		var m CircleMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.JoinedAt); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		members = append(members, m)
	}
	return members, nil
}

// IsCircleMember はユーザーがサークルのメンバーかどうか確認する
func IsCircleMember(userID string, circleID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM user_circles
			WHERE user_id = $1 AND circle_id = $2 AND status = 'active'
		)
	`, userID, circleID).Scan(&exists)

	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetOrCreateCircle はサークルを取得、なければ作成する
func GetOrCreateCircle(name, createdBy string) (*Circle, error) {
	circle, err := GetCircleByName(name)
	if err != nil {
		return nil, err
	}
	if circle != nil {
		return circle, nil
	}
	return CreateCircle(name, createdBy)
}

// SetPrimaryCircle はユーザーの主サークルを設定する
func SetPrimaryCircle(userID string, circleID int) error {
	_, err := db.Exec(`
		UPDATE users SET primary_circle_id = $1, updated_at = NOW()
		WHERE user_id = $2
	`, circleID, userID)
	return err
}

// GetUserCircleStatus はユーザーのサークル参加状況を取得する
func GetUserCircleStatus(userID string, circleID int) (*UserCircle, error) {
	var uc UserCircle
	var leftAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, circle_id, status, joined_at, left_at
		FROM user_circles
		WHERE user_id = $1 AND circle_id = $2
	`, userID, circleID).Scan(&uc.ID, &uc.UserID, &uc.CircleID, &uc.Status, &uc.JoinedAt, &leftAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if leftAt.Valid {
		uc.LeftAt = &leftAt.Time
	}

	return &uc, nil
}

// GetCircleMemberCount はサークルのアクティブメンバー数を取得する
func GetCircleMemberCount(circleID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM user_circles
		WHERE circle_id = $1 AND status = 'active'
	`, circleID).Scan(&count)
	return count, err
}

// ========== レガシー互換性 ==========

// GetUsersByCircleLegacy は旧circle名でメンバーを取得する（後方互換性）
func GetUsersByCircleLegacy(circleName string, excludeUserID string) ([]User, error) {
	// まずcirclesテーブルからIDを取得
	circle, err := GetCircleByName(circleName)
	if err != nil || circle == nil {
		// 旧方式にフォールバック
		return GetUsersByCircleOld(circleName, excludeUserID)
	}

	// 新方式で取得
	rows, err := db.Query(`
		SELECT u.user_id, u.name, COALESCE(u.circle, '') as circle
		FROM users u
		JOIN user_circles uc ON u.user_id = uc.user_id
		WHERE uc.circle_id = $1 AND uc.status = 'active' AND u.step = 3 AND u.user_id != $2
		ORDER BY u.name
	`, circle.ID, excludeUserID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.UserID, &u.Name, &u.Circle); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// GetUsersByCircleOld は旧方式でメンバーを取得する
func GetUsersByCircleOld(circleName string, excludeUserID string) ([]User, error) {
	rows, err := db.Query(`
		SELECT user_id, name, circle
		FROM users
		WHERE circle = $1 AND step = 3 AND user_id != $2
		ORDER BY name
	`, circleName, excludeUserID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.UserID, &u.Name, &u.Circle); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// ========== ユーティリティ ==========

// GetPrimaryCircle はユーザーの主サークルを取得する
func GetPrimaryCircle(userID string) (*Circle, error) {
	var circle Circle
	err := db.QueryRow(`
		SELECT c.id, c.name, c.created_by, c.created_at
		FROM circles c
		JOIN users u ON c.id = u.primary_circle_id
		WHERE u.user_id = $1
	`, userID).Scan(&circle.ID, &circle.Name, &circle.CreatedBy, &circle.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &circle, nil
}

// CreateCircleAndJoin はサークルを作成してユーザーを参加させる
func CreateCircleAndJoin(name, userID string) (*Circle, error) {
	circle, err := CreateCircle(name, userID)
	if err != nil {
		return nil, err
	}

	if err := JoinCircle(userID, circle.ID); err != nil {
		return nil, err
	}

	if err := SetPrimaryCircle(userID, circle.ID); err != nil {
		return nil, err
	}

	return circle, nil
}

// JoinCircleByName はサークル名でサークルに参加する
func JoinCircleByName(userID, circleName string) (*Circle, error) {
	circle, err := GetCircleByName(circleName)
	if err != nil {
		return nil, err
	}
	if circle == nil {
		return nil, fmt.Errorf("circle not found: %s", circleName)
	}

	if err := JoinCircle(userID, circle.ID); err != nil {
		return nil, err
	}

	// 主サークルが設定されていなければ設定
	user, err := GetUser(userID)
	if err == nil && user != nil && user.PrimaryCircleID == nil {
		SetPrimaryCircle(userID, circle.ID)
	}

	return circle, nil
}

// dummy for time import
var _ = time.Now
