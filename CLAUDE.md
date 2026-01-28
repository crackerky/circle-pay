# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CirclePay is a hybrid LINE Bot + LIFF application for expense splitting in student circles/clubs. The system enables:
- **Organizers**: Use LIFF web app to create events, select participants, and approve payments
- **Participants**: Use LINE Bot with Quick Reply buttons to report payments
- **Architecture**: Go backend (Gin framework) + React/TypeScript LIFF frontend + PostgreSQL database

## Development Commands

### Backend (Go + Gin)
```bash
cd backend

# Run backend (reads .env from parent directory)
go run .

# Build
go build -o circle-pay

# Run built binary
./circle-pay
```

### Frontend (React + Vite + LIFF SDK)
```bash
cd frontend

# Install dependencies
npm install

# Development server (port 5173, proxies to backend on 8080)
npm run dev

# Build for production (outputs to dist/)
npm run build

# Lint
npm run lint

# Preview production build
npm run preview
```

### Database
PostgreSQL tables are created automatically on backend startup via `createTables()` in `db.go`.

## Architecture

### Hybrid Bot + LIFF Design

**LINE Bot (Quick Reply buttons)**:
- Used by participants to report payments
- Interactive buttons replace text commands
- Implemented in `backend/handler_bot.go`

**LIFF App (Web interface)**:
- Used by organizers to create events and approve payments
- Full React SPA with routing
- Runs inside LINE app, authenticated with LIFF tokens
- Implemented in `frontend/src/`

**Key principle**: Organizers use feature-rich web UI, participants use simple bot interactions.

### Backend Structure (Go + Gin) - Layered Architecture

```
backend/
├── main.go              # Entry point only
├── models.go            # All data structures (User, Event, Participant, etc.)
├── util.go              # Utility functions (sanitizeInput, formatAmount)
│
├── db.go                # Database connection and table creation
├── user_repo.go         # User CRUD operations
├── event_repo.go        # Event CRUD operations
├── participant_repo.go  # Participant CRUD operations
├── message_repo.go      # Message CRUD operations
│
├── handler_bot.go       # LINE Bot webhook handler
├── handler_liff.go      # LIFF API handlers
├── handler_admin.go     # Admin API handlers
│
├── line_client.go       # LINE messaging API (Strategy Pattern)
├── line_richmenu.go     # Rich menu management
│
├── middleware.go        # Authentication middleware
├── router.go            # Gin router setup
├── static.go            # Static file serving
└── reminder.go          # Reminder scheduler
```

#### Layer Responsibilities

| Layer | Files | Responsibility |
|-------|-------|----------------|
| **Entry** | `main.go` | App initialization, server startup |
| **Models** | `models.go` | Data structure definitions |
| **Repository** | `*_repo.go` | Database operations (CRUD) |
| **Handler** | `handler_*.go` | HTTP request/response handling |
| **LINE Client** | `line_*.go` | LINE API communication |
| **Infrastructure** | `middleware.go`, `router.go`, `static.go` | HTTP infrastructure |

#### File Details

**`main.go`** - Entry point
- Environment variable loading
- Database initialization
- Server startup

**`models.go`** - All data structures
- Domain models: `User`, `Event`, `Participant`, `UnpaidParticipant`
- LINE structures: `WebhookRequest`, `QuickReplyButton`, `RichMenu`, etc.

**`*_repo.go`** - Repository layer (database operations)
- `user_repo.go`: `GetUser()`, `SaveUser()`, `UpdateUser()`, `GetAllUsers()`
- `event_repo.go`: `GetEvent()`, `CreateEvent()`, `GetEventsByOrganizer()`
- `participant_repo.go`: `CreateParticipant()`, `GetUnpaidParticipants()`, `ApproveParticipant()`
- `message_repo.go`: `SaveMessage()`

**`handler_bot.go`** - LINE Bot handlers
- `handleWebhook()`: Webhook receiver
- `handleMessage()`: Message routing
- User registration flow (Steps 1-3)
- Payment reporting flow

**`handler_liff.go`** - LIFF API handlers
- `handleRegisterUser()`, `handleGetMyInfo()`
- `handleGetEvents()`, `handleCreateEvent()`
- `handleGetApprovals()`, `handleApprovePayments()`
- `handleGetCircleMembers()`

