# Circle Pay - LINE Bot

LINE Messaging APIを使ったシンプルなWebhookサーバー。Go標準ライブラリのみで実装されています。

## 機能

- LINE Webhook受信
- メッセージの自動返信
- 会話履歴の自動保存（JSON形式）
- ユーザーごとの会話履歴管理
- 標準ライブラリのみで実装（外部依存なし）

## セットアップ

### 1. LINE Developersでチャネルを作成

1. [LINE Developers Console](https://developers.line.biz/console/)にアクセス
2. 新しいプロバイダーとMessaging APIチャネルを作成
3. 以下の情報を取得：
   - Channel Access Token (長期)
   - Channel Secret

### 2. 環境変数の設定

`.env`ファイルを編集して、取得した情報を設定：

```bash
LINE_CHANNEL_ACCESS_TOKEN=your_channel_access_token_here
LINE_CHANNEL_SECRET=your_channel_secret_here
PORT=8080
```

### 3. ビルドと実行

```bash
# ビルド
go build -o circle-pay main.go

# 実行
./circle-pay
```

または直接実行：

```bash
go run main.go
```

### 4. Webhook URLの設定

1. サーバーを公開（ngrokなどを使用）
2. LINE Developers Consoleで Webhook URL を設定：
   ```
   https://your-domain.com/webhook
   ```
3. Webhookを有効化

## 使い方

LINEでボットにメッセージを送ると、「受信しました: [あなたのメッセージ]」と返信されます。

## 会話履歴

### 自動保存
すべての会話は `conversations.json` ファイルに自動保存されます。このファイルには以下の情報が含まれます：

- ユーザーID
- タイムスタンプ
- メッセージの方向（received/sent）
- メッセージ本文

### ファイル形式
```json
{
  "ユーザーID": {
    "userId": "ユーザーID",
    "messages": [
      {
        "timestamp": "2025-11-02T12:00:00Z",
        "direction": "received",
        "text": "こんにちは"
      },
      {
        "timestamp": "2025-11-02T12:00:01Z",
        "direction": "sent",
        "text": "受信しました: こんにちは"
      }
    ]
  }
}
```

### プライバシーについて
`conversations.json` には個人情報が含まれるため、`.gitignore` で除外されています。本番環境では適切なアクセス制御を行ってください。

## 構造体

### WebhookRequest
LINEから送信されるWebhookデータの構造体

### Event
各イベントの情報を保持

### ReplyRequest
LINE APIへの返信リクエスト用の構造体

### ConversationMessage
個々のメッセージ情報（タイムスタンプ、方向、テキスト）

### UserConversation
ユーザーごとの会話履歴

### ConversationHistory
全ユーザーの会話履歴を管理（スレッドセーフ）

## 技術スタック

- Go標準ライブラリ
  - `net/http`: HTTPサーバー
  - `encoding/json`: JSON処理
  - `bufio`: .envファイル読み込み
  - `sync`: スレッドセーフな会話履歴管理
  - `time`: タイムスタンプ管理

## ライセンス

MIT
