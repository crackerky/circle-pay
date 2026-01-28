package main

import "time"

// ========== ドメインモデル ==========

// User はユーザー情報を管理する構造体
type User struct {
	UserID          string
	Name            string
	Circle          string // レガシー: 後方互換性のため残す
	PrimaryCircleID *int   // 主サークルID
	Step            int    // 0:未登録 1:名前待ち 2:サークル選択待ち 3:完了
	SplitEventStep  int    // 0:なし 1:イベント名待ち 2:金額待ち 3:参加者選択待ち
	TempEventID     int    // 作成中のイベントID
	ApprovalStep    int    // 0:なし 1:承認番号待ち
	ApprovalEventID int    // 承認中のイベントID
}

// Circle はサークル情報を管理する構造体
type Circle struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserCircle はユーザーとサークルの関係を管理する構造体
type UserCircle struct {
	ID       int        `json:"id"`
	UserID   string     `json:"userId"`
	CircleID int        `json:"circleId"`
	Status   string     `json:"status"` // 'active', 'left', 'removed'
	JoinedAt time.Time  `json:"joinedAt"`
	LeftAt   *time.Time `json:"leftAt,omitempty"`
}

// CircleMember はサークルメンバー情報（API用）
type CircleMember struct {
	UserID   string    `json:"userId"`
	Name     string    `json:"name"`
	JoinedAt time.Time `json:"joinedAt"`
}

// Event は割り勘イベント情報を管理する構造体
type Event struct {
	ID          int
	EventName   string
	OrganizerID string
	Circle      string // レガシー: 後方互換性のため残す
	CircleID    *int   // サークルID
	TotalAmount int
	SplitAmount int
	Status      string // 'selecting' / 'confirmed' / 'completed'
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Participant はイベント参加者情報を管理する構造体
type Participant struct {
	ID         int
	EventID    int
	UserID     string
	UserName   string
	Paid       bool
	ReportedAt *time.Time
	ApprovedAt *time.Time
	CreatedAt  time.Time
}

// UnpaidParticipant は未払い参加者情報（催促用）
type UnpaidParticipant struct {
	UserID      string
	EventID     int
	EventName   string
	UserName    string
	SplitAmount int
	CreatedAt   time.Time
}

// ReceivedMessage は受信メッセージの記録
type ReceivedMessage struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userID"`
	Text      string    `json:"text"`
}

// ========== LINE Webhook構造体 ==========

// WebhookRequest はLINE Webhookリクエスト
type WebhookRequest struct {
	Events []WebhookEvent `json:"events"`
}

// WebhookEvent はWebhookイベント
type WebhookEvent struct {
	Type       string  `json:"type"`
	ReplyToken string  `json:"replyToken"`
	Message    Message `json:"message"`
	Source     Source  `json:"source"`
}

// Message はメッセージ内容
type Message struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Source はメッセージ送信元
type Source struct {
	UserID string `json:"userId"`
}

// ========== Quick Reply構造体 ==========

// QuickReplyButton はQuick Replyボタン
type QuickReplyButton struct {
	Type   string       `json:"type"`
	Action ActionObject `json:"action"`
}

// ActionObject はボタンアクション
type ActionObject struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Text  string `json:"text,omitempty"`
	URI   string `json:"uri,omitempty"`
}

// ========== リッチメニュー構造体 ==========

// RichMenu はLINEリッチメニュー
type RichMenu struct {
	Size        RichMenuSize   `json:"size"`
	Selected    bool           `json:"selected"`
	Name        string         `json:"name"`
	ChatBarText string         `json:"chatBarText"`
	Areas       []RichMenuArea `json:"areas"`
}

// RichMenuSize はリッチメニューのサイズ
type RichMenuSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RichMenuArea はリッチメニューのタップ領域
type RichMenuArea struct {
	Bounds RichMenuBounds `json:"bounds"`
	Action RichMenuAction `json:"action"`
}

// RichMenuBounds はタップ領域の座標
type RichMenuBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RichMenuAction はリッチメニューのアクション
type RichMenuAction struct {
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
	URI   string `json:"uri,omitempty"`
}

// RichMenuResponse はリッチメニュー作成APIのレスポンス
type RichMenuResponse struct {
	RichMenuID string `json:"richMenuId"`
}

// RichMenuListResponse はリッチメニュー一覧のレスポンス
type RichMenuListResponse struct {
	RichMenus []RichMenuInfo `json:"richmenus"`
}

// RichMenuInfo はリッチメニュー情報
type RichMenuInfo struct {
	RichMenuID  string       `json:"richMenuId"`
	Name        string       `json:"name"`
	Size        RichMenuSize `json:"size"`
	ChatBarText string       `json:"chatBarText"`
	Selected    bool         `json:"selected"`
}