**`handler_admin.go`** - Admin API handlers
- `handleGetUsers()`, `handleAllMessages()`
- `handleSend()`, `handleTestReminder()`
- Rich menu management handlers

**`line_client.go`** - LINE Messaging API (Strategy Pattern)
- Delivery strategies: `ReplyDelivery`, `PushDelivery`, `MulticastDelivery`
- Content types: `TextContent`, `QuickReplyContent`
- Helper functions: `ReplyMessage()`, `PushMessage()`, etc.

**`middleware.go`** - Authentication middleware
- `LIFFAuthMiddleware()`: LIFF token verification
- `AdminAuthMiddleware()`: API key validation
- `LineSignatureMiddleware()`: Webhook signature verification

### API Endpoints

**LINE Bot** (signature validation):
```
POST /webhook              - LINE webhook receiver
```

**LIFF APIs** (LIFF token required):
```
POST /api/liff/register         - Register/update user
POST /api/liff/message          - Save message
GET  /api/liff/me               - Get user profile
GET  /api/liff/events           - List events
POST /api/liff/events           - Create event
GET  /api/liff/approvals        - List pending approvals
POST /api/liff/approvals        - Approve payments
GET  /api/liff/circle/members   - Get circle members
```

**Admin APIs** (API key required via `X-API-Key` header):
```
GET  /api/admin/users                  - List users
GET  /api/admin/messages               - List messages
POST /api/admin/send                   - Send message to user
POST /api/admin/test/send-reminders    - Trigger reminder manually
GET  /api/admin/richmenu               - List rich menus
POST /api/admin/richmenu               - Create rich menu
POST /api/admin/richmenu/:id/image     - Upload rich menu image
POST /api/admin/richmenu/:id/default   - Set default rich menu
DELETE /api/admin/richmenu/:id         - Delete rich menu
```

### Frontend Structure (React + TypeScript)

```
frontend/src/
├── main.tsx              # Entry point, renders LiffApp or App based on route
├── LiffApp.tsx           # LIFF app with React Router
├── App.tsx               # Admin panel (separate from LIFF)
├── liff/
│   ├── useLiff.ts        # LIFF initialization hook with auth state
│   └── api.ts            # Backend API client functions
├── pages/
│   ├── EventsPage.tsx    # Event list and navigation hub
│   ├── CreateEvent.tsx   # Event creation with member selection
│   └── ApprovePage.tsx   # Payment approval interface
└── types/
    └── index.ts          # TypeScript type definitions
```

**LIFF Routes**:
- `/events` - List of events (default)
- `/create` - Create new event form
- `/approve` - Approve pending payments

**Admin Panel**:
- `/admin` - View users and send messages (requires API key)

### Key Design Patterns

**Gin Middleware Pattern**: Authentication handled by middleware before handlers execute.

**State Machine Pattern**: User interactions tracked by database state fields:
- `User.Step`: Registration (0=unregistered, 1=name, 2=circle, 3=complete)

**Database-First State**: All state persisted to PostgreSQL immediately, no in-memory state.

**Async Notifications**: Push messages sent in goroutines to avoid blocking HTTP responses.

**SPA Serving**: Backend serves `index.html` for all non-API routes to support React Router.

**LIFF Authentication**:
1. Frontend calls `liff.init()` with LIFF ID
2. If not logged in, redirects to LINE login
3. After login, gets access token via `liff.getAccessToken()`
4. All API calls include `Authorization: Bearer <token>` header
5. Backend `LIFFAuthMiddleware()` verifies token with LINE API

**Admin Authentication**:
1. Client includes `X-API-Key: <key>` header (or `?api_key=<key>` query param)
2. Backend `AdminAuthMiddleware()` compares with `ADMIN_API_KEY` env var
3. Returns 401 if missing or invalid

## Environment Variables

**Backend** (`.env` in project root):
```bash
LINE_CHANNEL_ACCESS_TOKEN=your_token
LINE_CHANNEL_SECRET=your_secret
DATABASE_URL=postgres://user:pass@host:port/db?sslmode=disable
PORT=8080
LIFF_ID=2008577348-GDBXaBEr
LIFF_URL=https://liff.line.me/2008577348-GDBXaBEr

# Admin API authentication (required for /api/admin/* endpoints)
ADMIN_API_KEY=your_secure_random_key
```

