package main

import (
	"fmt"
	"log"
	"time"
)

// ========== 催促システム ==========

// sendReminderToUnpaidUsers は未払いユーザーに催促メッセージを送信
func sendReminderToUnpaidUsers() {
	log.Println("[催促システム] 未払いユーザーの確認を開始...")

	participants, err := GetUnpaidParticipants()
	if err != nil {
		log.Printf("[催促システム] エラー: %v", err)
		return
	}

	if len(participants) == 0 {
		log.Println("[催促システム] 未払いユーザーはいません")
		return
	}

	log.Printf("[催促システム] %d人の未払いユーザーに催促を送信します", len(participants))

	for _, p := range participants {
		message := fmt.Sprintf(
			"⏰ お支払いの催促\n\n"+
				"【イベント】%s\n"+
				"【金額】%s円\n\n"+
				"まだお支払いが確認できていません。\n"+
				"お支払い済みの場合は「💰 支払いました」ボタンから報告してください。",
			p.EventName,
			formatAmount(p.SplitAmount),
		)

		go func(userID, msg string) {
			if err := PushMessage(userID, msg); err != nil {
				log.Printf("[催促システム] 送信失敗 (UserID: %s): %v", userID, err)
			} else {
				log.Printf("[催促システム] 送信成功: %s", userID)
			}
		}(p.UserID, message)

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[催促システム] 催促メッセージの送信処理を完了しました")
}

// startReminderScheduler は毎日12時に催促を実行するスケジューラー
func startReminderScheduler() {
	go func() {
		log.Println("[催促システム] スケジューラーを起動しました")

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			duration := next.Sub(now)
			log.Printf("[催促システム] 次回実行: %s (%s後)", next.Format("2006-01-02 15:04:05"), duration.Round(time.Second))

			time.Sleep(duration)
			sendReminderToUnpaidUsers()
		}
	}()
}
