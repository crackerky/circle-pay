package main

// ========== メッセージリポジトリ ==========

// SaveMessage はメッセージを保存する
func SaveMessage(userID, text string) error {
	_, err := db.Exec(`
		INSERT INTO messages (user_id, text) VALUES ($1, $2)
	`, userID, text)
	return err
}
