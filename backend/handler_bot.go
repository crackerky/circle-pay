package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ========== Webhookハンドラー ==========

// handleWebhook はLINE Webhookを処理（署名検証はミドルウェアで実施済み）
func handleWebhook(c *gin.Context) {
	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Webhookデコードエラー: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	log.Printf("Webhook受信（署名検証済み）")
	log.Printf("イベント数: %d", len(req.Events))

	for _, event := range req.Events {
		log.Printf("イベントタイプ: %s", event.Type)

		if event.Type == "message" && event.Message.Type == "text" {
			userID := event.Source.UserID
			messageText := event.Message.Text
			replyToken := event.ReplyToken

			log.Printf("メッセージ受信: UserID=%s", userID)

			// メッセージを記録（スレッドセーフ）
			addReceivedMessage(ReceivedMessage{
				Timestamp: time.Now(),
				UserID:    userID,
				Text:      messageText,
			})

			// handleMessage関数を利用
			handleMessage(userID, messageText, replyToken)
		}
	}

	c.Status(http.StatusOK)
}

// ========== メッセージ処理 ==========

// handleMessage はメッセージ処理のメインロジック
func handleMessage(userID, message, replyToken string) {
	message = sanitizeInput(message)

	if message == "" {
		ReplyMessage(replyToken, "中身がないです")
		return
	}

	// DBからユーザー取得
	user, err := GetUser(userID)
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

// startUserRegistration は新規ユーザー登録を開始
func startUserRegistration(userID, replyToken string) {
	newUser := &User{
		UserID: userID,
		Name:   "",
		Circle: "",
		Step:   1,
	}
	if err := SaveUser(newUser); err != nil {
		log.Printf("ユーザー保存エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}
	ReplyMessage(replyToken, "初めまして！お名前を教えてください！")
}

// handleNameInput は名前入力処理
func handleNameInput(user *User, name, replyToken string) {
	user.Name = name
	user.Step = 2
	if err := UpdateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度お試しください。")
		return
	}

	// サークル作成/参加の選択肢を表示
	buttons := []QuickReplyButton{
		{
			Type: "action",
			Action: ActionObject{
				Type:  "message",
				Label: "🆕 新規作成",
				Text:  "サークル:新規作成",
			},
		},
		{
			Type: "action",
			Action: ActionObject{
				Type:  "message",
				Label: "🔍 既存に参加",
				Text:  "サークル:既存参加",
			},
		},
	}

	msg := fmt.Sprintf("%sさんありがとうございます！\n\nサークルを新規作成しますか？\nそれとも既存のサークルに参加しますか？", user.Name)
	if err := ReplyMessageWithQuickReply(replyToken, msg, buttons); err != nil {
		log.Printf("Quick Reply送信エラー: %v", err)
		ReplyMessage(replyToken, msg+"\n\n「サークル:新規作成」または「サークル:既存参加」と入力してください。")
	}
}

// handleCircleInput はサークル名入力処理
func handleCircleInput(user *User, message, replyToken string) {
	// サークル作成/参加の選択をハンドリング
	switch message {
	case "サークル:新規作成":
		user.SplitEventStep = 1 // 新規作成モード
		if err := UpdateUser(user); err != nil {
			log.Printf("ユーザー更新エラー: %v", err)
			ReplyMessage(replyToken, "エラーが発生しました。")
			return
		}
		ReplyMessage(replyToken, "新しいサークルを作成します！\nサークル名を入力してください：")
		return

	case "サークル:既存参加":
		user.SplitEventStep = 2 // 既存参加モード
		if err := UpdateUser(user); err != nil {
			log.Printf("ユーザー更新エラー: %v", err)
			ReplyMessage(replyToken, "エラーが発生しました。")
			return
		}
		ReplyMessage(replyToken, "参加するサークル名を入力してください：\n（サークル名は完全一致で検索されます）")
		return
	}

	// サークル名の入力をハンドリング
	if user.SplitEventStep == 1 {
		// 新規作成モード
		handleCircleCreate(user, message, replyToken)
	} else if user.SplitEventStep == 2 {
		// 既存参加モード
		handleCircleJoin(user, message, replyToken)
	} else {
		// モードが設定されていない場合（レガシー互換）
		handleCircleLegacy(user, message, replyToken)
	}
}

// handleCircleCreate はサークル新規作成処理
func handleCircleCreate(user *User, circleName, replyToken string) {
	// 既存サークルチェック
	existing, _ := GetCircleByName(circleName)
	if existing != nil {
		ReplyMessage(replyToken, "このサークル名は既に使用されています。\n別の名前を入力するか、「サークル:既存参加」と入力して参加してください。")
		return
	}

	// サークル作成と参加
	circle, err := CreateCircleAndJoin(circleName, user.UserID)
	if err != nil {
		log.Printf("サークル作成エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。もう一度お試しください。")
		return
	}

	// ユーザー情報を更新
	user.Circle = circleName
	user.PrimaryCircleID = &circle.ID
	user.Step = 3
	user.SplitEventStep = 0
	if err := UpdateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
	}

	text := fmt.Sprintf("登録完了しました！\n\n名前: %s\nサークル: %s（新規作成）\n\nこれから CirclePay をご利用いただけます！", user.Name, circleName)
	showMainMenu(user, replyToken, text)
}

// handleCircleJoin は既存サークル参加処理
func handleCircleJoin(user *User, circleName, replyToken string) {
	// サークル検索
	circle, err := GetCircleByName(circleName)
	if err != nil {
		log.Printf("サークル検索エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。もう一度お試しください。")
		return
	}

	if circle == nil {
		// 部分一致で候補を検索
		candidates, _ := SearchCirclesByName(circleName)
		if len(candidates) > 0 {
			var suggestion string
			for i, c := range candidates {
				if i >= 5 {
					suggestion += "...\n"
					break
				}
				suggestion += fmt.Sprintf("・%s\n", c.Name)
			}
			ReplyMessage(replyToken, fmt.Sprintf("「%s」というサークルは見つかりませんでした。\n\n似た名前のサークル：\n%s\n正確なサークル名を入力してください。", circleName, suggestion))
		} else {
			ReplyMessage(replyToken, fmt.Sprintf("「%s」というサークルは見つかりませんでした。\n\n正確なサークル名を入力するか、「サークル:新規作成」と入力して新しく作成してください。", circleName))
		}
		return
	}

	// サークルに参加
	if err := JoinCircle(user.UserID, circle.ID); err != nil {
		if err.Error() == "already a member of this circle" {
			ReplyMessage(replyToken, "既にこのサークルに参加しています。")
		} else {
			log.Printf("サークル参加エラー: %v", err)
			ReplyMessage(replyToken, "エラーが発生しました。もう一度お試しください。")
		}
		return
	}

	// ユーザー情報を更新
	user.Circle = circleName
	user.PrimaryCircleID = &circle.ID
	user.Step = 3
	user.SplitEventStep = 0
	if err := UpdateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
	}

	// メンバー数を取得
	memberCount, _ := GetCircleMemberCount(circle.ID)

	text := fmt.Sprintf("登録完了しました！\n\n名前: %s\nサークル: %s（%d人参加中）\n\nこれから CirclePay をご利用いただけます！", user.Name, circleName, memberCount)
	showMainMenu(user, replyToken, text)
}