**Frontend** (`frontend/.env`):
```bash
VITE_LIFF_ID=2008577348-GDBXaBEr
VITE_API_BASE=    # Empty for same-origin requests
```

## Database Schema

Tables (auto-created):
- `users`: User registration and state tracking
  - Primary key: `user_id` (LINE user ID)
  - State fields: `step`, `split_event_step`, `approval_step`
  - Profile: `name`, `circle`
- `events`: Split payment events
  - Primary key: `id` (serial)
  - Fields: `event_name`, `organizer_id`, `circle`, `total_amount`, `split_amount`, `status`
  - Status values: 'selecting', 'confirmed', 'completed'
- `event_participants`: Many-to-many event participation
  - Foreign key: `event_id` → `events(id)`
  - Fields: `user_id`, `user_name`, `paid`, `reported_at`, `approved_at`
- `messages`: Message history (admin only)

Indexes:
- `events`: `organizer_id`, `circle`
- `event_participants`: `event_id`, `user_id`

## LINE Bot Commands

Quick Reply buttons shown to registered users:
- **💰 支払いました**: Report payment completion
- **📊 状況確認**: Check payment status
- **👤 会計者になる**: Opens LIFF app (organizer features)

## Development Workflow

1. **Start PostgreSQL**: Ensure running and `DATABASE_URL` configured
2. **Start Backend**: `cd backend && go run .` (port 8080)
3. **Start Frontend Dev**: `cd frontend && npm run dev` (port 5173)
4. **Access Locally**: http://localhost:5173
5. **Test with LINE**: Use ngrok to expose backend, configure webhook

## Testing with LINE (ngrok)

```bash
# Terminal 1: Start ngrok
ngrok http 8080

# Terminal 2: Start backend
cd backend && go run .

# Configure LINE Developers Console:
# 1. Webhook URL: https://xxx.ngrok-free.app/webhook
# 2. LIFF Endpoint URL: https://xxx.ngrok-free.app
# 3. Enable webhook
```

**Important**: Backend must be run from `backend/` directory so static files are served from `../frontend/dist`.

## Testing Admin API

```bash
# With API key header (recommended)
curl -H "X-API-Key: your_key" http://localhost:8080/api/admin/users

# With query parameter (for browser testing)
curl "http://localhost:8080/api/admin/users?api_key=your_key"

# Send message
curl -H "X-API-Key: your_key" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8080/api/admin/send \
     -d '{"userID":"U...","message":"Hello"}'
```

## Common Issues

**Loading screen stuck on LIFF app**:
- Check browser console for `[LIFF]` debug logs
- Verify LIFF Endpoint URL matches ngrok URL
- Ensure ngrok URL is HTTPS
- Check LIFF ID matches in both frontend `.env` and backend `.env`

**Static files return 404**:
- Backend must run from `backend/` directory
- Frontend must be built: `cd frontend && npm run build`
- Check `../frontend/dist/index.html` exists from backend directory

**"address already in use" error**:
- Kill existing process: `lsof -ti:8080 | xargs kill -9`

**Infinite login loop on LIFF**:
- Clear sessionStorage: `sessionStorage.clear()`
- Check redirectUri is correct (should be current URL)
- Verify LIFF token verification is working (check backend logs)

**Admin API returns 401**:
- Check `ADMIN_API_KEY` is set in `.env`
- Ensure header is `X-API-Key` (case-sensitive)
- Verify key matches exactly

**Admin API returns 503**:
- `ADMIN_API_KEY` environment variable is not set
- Add it to `.env` and restart backend

## Security Considerations

- **Input Sanitization**: `sanitizeInput()` escapes HTML and trims whitespace
- **LIFF Token Verification**: Backend verifies all tokens with LINE OAuth API
- **Admin API Protection**: API key required for all admin endpoints
- **LINE Webhook Signature**: HMAC-SHA256 validation for webhook requests
- **Authorization**: Approval endpoints check organizer identity
- **Secrets**: `.env` files excluded from git
