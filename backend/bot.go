package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// LINE Webhook用構造体
type WebhookRequest struct {
	Events []WebhookEvent `json:"events"`
}

type WebhookEvent struct {
	Type       string  `json:"type"`
	ReplyToken string  `json:"replyToken"`
	Message    Message `json:"message"`
	Source     Source  `json:"source"`
}

type Message struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Source struct {
	UserID string `json:"userId"`
}

// Quick Reply用構造体
type QuickReplyButton struct {
	Type   string       `json:"type"`
	Action ActionObject `json:"action"`
}

type ActionObject struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Text  string `json:"text,omitempty"`
	URI   string `json:"uri,omitempty"`
}

// ========== Webhookハンドラー ==========

// validateWebhookSignature はLINE Webhookの署名を検証する
func validateWebhookSignature(body []byte, signature string) bool {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	if channelSecret == "" {
		log.Printf("警告: LINE_CHANNEL_SECRET が設定されていません")
		return false
	}

	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Webhook: LINEからメッセージ受信
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// リクエストボディを読み取る
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("リクエストボディ読み取りエラー: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 署名を検証
	signature := r.Header.Get("X-Line-Signature")
	if signature == "" {
		log.Printf("署名がありません")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !validateWebhookSignature(bodyBytes, signature) {
		log.Printf("署名検証失敗")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Printf("Webhook受信（署名検証済み）")

	// ボディを再度使えるようにする
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Webhookデコードエラー: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("イベント数: %d", len(req.Events))

	for _, event := range req.Events {
		log.Printf("イベントタイプ: %s", event.Type)

		if event.Type == "message" && event.Message.Type == "text" {
			userID := event.Source.UserID
			messageText := event.Message.Text
			replyToken := event.ReplyToken

			log.Printf("メッセージ受信: UserID=%s, Text=%s", userID, messageText)

			// メッセージをDBに保存
			messages = append(messages, ReceivedMessage{
				Timestamp: time.Now(),
				UserID:    userID,
				Text:      messageText,
			})

			// handleMessage関数を利用
			handleMessage(userID, messageText, replyToken)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// ========== メッセージ処理 ==========

// メッセージ処理のメインロジック
func handleMessage(userID, message, replyToken string) {
	message = sanitizeInput(message)

	if message == "" {
		ReplyMessage(replyToken, "中身がないです")
		return
	}

	// DBからユーザー取得
	user, err := getUser(userID)
	if err != nil {
		log.Printf("ユーザー取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}

	// 新規ユーザーの場合
	if user == nil {
		startUserRegistration(userID, replyToken)
		return
	}

	// 登録段階に応じて対応を変える
	switch user.Step {
	case 1:
		handleNameInput(user, message, replyToken)
	case 2:
		handleCircleInput(user, message, replyToken)
	case 3:
		// 登録完了後のメッセージ処理
		handleRegisteredUserMessage(user, message, replyToken)
	default:
		ReplyMessage(replyToken, "エラーが発生しました")
	}
}

// ========== ユーザー登録フロー ==========

// 新規ユーザー登録を開始
func startUserRegistration(userID, replyToken string) {
	newUser := &User{
		UserID: userID,
		Name:   "",
		Circle: "",
		Step:   1,
	}
	if err := saveUser(newUser); err != nil {
		log.Printf("ユーザー保存エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}
	ReplyMessage(replyToken, "初めまして！お名前を教えてください！")
}

// 名前入力処理
func handleNameInput(user *User, name, replyToken string) {
	user.Name = name
	user.Step = 2
	if err := updateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}
	ReplyMessage(replyToken, fmt.Sprintf("%sさんありがとうございます！\n所属しているサークル名を教えてください！", user.Name))
}

// サークル名入力処理
func handleCircleInput(user *User, circle, replyToken string) {
	user.Circle = circle
	user.Step = 3
	if err := updateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}

	// 登録完了後、メインメニューを表示
	text := fmt.Sprintf("登録完了しました！\n名前: %s\nサークル: %s\n\nこれから CirclePay をご利用いただけます！", user.Name, user.Circle)
	showMainMenu(user, replyToken, text)
}

// ========== 登録済みユーザーの処理 ==========

// 登録済みユーザーのメッセージ処理
func handleRegisteredUserMessage(user *User, message, replyToken string) {
	// 特定のコマンドに応じて処理
	switch message {
	case "💰 支払いました":
		handlePaymentReport(user, replyToken)
	case "📊 状況確認":
		showMyPaymentStatus(user, replyToken)
	case "👤 会計者になる":
		sendLIFFButton(user, replyToken)
	default:
		// コマンド一覧を表示
		showMainMenu(user, replyToken, fmt.Sprintf("こんにちは、%sさん！\n操作を選択してください：", user.Name))
	}
}

// ========== メインメニュー表示 ==========

// Quick Replyでメインメニューを表示
func showMainMenu(user *User, replyToken, messageText string) {
	buttons := []QuickReplyButton{
		{
			Type: "action",
			Action: ActionObject{
				Type:  "message",
				Label: "💰 支払いました",
				Text:  "💰 支払いました",
			},
		},
		{
			Type: "action",
			Action: ActionObject{
				Type:  "message",
				Label: "📊 状況確認",
				Text:  "📊 状況確認",
			},
		},
		{
			Type: "action",
			Action: ActionObject{
				Type:  "uri",
				Label: "👤 会計者になる",
				URI:   os.Getenv("LIFF_URL"),
			},
		},
	}

	if err := ReplyMessageWithQuickReply(replyToken, messageText, buttons); err != nil {
		log.Printf("Quick Reply送信エラー: %v", err)
		ReplyMessage(replyToken, messageText)
	}
}

// ========== 支払い報告 ==========

// 支払い報告処理
func handlePaymentReport(user *User, replyToken string) {
	// 参加中の未払いイベントを取得
	rows, err := db.Query(`
		SELECT e.id, e.event_name, e.split_amount
		FROM events e
		JOIN event_participants ep ON e.id = ep.event_id
		WHERE ep.user_id = $1 AND ep.paid = false AND e.status = 'confirmed'
		ORDER BY e.created_at DESC
		LIMIT 10
	`, user.UserID)

	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}
	defer rows.Close()

	var events []struct {
		ID     int
		Name   string
		Amount int
	}

	for rows.Next() {
		var e struct {
			ID     int
			Name   string
			Amount int
		}
		if err := rows.Scan(&e.ID, &e.Name, &e.Amount); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		ReplyMessage(replyToken, "未払いのイベントはありません")
		return
	}

	// Quick Replyでイベント選択
	buttons := []QuickReplyButton{}
	for _, e := range events {
		buttons = append(buttons, QuickReplyButton{
			Type: "action",
			Action: ActionObject{
				Type:  "message",
				Label: fmt.Sprintf("%s (%d円)", e.Name, e.Amount),
				Text:  fmt.Sprintf("支払い報告:%d", e.ID),
			},
		})
	}

	ReplyMessageWithQuickReply(replyToken, "どのイベントの支払いを報告しますか？", buttons)
}

// ========== 状況確認 ==========

// 簡易版：自分の支払い状況を表示
func showMyPaymentStatus(user *User, replyToken string) {
	rows, err := db.Query(`
		SELECT e.event_name, e.split_amount, ep.paid
		FROM events e
		JOIN event_participants ep ON e.id = ep.event_id
		WHERE ep.user_id = $1
		ORDER BY e.created_at DESC
		LIMIT 10
	`, user.UserID)

	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}
	defer rows.Close()

	var status string
	for rows.Next() {
		var name string
		var amount int
		var paid bool
		if err := rows.Scan(&name, &amount, &paid); err != nil {
			log.Printf("スキャンエラー: %v", err)
			continue
		}

		paidStatus := "✅ 支払い済み"
		if !paid {
			paidStatus = "⏳ 未払い"
		}
		status += fmt.Sprintf("・%s: %d円 %s\n", name, amount, paidStatus)
	}

	if status == "" {
		status = "参加中のイベントはありません"
	} else {
		status = "【あなたの支払い状況】\n\n" + status
	}

	ReplyMessage(replyToken, status)
}

// ========== LIFF誘導ボタン ==========

// LIFFアプリへの誘導ボタンを送信
func sendLIFFButton(user *User, replyToken string) {
	liffURL := os.Getenv("LIFF_URL")

	buttons := []QuickReplyButton{
		{
			Type: "action",
			Action: ActionObject{
				Type:  "uri",
				Label: "📝 イベント作成画面を開く",
				URI:   liffURL + "/create",
			},
		},
		{
			Type: "action",
			Action: ActionObject{
				Type:  "uri",
				Label: "✅ 承認画面を開く",
				URI:   liffURL + "/approve",
			},
		},
		{
			Type: "action",
			Action: ActionObject{
				Type:  "uri",
				Label: "📊 イベント管理画面を開く",
				URI:   liffURL + "/events",
			},
		},
	}

	if err := ReplyMessageWithQuickReply(replyToken, "会計者メニューを選択してください：", buttons); err != nil {
		log.Printf("Quick Reply送信エラー: %v", err)
		ReplyMessage(replyToken, "会計者機能を利用するには、LIFFアプリを開いてください："+liffURL)
	}
}
