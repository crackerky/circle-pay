package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ========== Strategy Pattern: メッセージ送信システム ==========

// MessageContent はメッセージの内容を表すインターフェース（何を送るか）
type MessageContent interface {
	Build() map[string]interface{}
}

// DeliveryStrategy は送信方式を表すインターフェース（どこに送るか）
type DeliveryStrategy interface {
	Endpoint() string
	WrapPayload(messages []map[string]interface{}) map[string]interface{}
}

// ========== MessageContent 実装 ==========

// TextContent はテキストメッセージ
type TextContent struct {
	Text string
}

func (c TextContent) Build() map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": c.Text,
	}
}

// QuickReplyContent はQuickReply付きテキストメッセージ
type QuickReplyContent struct {
	Text    string
	Buttons []QuickReplyButton
}

func (c QuickReplyContent) Build() map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": c.Text,
		"quickReply": map[string]interface{}{
			"items": c.Buttons,
		},
	}
}

// ========== DeliveryStrategy 実装 ==========

// ReplyDelivery はReply API用の送信方式
type ReplyDelivery struct {
	ReplyToken string
}

func (d ReplyDelivery) Endpoint() string {
	return "https://api.line.me/v2/bot/message/reply"
}

func (d ReplyDelivery) WrapPayload(messages []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"replyToken": d.ReplyToken,
		"messages":   messages,
	}
}

// PushDelivery はPush API用の送信方式
type PushDelivery struct {
	UserID string
}

func (d PushDelivery) Endpoint() string {
	return "https://api.line.me/v2/bot/message/push"
}

func (d PushDelivery) WrapPayload(messages []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"to":       d.UserID,
		"messages": messages,
	}
}

// MulticastDelivery は複数ユーザーへの一斉送信用
type MulticastDelivery struct {
	UserIDs []string
}

func (d MulticastDelivery) Endpoint() string {
	return "https://api.line.me/v2/bot/message/multicast"
}

func (d MulticastDelivery) WrapPayload(messages []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"to":       d.UserIDs,
		"messages": messages,
	}
}

// BroadcastDelivery は全ユーザーへの送信用
type BroadcastDelivery struct{}

func (d BroadcastDelivery) Endpoint() string {
	return "https://api.line.me/v2/bot/message/broadcast"
}

func (d BroadcastDelivery) WrapPayload(messages []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"messages": messages,
	}
}

// ========== 統一送信関数 ==========

// SendMessage は送信方式とメッセージ内容を組み合わせて送信する
func SendMessage(delivery DeliveryStrategy, contents ...MessageContent) error {
	// メッセージ内容をビルド
	messages := make([]map[string]interface{}, len(contents))
	for i, content := range contents {
		messages[i] = content.Build()
	}

	// ペイロードを構築
	payload := delivery.WrapPayload(messages)

	// JSON化
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %w", err)
	}

	// HTTPリクエストを作成
	req, err := http.NewRequest("POST", delivery.Endpoint(), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	// リクエストを実行
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスを検証
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LINE API error: %s", string(body))
	}

	return nil
}

// ========== 便利関数 ==========

// ReplyMessage はシンプルなテキストメッセージで返信
func ReplyMessage(replyToken, text string) error {
	return SendMessage(ReplyDelivery{replyToken}, TextContent{text})
}

// ReplyMessageWithQuickReply はQuick Replyボタン付きメッセージで返信
func ReplyMessageWithQuickReply(replyToken, text string, buttons []QuickReplyButton) error {
	return SendMessage(ReplyDelivery{replyToken}, QuickReplyContent{text, buttons})
}

// PushMessage はユーザーにメッセージをプッシュ送信
func PushMessage(userID, text string) error {
	return SendMessage(PushDelivery{userID}, TextContent{text})
}

// PushMessageWithQuickReply はQuickReply付きプッシュメッセージを送信
func PushMessageWithQuickReply(userID, text string, buttons []QuickReplyButton) error {
	return SendMessage(PushDelivery{userID}, QuickReplyContent{text, buttons})
}

// MulticastMessage は複数ユーザーにメッセージを一斉送信
func MulticastMessage(userIDs []string, text string) error {
	return SendMessage(MulticastDelivery{userIDs}, TextContent{text})
}

// BroadcastMessage は全ユーザーにメッセージを送信
func BroadcastMessage(text string) error {
	return SendMessage(BroadcastDelivery{}, TextContent{text})
}
