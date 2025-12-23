# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CirclePay is a hybrid LINE Bot + LIFF application for expense splitting in student circles/clubs. The system enables:
- **Organizers**: Use LIFF web app to create events, select participants, and approve payments
- **Participants**: Use LINE Bot with Quick Reply buttons to report payments
- **Architecture**: Go backend + React/TypeScript LIFF frontend + PostgreSQL database

## Development Commands

### Backend (Go)
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
PostgreSQL tables are created automatically on backend startup via `createTables()` in `main.go`.

## Architecture

### Hybrid Bot + LIFF Design

**LINE Bot (Quick Reply buttons)**:
- Used by participants to report payments
- Interactive buttons replace text commands
- Implemented in `backend/bot.go`

**LIFF App (Web interface)**:
- Used by organizers to create events and approve payments
- Full React SPA with routing
- Runs inside LINE app, authenticated with LIFF tokens
- Implemented in `frontend/src/`

**Key principle**: Organizers use feature-rich web UI, participants use simple bot interactions.

### Backend Structure (Go)

Backend is split into three files:

**`backend/main.go`** - Common infrastructure
- Data models: `User`, `Event`, `Participant`
- Database layer: `initDB()`, `createTables()`, CRUD functions
- Routing setup
- Static file serving (serves `../frontend/dist`)

**`backend/bot.go`** - LINE Bot functionality
- Webhook handler: `handleWebhook()`
- Quick Reply implementation: `QuickReplyButton`, `showMainMenu()`
- Message handlers: `handleMessage()`, `handleRegisteredUserMessage()`
- LINE API functions: `ReplyMessage()`, `PushMessage()`, `ReplyMessageWithQuickReply()`
- User registration flow (Steps 1-3: Name → Circle → Complete)
- Payment reporting flow for participants

**`backend/liff.go`** - LIFF API endpoints
- LIFF token verification: `verifyLIFFToken()`, `authenticateRequest()`
- Event creation: `handleEvents()` POST endpoint with participant selection
- Approval flow: `handleApprovals()` GET/POST endpoints
- Circle member lookup: `handleGetCircleMembers()`
- User info: `handleGetMyInfo()`
- Registration: `handleRegisterUser()`

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
- `/admin` - View users and send messages (testing only)

### Key Design Patterns

**State Machine Pattern**: User interactions tracked by database state fields:
- `User.Step`: Registration (0=unregistered, 1=name, 2=circle, 3=complete)
- `User.SplitEventStep`: Event creation flow (unused in current LIFF design)
- `User.ApprovalStep`: Approval flow (unused in current LIFF design)

**Database-First State**: All state persisted to PostgreSQL immediately, no in-memory state.

**Async Notifications**: Push messages sent in goroutines to avoid blocking HTTP responses.

**SPA Serving**: Backend serves `index.html` for all non-API routes to support React Router.

**LIFF Authentication**:
1. Frontend calls `liff.init()` with LIFF ID
2. If not logged in, redirects to LINE login
3. After login, gets access token via `liff.getAccessToken()`
4. All API calls include `Authorization: Bearer <token>` header
5. Backend verifies token with LINE API before processing

## Environment Variables

**Backend** (`.env` in project root):
```bash
LINE_CHANNEL_ACCESS_TOKEN=your_token
LINE_CHANNEL_SECRET=your_secret
DATABASE_URL=postgres://user:pass@host:port/db?sslmode=disable
PORT=8080
LIFF_ID=2008577348-GDBXaBEr
LIFF_URL=https://liff.line.me/2008577348-GDBXaBEr
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
- **👤 会計者になる**: Opens LIFF app (organizer features)

Legacy text commands (still work but not shown):
- `割り勘`: Create event (deprecated, use LIFF)
- `状況確認`: Check payment status (deprecated)
- `承認`: Approve payments (deprecated, use LIFF)

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

## Security Considerations

- **Input Sanitization**: `sanitizeInput()` escapes HTML and trims whitespace
- **LIFF Token Verification**: Backend verifies all tokens with LINE OAuth API
- **Authorization**: Approval endpoints check organizer identity
- **Secrets**: `.env` files excluded from git
