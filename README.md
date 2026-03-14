# Official WhatsApp Business Api

A self-hosted WhatsApp automation dashboard built on Meta Cloud API (v25.0). Supports real-time chat, mass broadcasting, auto-replies, and follow-up sequences — all from a clean, responsive UI.

---

## 🚀 Quick Start

### Local (Go)
```bash
# 1. Setup environment
cp .env.example .env
# Fill in your WHATSAPP_TOKEN, PHONE_NUMBER_ID, WABA_ID

# 2. Run
go run auto.go
# Open http://localhost:8086
```

### Docker
```bash
docker-compose up -d --build
# Open http://localhost:8086
```

### 🐳 Docker Hub Image
Get the pre-built image from Docker Hub:  
`docker pull akashyadav758/whatsapp-hub`  
Explore on [Docker Hub](https://hub.docker.com/r/akashyadav758/whatsapp-hub)

---


## ⚙️ Environment Variables (`.env`)

| Variable | Description |
|---|---|
| `WHATSAPP_TOKEN` | Meta Cloud API Bearer token |
| `PHONE_NUMBER_ID` | WhatsApp Phone Number ID |
| `WABA_ID` | WhatsApp Business Account ID |
| `WEBHOOK_PORT` | Local server port (default: `8086`) |
| `WEBHOOK_URL` | Public URL for Meta webhook (e.g. Cloudflare Tunnel) |
| `WEBHOOK_VERIFY_TOKEN` | Webhook verification token |
| `APP_NAME` | Dashboard display name |
| `API_VERSION` | Meta API version (default: `v25.0`) |
| `AUTO_REPLY_DELAY_SEC` | Delay before auto-reply fires (default: `2`) |
| `BROADCAST_DELAY_MS` | Delay between broadcast messages (default: `200`) |
| `TZ` | Timezone (default: `Asia/Kolkata`) |

---

## ✨ Features

- **📨 Real-time Chat** — Live incoming/outgoing messages via SSE + polling. Read state persisted in DB (works across browsers/devices).
- **📢 Mass Broadcast** — Template-based broadcast with rate limiting and variable substitution.
- **🤖 Auto-Reply** — Smart auto-reply with configurable delay and 24h loop prevention.
- **⏱ Follow-Up Sequences** — Up to 3 timed follow-up messages per contact.
- **💡 Dark / Light Mode** — Theme toggle, persisted per browser.
- **📱 Mobile Responsive** — WhatsApp-style layout with bottom tab bar on mobile.
- **🔔 Unread Badges** — Per-contact unread counts, server-synced so badges stay correct across sessions.

---

## 📡 API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/messages` | Fetch all messages (filter by `?phone=`) |
| GET | `/api/statuses` | Fetch message delivery statuses |
| GET/POST | `/api/readstate` | Get/set per-contact read timestamps (DB-backed) |
| GET | `/api/templates` | Fetch Meta approved templates |
| POST | `/api/broadcast` | Send broadcast to multiple numbers |
| POST | `/api/reply` | Send a single reply to a contact |
| GET/POST | `/api/config` | Get/update runtime config |
| GET/POST | `/api/settings` | Get/update persistent settings (auto-reply, follow-ups) |
| GET | `/api/events` | SSE stream for real-time updates |
| GET | `/health` | Health check |
| GET/POST | `/webhook` | Meta Cloud API webhook endpoint |

---

## 📁 Project Structure

```
.
├── auto.go              # Core backend (Go) — HTTP server, webhook handler, DB, SSE
├── go.mod / go.sum      # Go module files
├── .env                 # Environment config (not committed)
├── docker-compose.yml   # Docker deployment
├── docker/
│   └── Dockerfile       # Container build config
├── static/
│   ├── index.html       # App shell
│   ├── app.js           # React UI (single file, no build step)
│   └── app.css          # Styles (CSS variables, dark/light theme)
└── data/
    └── messages.db      # SQLite database (auto-created on first run)
```

---

## 🗄 Database

SQLite at `data/messages.db`. Key tables:

- **`messages`** — All incoming/outgoing messages with direction, phone, type, timestamps
- **`statuses`** — Message delivery status updates (sent/delivered/read)
- **`settings`** — Persistent key-value config including read state (`readstate_PHONE`)

---

## 🔗 Webhook Setup

1. Set `WEBHOOK_URL` to your public URL (e.g. via Cloudflare Tunnel).
2. In Meta Developer Console → WhatsApp → Configuration → set Webhook URL to `{WEBHOOK_URL}/webhook`.
3. Set Verify Token to match `WEBHOOK_VERIFY_TOKEN` in `.env`.
4. Subscribe to `messages` field.