// handleCircleLegacy はレガシー互換のサークル処理（直接サークル名入力）
func handleCircleLegacy(user *User, circleName, replyToken string) {
	// サークルを取得または作成
	circle, err := GetOrCreateCircle(circleName, user.UserID)
	if err != nil {
		log.Printf("サークル取得/作成エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。")
		return
	}

	// サークルに参加
	JoinCircle(user.UserID, circle.ID)

	// ユーザー情報を更新
	user.Circle = circleName
	user.PrimaryCircleID = &circle.ID
	user.Step = 3
	if err := UpdateUser(user); err != nil {
		log.Printf("ユーザー更新エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました。")
		return
	}

	text := fmt.Sprintf("登録完了しました！\n名前: %s\nサークル: %s\n\nこれから CirclePay をご利用いただけます！", user.Name, user.Circle)
	showMainMenu(user, replyToken, text)
}

// ========== 登録済みユーザーの処理 ==========

// handleRegisteredUserMessage は登録済みユーザーのメッセージ処理
func handleRegisteredUserMessage(user *User, message, replyToken string) {
	// 支払い報告のハンドリング
	if strings.HasPrefix(message, "支払い報告:") {
		eventIDStr := strings.TrimPrefix(message, "支払い報告:")
		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil {
			ReplyMessage(replyToken, "無効なイベントIDです")
			return
		}
		handlePaymentConfirm(user, eventID, replyToken)
		return
	}

	// サークル参加のハンドリング
	if strings.HasPrefix(message, "サークル参加:") {
		circleName := strings.TrimPrefix(message, "サークル参加:")
		handleAdditionalCircleJoin(user, circleName, replyToken)
		return
	}

	// 特定のコマンドに応じて処理
	switch message {
	case "💰 支払いました":
		handlePaymentReport(user, replyToken)
	case "📊 状況確認":
		showMyPaymentStatus(user, replyToken)
	case "👤 会計者になる":
		sendLIFFButton(user, replyToken)
	case "🔄 サークル追加":
		showCircleAddMenu(user, replyToken)
	case "📋 サークル一覧":
		showUserCircles(user, replyToken)
	default:
		// コマンド一覧を表示
		showMainMenu(user, replyToken, fmt.Sprintf("こんにちは、%sさん！\n操作を選択してください：", user.Name))
	}
}

// ========== サークル管理（登録済みユーザー向け） ==========

// showCircleAddMenu はサークル追加メニューを表示
func showCircleAddMenu(user *User, replyToken string) {
	ReplyMessage(replyToken, "追加で参加するサークル名を入力してください：\n\n（「サークル参加:〇〇」の形式で送信）\n例: サークル参加:テニスサークル")
}

