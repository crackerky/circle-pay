# CirclePay システム構成図

## システム全体図

```mermaid
flowchart TB
    subgraph Users["ユーザー"]
        Participant["参加者"]
        Organizer["会計者"]
    end

    subgraph LINE["LINE Platform"]
        MessagingAPI["Messaging API"]
        OAuth["OAuth2"]
        LIFF["LIFF SDK"]
    end

    subgraph LocalEnv["ローカル開発環境"]
        subgraph Frontend["Frontend :5173"]
            ReactApp["React SPA"]
            ViteProxy["Vite Proxy"]
        end

        subgraph Backend["Backend :8080"]
            BotHandler["Bot Handler"]
            LiffAPI["LIFF API"]
            Messaging["Messaging"]
            DBLayer["DB Layer"]
        end

        ngrok["ngrok Tunnel"]
    end

    subgraph Database["PostgreSQL"]
        users_t["users"]
        events_t["events"]
        participants_t["participants"]
    end

    Participant -->|"メッセージ"| MessagingAPI
    Organizer -->|"LIFF起動"| LIFF
    MessagingAPI -->|"Webhook"| ngrok
    ngrok --> BotHandler
    LIFF --> ReactApp
    ReactApp --> ViteProxy
    ViteProxy -->|"/api/*"| LiffAPI
    LiffAPI -->|"トークン検証"| OAuth
    BotHandler --> Messaging
    LiffAPI --> DBLayer
    Messaging -->|"Reply/Push"| MessagingAPI
    DBLayer --> Database

    classDef user fill:#e3f2fd,stroke:#1976d2
    classDef line fill:#00c300,stroke:#00a000,color:#fff
    classDef frontend fill:#fff3e0,stroke:#f57c00
    classDef backend fill:#f3e5f5,stroke:#7b1fa2
    classDef db fill:#e8f5e9,stroke:#388e3c

    class Participant,Organizer user
    class MessagingAPI,OAuth,LIFF line
    class ReactApp,ViteProxy frontend
    class BotHandler,LiffAPI,Messaging,DBLayer,ngrok backend
    class users_t,events_t,participants_t db
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

## 開発環境

```mermaid
flowchart LR
    subgraph Internet
        LINE["LINE API"]
    end

    subgraph ngrok
        tunnel["ngrok :443"]
    end

    subgraph Local["ローカル"]
        BE["Backend :8080"]
        FE["Frontend :5173"]
        DB["PostgreSQL :5432"]
    end

    LINE <--> tunnel
    tunnel <--> BE
    FE -->|"proxy"| BE
    BE --> DB

    classDef ext fill:#e3f2fd,stroke:#1976d2
    classDef local fill:#e8f5e9,stroke:#388e3c

    class LINE,tunnel ext
    class BE,FE,DB local
```

---

## コンポーネント一覧

| レイヤー | コンポーネント | 役割 |
|---------|--------------|------|
| Frontend | React SPA | LIFF Webアプリ |
| Frontend | Vite Proxy | API転送 |
| Backend | Bot Handler | LINE Bot処理 |
| Backend | LIFF API | Web API |
| Backend | Messaging | メッセージ送信 |
| Backend | DB Layer | データ操作 |
| Database | PostgreSQL | データ永続化 |
