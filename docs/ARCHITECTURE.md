# CirclePay システム構成図

## システム全体図

```mermaid
flowchart TB
    %% ========== ユーザー ==========
    subgraph Users["ユーザー"]
        direction LR
        Participant["参加者<br/>LINE Bot"]
        Organizer["会計者<br/>LIFF App"]
    end

    %% ========== LINE Platform ==========
    subgraph LINE["LINE Platform"]
        direction LR
        subgraph MsgAPI["Messaging API"]
            Webhook["Webhook"]
            Reply["Reply"]
            Push["Push"]
        end
        OAuth["OAuth2"]
        LIFF["LIFF SDK"]
    end

    %% ========== ローカル環境 ==========
    subgraph Local["ローカル開発環境"]
        direction TB

        ngrok["ngrok Tunnel<br/>:443 → :8080"]

        subgraph FE["Frontend :5173"]
            direction LR
            Pages["Pages<br/>Events / Create / Approve"]
            Hooks["Hooks<br/>useLiff / api"]
            Proxy["Vite Proxy"]
        end

        subgraph BE["Backend :8080"]
            direction TB
            subgraph Handlers["Handlers"]
                direction LR
                Bot["bot.go<br/>Webhook処理"]
                API["liff.go<br/>REST API"]
            end
            subgraph Core["Core"]
                direction LR
                Auth["http.go<br/>認証"]
                Msg["messaging.go<br/>送信"]
                DB["database.go<br/>CRUD"]
            end
        end
    end

    %% ========== データベース ==========
    subgraph Database["PostgreSQL"]
        direction LR
        users["users"]
        events["events"]
        participants["participants"]
    end

    %% ========== 接続（上から下へ流れる） ==========

    %% ユーザー → LINE
    Participant --> Webhook
    Organizer --> LIFF

    %% LINE → ローカル（Bot経路）
    Webhook --> ngrok
    ngrok --> Bot
    Bot --> Msg
    Msg --> Reply
    Msg --> Push

    %% LINE → ローカル（LIFF経路）
    LIFF --> Pages
    Pages --> Hooks
    Hooks --> Proxy
    Proxy --> API
    API --> Auth
    Auth --> OAuth

    %% Backend → Database
    API --> DB
    Bot --> DB
    DB --> Database

    %% ========== スタイル ==========
    classDef user fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef line fill:#00c300,stroke:#00a000,color:#fff,stroke-width:2px
    classDef tunnel fill:#ffebee,stroke:#c62828,stroke-width:2px
    classDef fe fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef be fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef db fill:#e8f5e9,stroke:#388e3c,stroke-width:2px

    class Participant,Organizer user
    class Webhook,Reply,Push,OAuth,LIFF line
    class ngrok tunnel
    class Pages,Hooks,Proxy fe
    class Bot,API,Auth,Msg,DB be
    class users,events,participants db
```

---

## データフロー

```mermaid
flowchart LR
    subgraph BotFlow["Bot経路（参加者）"]
        direction LR
        U1["参加者"] --> L1["LINE"]
        L1 --> N1["ngrok"]
        N1 --> B1["Bot Handler"]
        B1 --> D1["Database"]
        B1 --> M1["Messaging"]
        M1 --> L1
    end

    subgraph LIFFFlow["LIFF経路（会計者）"]
        direction LR
        U2["会計者"] --> L2["LIFF"]
        L2 --> F2["Frontend"]
        F2 --> A2["LIFF API"]
        A2 --> O2["OAuth検証"]
        A2 --> D2["Database"]
        A2 --> M2["Push通知"]
        M2 --> L2
    end

    classDef user fill:#e3f2fd,stroke:#1976d2
    classDef ext fill:#00c300,stroke:#00a000,color:#fff
    classDef app fill:#f3e5f5,stroke:#7b1fa2
    classDef db fill:#e8f5e9,stroke:#388e3c

    class U1,U2 user
    class L1,L2,N1,O2 ext
    class B1,F2,A2,M1,M2 app
    class D1,D2 db
```

---

## データベース構成

```mermaid
erDiagram
    users ||--o{ events : "creates"
    users ||--o{ event_participants : "joins"
    events ||--|{ event_participants : "has"

    users {
        text user_id PK
        text name
        text circle
        int step
    }

    events {
        int id PK
        text event_name
        text organizer_id FK
        int total_amount
        int split_amount
        text status
    }

    event_participants {
        int id PK
        int event_id FK
        text user_id
        bool paid
        timestamp approved_at
    }
```

---

## 開発環境ネットワーク

```mermaid
flowchart LR
    subgraph Internet["インターネット"]
        LINE["LINE API"]
    end

    subgraph Cloud["ngrok Cloud"]
        ngrok["ngrok<br/>xxx.ngrok-free.app"]
    end

    subgraph Local["ローカルマシン"]
        BE["Backend<br/>:8080"]
        FE["Frontend<br/>:5173"]
        DB["PostgreSQL<br/>:5432"]
    end

    LINE <--> ngrok
    ngrok <--> BE
    FE -->|"Proxy"| BE
    BE --> DB

    classDef ext fill:#e3f2fd,stroke:#1976d2
    classDef cloud fill:#ffebee,stroke:#c62828
    classDef local fill:#e8f5e9,stroke:#388e3c

    class LINE ext
    class ngrok cloud
    class BE,FE,DB local
```

---

## コンポーネント詳細

| レイヤー | ファイル | 役割 |
|---------|---------|------|
| Frontend | `pages/*.tsx` | イベント一覧・作成・承認画面 |
| Frontend | `useLiff.ts` | LIFF認証・状態管理 |
| Frontend | `api.ts` | Backend APIクライアント |
| Backend | `main.go` | ルーティング・静的配信 |
| Backend | `bot.go` | LINE Webhook処理 |
| Backend | `liff.go` | LIFF用REST API |
| Backend | `http.go` | 認証ミドルウェア |
| Backend | `messaging.go` | LINE送信（Reply/Push） |
| Backend | `database.go` | PostgreSQL操作 |
| Database | `users` | ユーザー情報 |
| Database | `events` | 割り勘イベント |
| Database | `event_participants` | 参加者・支払い状態 |