// showUserCircles はユーザーの所属サークル一覧を表示
func showUserCircles(user *User, replyToken string) {
	circles, err := GetUserCircles(user.UserID)
	if err != nil {
		log.Printf("サークル取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}

	if len(circles) == 0 {
		ReplyMessage(replyToken, "所属しているサークルはありません")
		return
	}

	var text string
	for i, c := range circles {
		memberCount, _ := GetCircleMemberCount(c.ID)
		primary := ""
		if user.PrimaryCircleID != nil && *user.PrimaryCircleID == c.ID {
			primary = " ⭐"
		}
		text += fmt.Sprintf("%d. %s (%d人)%s\n", i+1, c.Name, memberCount, primary)
	}

	ReplyMessage(replyToken, fmt.Sprintf("【所属サークル一覧】\n\n%s\n⭐ = メインサークル\n\nサークルの管理はLIFFアプリから行えます。", text))
}

// handleAdditionalCircleJoin は追加サークル参加処理
func handleAdditionalCircleJoin(user *User, circleName, replyToken string) {
	circleName = sanitizeInput(circleName)
	if circleName == "" {
		ReplyMessage(replyToken, "サークル名を入力してください")
		return
	}

	circle, err := GetCircleByName(circleName)
	if err != nil {
		log.Printf("サークル検索エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}

	if circle == nil {
		// 候補を検索
		candidates, _ := SearchCirclesByName(circleName)
		if len(candidates) > 0 {
			var suggestion string
			for i, c := range candidates {
				if i >= 5 {
					break
				}
				suggestion += fmt.Sprintf("・%s\n", c.Name)
			}
			ReplyMessage(replyToken, fmt.Sprintf("「%s」は見つかりませんでした。\n\n似た名前：\n%s", circleName, suggestion))
		} else {
			ReplyMessage(replyToken, fmt.Sprintf("「%s」というサークルは見つかりませんでした。", circleName))
		}
		return
	}

	if err := JoinCircle(user.UserID, circle.ID); err != nil {
		if err.Error() == "already a member of this circle" {
			ReplyMessage(replyToken, "既にこのサークルに参加しています。")
		} else {
			log.Printf("サークル参加エラー: %v", err)
			ReplyMessage(replyToken, "エラーが発生しました")
		}
		return
	}

	memberCount, _ := GetCircleMemberCount(circle.ID)
	ReplyMessage(replyToken, fmt.Sprintf("「%s」に参加しました！（%d人参加中）", circleName, memberCount))
}

// ========== メインメニュー表示 ==========

// showMainMenu はQuick Replyでメインメニューを表示
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
				Type:  "message",
				Label: "📋 サークル一覧",
				Text:  "📋 サークル一覧",
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

// handlePaymentReport は支払い報告処理
func handlePaymentReport(user *User, replyToken string) {
	events, err := GetUnpaidEventsForUser(user.UserID)
	if err != nil {
		log.Printf("イベント取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
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

// handlePaymentConfirm は支払い確定処理
func handlePaymentConfirm(user *User, eventID int, replyToken string) {
	if err := ReportPayment(eventID, user.UserID); err != nil {
		log.Printf("支払い報告エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}

	// イベント情報を取得して会計者に通知
	event, err := GetEvent(eventID)
	if err != nil || event == nil {
		log.Printf("イベント取得エラー: %v", err)
		ReplyMessage(replyToken, "支払いを報告しました！会計者の承認をお待ちください。")
		return
	}

	// 会計者に通知（非同期）
	go func() {
		notifyText := fmt.Sprintf("💰 支払い報告\n\n%sさんが「%s」の支払いを報告しました。\n\n承認画面から確認してください。",
			user.Name, event.EventName)
		PushMessage(event.OrganizerID, notifyText)
	}()

	ReplyMessage(replyToken, "支払いを報告しました！会計者の承認をお待ちください。")
}

// ========== 状況確認 ==========

// showMyPaymentStatus は自分の支払い状況を表示
func showMyPaymentStatus(user *User, replyToken string) {
	statuses, err := GetUserPaymentStatus(user.UserID)
	if err != nil {
		log.Printf("ステータス取得エラー: %v", err)
		ReplyMessage(replyToken, "エラーが発生しました")
		return
	}

	if len(statuses) == 0 {
		ReplyMessage(replyToken, "参加中のイベントはありません")
		return
	}

	var status string
	for _, s := range statuses {
		paidStatus := "✅ 支払い済み"
		if !s.Paid {
			paidStatus = "⏳ 未払い"
		}
		status += fmt.Sprintf("・%s: %d円 %s\n", s.EventName, s.Amount, paidStatus)
	}

	ReplyMessage(replyToken, "【あなたの支払い状況】\n\n"+status)
}

// ========== LIFF誘導ボタン ==========

// sendLIFFButton はLIFFアプリへの誘導ボタンを送信
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
