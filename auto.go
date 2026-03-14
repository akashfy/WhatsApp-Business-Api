package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	cfg        *WebhookCfg
	cfgMu      sync.RWMutex
	db         *sql.DB
	logger     = slog.New(slog.NewTextHandler(os.Stdout, nil))
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	tplCache   []TemplateItem
	tplCacheMu sync.RWMutex
	tplCacheTs time.Time

	// SSE: real-time event broadcasting
	sseClients   = make(map[chan string]struct{})
	sseClientsMu sync.Mutex
)

type WebhookCfg struct {
	Token        string
	PhoneID      string
	APIVer       string
	VerifyToken  string
	Port         string
	WABAID       string
	WebhookURL   string
	TunnelURL    string
	AppName      string
	BcastDelay   int
	AppID        string
	Debug        string
	TplLang      string
	TplName      string
	Var1         string
	Var2         string
	Var3         string
	Var4         string
	ForwardPorts map[string]string // phoneNumberId -> "localhost:port" for multi-number routing
}

type TemplateComponent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type TemplateItem struct {
	Name       string              `json:"name"`
	Category   string              `json:"category"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components"`
}

func loadCfg() *WebhookCfg {
	_ = godotenv.Load()
	vToken := os.Getenv("WEBHOOK_VERIFY_TOKEN")
	if vToken == "" {
		vToken = "whatsapp_secure_vtoken_2026"
	}
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8080"
	}

	apiV := os.Getenv("API_VERSION")
	if apiV == "" {
		apiV = "v25.0"
	}

	bDelay, _ := strconv.Atoi(os.Getenv("BROADCAST_DELAY_MS"))
	if bDelay <= 0 {
		bDelay = 200
	}

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "WhatsApp Business"
	}

	// Parse forward ports: "phoneId1:port1,phoneId2:port2"
	forwardPorts := make(map[string]string)
	if fp := os.Getenv("WEBHOOK_FORWARD_PORTS"); fp != "" {
		for _, entry := range strings.Split(fp, ",") {
			parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
			if len(parts) == 2 {
				forwardPorts[parts[0]] = parts[1]
				fmt.Printf("📡 Webhook forward: phone %s → localhost:%s\n", parts[0], parts[1])
			}
		}
	}

	return &WebhookCfg{
		Token:        os.Getenv("WHATSAPP_TOKEN"),
		PhoneID:      os.Getenv("PHONE_NUMBER_ID"),
		APIVer:       apiV,
		VerifyToken:  vToken,
		Port:         port,
		WABAID:       os.Getenv("WABA_ID"),
		WebhookURL:   os.Getenv("WEBHOOK_URL"),
		TunnelURL:    getTunnelURL(),
		AppName:      appName,
		BcastDelay:   bDelay,
		AppID:        os.Getenv("APP_ID"),
		Debug:        os.Getenv("DEBUG"),
		TplLang:      os.Getenv("TEMPLATE_LANG"),
		TplName:      os.Getenv("TEMPLATE_NAME"),
		Var1:         os.Getenv("VAR1"),
		Var2:         os.Getenv("VAR2"),
		Var3:         os.Getenv("VAR3"),
		Var4:         os.Getenv("VAR4"),
		ForwardPorts: forwardPorts,
	}
}

func getTunnelURL() string {
	// First check env variable for explicit URL
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL != "" {
		return webhookURL
	}

	logPath := os.Getenv("TUNNEL_LOG_PATH")
	if logPath == "" {
		logPath = "/app/data/tunnel.log"
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		port := os.Getenv("WEBHOOK_PORT")
		if port == "" {
			port = "8080"
		}
		return "http://localhost:" + port
	}
	lines := strings.Split(string(content), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "trycloudflare.com") {
			parts := strings.Split(lines[i], "|")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Tunnel starting..."
}

func initDB() {
	var err error
	os.MkdirAll("data", 0755)
	db, err = sql.Open("sqlite3", "file:data/messages.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		fmt.Printf("[FATAL] Cannot open database: %v\n", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		direction TEXT NOT NULL DEFAULT 'incoming',
		phone TEXT NOT NULL,
		push_name TEXT DEFAULT '',
		message TEXT DEFAULT '',
		message_type TEXT DEFAULT 'text',
		message_id TEXT DEFAULT '',
		context_message_id TEXT DEFAULT '',
		location_lat REAL DEFAULT 0,
		location_lng REAL DEFAULT 0,
		location_name TEXT DEFAULT '',
		meta_timestamp DATETIME,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(phone, message_id)
	)`)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create messages table: %v\n", err)
		return
	}

	// statuses table: delivery/read receipts
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS statuses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT NOT NULL,
		phone TEXT NOT NULL,
		status TEXT NOT NULL,
		meta_timestamp DATETIME,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, status)
	)`)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create statuses table: %v\n", err)
	}

	// auto_replies table: tracking to prevent loops
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS auto_replies (
		phone TEXT PRIMARY KEY,
		last_sent DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create auto_replies table: %v\n", err)
	}

	// scheduled_followups table: auto send follow-up after X hours
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduled_followups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		phone TEXT NOT NULL,
		message TEXT NOT NULL,
		scheduled_at DATETIME NOT NULL,
		sent INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create scheduled_followups table: %v\n", err)
	}

	// settings table: store auto-reply message and other UI configs
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	)`)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create settings table: %v\n", err)
	}

	// Default settings
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('auto_reply_message', 'Thank you for contacting us! We will get back to you shortly.')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('auto_reply_enabled', 'false')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('auto_reply_interval_hours', '12')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('auto_reply_delay_seconds', '300')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup_enabled', 'false')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup_message', '🔔 Reminder: Check our latest offers!')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup_delay_minutes', '480')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup2_enabled', 'false')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup2_message', '⏰ Last chance! This offer expires soon.')")
	db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('followup2_delay_minutes', '720')")

	// Migrate: add columns if old DB
	db.Exec(`ALTER TABLE messages ADD COLUMN push_name TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE messages ADD COLUMN message_type TEXT DEFAULT 'text'`)
	db.Exec(`ALTER TABLE messages ADD COLUMN meta_timestamp DATETIME`)
	db.Exec(`ALTER TABLE messages ADD COLUMN context_message_id TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE messages ADD COLUMN location_lat REAL DEFAULT 0`)
	db.Exec(`ALTER TABLE messages ADD COLUMN location_lng REAL DEFAULT 0`)
	db.Exec(`ALTER TABLE messages ADD COLUMN location_name TEXT DEFAULT ''`)

	db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_phone ON messages(phone)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_statuses_message ON statuses(message_id)`)

	fmt.Println("✅ data/messages.db initialized with WAL mode")
}

func saveMessageDB(direction, phone, pushName, message, msgType, messageID, contextMsgID, locationName string, locationLat, locationLng float64, metaTS time.Time, ts time.Time) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		`INSERT OR IGNORE INTO messages (direction, phone, push_name, message, message_type, message_id, context_message_id, location_lat, location_lng, location_name, meta_timestamp, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		direction, phone, pushName, message, msgType, messageID, contextMsgID, locationLat, locationLng, locationName, metaTS, ts,
	)
	if err != nil {
		logger.Error("DB Save failed", "error", err)
	}
}

func saveStatusDB(messageID, phone, status string, metaTS time.Time) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		`INSERT OR IGNORE INTO statuses (message_id, phone, status, meta_timestamp) VALUES (?, ?, ?, ?)`,
		messageID, phone, status, metaTS,
	)
	if err != nil {
		logger.Error("Status DB Save failed", "error", err)
	}
}

// ==================== SSE (Server-Sent Events) ====================

func broadcastSSE(eventType, data string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	sseClientsMu.Lock()
	defer sseClientsMu.Unlock()
	for ch := range sseClients {
		select {
		case ch <- msg:
		default:
			// Client too slow, skip
		}
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 50)
	sseClientsMu.Lock()
	sseClients[ch] = struct{}{}
	sseClientsMu.Unlock()

	defer func() {
		sseClientsMu.Lock()
		delete(sseClients, ch)
		sseClientsMu.Unlock()
		close(ch)
	}()

	// Send initial ping
	fmt.Fprintf(w, "event: ping\ndata: connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, msg)
			flusher.Flush()
		}
	}
}

// ==================== API HANDLERS ====================

func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db == nil {
		fmt.Println("[ERROR] handleGetMessages: DB not initialized")
		http.Error(w, "DB not initialized", 500)
		return
	}
	phone := r.URL.Query().Get("phone")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 10000 {
		limit = 5000
	}

	var rows *sql.Rows
	var err error
	if phone != "" {
		rows, err = db.Query(`SELECT id, direction, phone, push_name, message, message_type, message_id, context_message_id, location_lat, location_lng, location_name, meta_timestamp, timestamp FROM messages WHERE phone = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`, phone, limit, offset)
	} else {
		rows, err = db.Query(`SELECT id, direction, phone, push_name, message, message_type, message_id, context_message_id, location_lat, location_lng, location_name, meta_timestamp, timestamp FROM messages ORDER BY timestamp DESC LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		fmt.Printf("[ERROR] GET /api/messages: %v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type MsgRow struct {
		ID            int     `json:"id"`
		Direction     string  `json:"direction"`
		Phone         string  `json:"phone"`
		PushName      string  `json:"push_name"`
		Message       string  `json:"message"`
		MessageType   string  `json:"message_type"`
		MessageID     string  `json:"message_id"`
		ContextMsgID  string  `json:"context_message_id"`
		LocationLat   float64 `json:"location_lat"`
		LocationLng   float64 `json:"location_lng"`
		LocationName  string  `json:"location_name"`
		MetaTimestamp string  `json:"meta_timestamp"`
		Timestamp     string  `json:"timestamp"`
	}
	msgs := []MsgRow{}
	for rows.Next() {
		var m MsgRow
		if err := rows.Scan(&m.ID, &m.Direction, &m.Phone, &m.PushName, &m.Message, &m.MessageType, &m.MessageID, &m.ContextMsgID, &m.LocationLat, &m.LocationLng, &m.LocationName, &m.MetaTimestamp, &m.Timestamp); err != nil {
			logger.Error("Row scan failed", "error", err)
			continue
		}
		msgs = append(msgs, m)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func handleGetStatuses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db == nil {
		fmt.Println("[ERROR] handleGetStatuses: DB not initialized")
		http.Error(w, "DB not initialized", 500)
		return
	}
	msgID := r.URL.Query().Get("message_id")
	var rows *sql.Rows
	var err error
	if msgID != "" {
		rows, err = db.Query(`SELECT id, message_id, phone, status, meta_timestamp, timestamp FROM statuses WHERE message_id = ? ORDER BY timestamp DESC`, msgID)
	} else {
		rows, err = db.Query(`SELECT id, message_id, phone, status, meta_timestamp, timestamp FROM statuses ORDER BY timestamp DESC LIMIT 200`)
	}
	if err != nil {
		fmt.Printf("[ERROR] GET /api/statuses: %v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type StRow struct {
		ID            int    `json:"id"`
		MessageID     string `json:"message_id"`
		Phone         string `json:"phone"`
		Status        string `json:"status"`
		MetaTimestamp string `json:"meta_timestamp"`
		Timestamp     string `json:"timestamp"`
	}
	statuses := []StRow{}
	for rows.Next() {
		var s StRow
		if err := rows.Scan(&s.ID, &s.MessageID, &s.Phone, &s.Status, &s.MetaTimestamp, &s.Timestamp); err != nil {
			logger.Error("Status Row scan failed", "error", err)
			continue
		}
		statuses = append(statuses, s)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func handleGetContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db == nil {
		http.Error(w, "DB not initialized", 500)
		return
	}
	rows, err := db.Query(`SELECT m.phone, m.push_name, m.message, m.message_type, m.direction, m.timestamp
		FROM messages m
		INNER JOIN (SELECT phone, MAX(id) as max_id FROM messages GROUP BY phone) latest
		ON m.phone = latest.phone AND m.id = latest.max_id
		ORDER BY m.timestamp DESC`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type ContactRow struct {
		Phone       string `json:"phone"`
		PushName    string `json:"push_name"`
		LastMessage string `json:"last_message"`
		LastType    string `json:"last_type"`
		Direction   string `json:"direction"`
		Timestamp   string `json:"timestamp"`
	}
	contacts := []ContactRow{}
	for rows.Next() {
		var c ContactRow
		if err := rows.Scan(&c.Phone, &c.PushName, &c.LastMessage, &c.LastType, &c.Direction, &c.Timestamp); err != nil {
			continue
		}
		contacts = append(contacts, c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

// processFollowups runs in background, checks every 60s for due follow-ups
func processFollowups() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if db == nil {
			continue
		}
		rows, err := db.Query(`SELECT id, phone, message FROM scheduled_followups WHERE sent = 0 AND scheduled_at <= ? ORDER BY scheduled_at ASC LIMIT 50`, time.Now())
		if err != nil {
			continue
		}
		type followup struct {
			id      int
			phone   string
			message string
		}
		var due []followup
		for rows.Next() {
			var f followup
			if e := rows.Scan(&f.id, &f.phone, &f.message); e == nil {
				due = append(due, f)
			}
		}
		rows.Close()

		for _, f := range due {
			// Mark as sent FIRST to prevent duplicate sends
			db.Exec("UPDATE scheduled_followups SET sent = 1 WHERE id = ?", f.id)

			logger.Info("📅 Sending scheduled follow-up", "to", f.phone)
			cfgMu.RLock()
			c := cfg
			cfgMu.RUnlock()
			sendText(c, f.phone, f.message)

			// Save to messages DB
			saveMessageDB("outgoing", f.phone, "", f.message, "text", "", "", "", 0, 0, time.Now(), time.Now())
			broadcastSSE("new_message", "{}")

			time.Sleep(500 * time.Millisecond) // small delay between sends
		}
	}
}

func main() {
	cfg = loadCfg()
	initDB()
	if db != nil {
		defer db.Close()
	}

	// Start background follow-up processor
	go processFollowups()

	// Static files (CSS, JS, images, etc.)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 1. Dashboard UI
	http.HandleFunc("/", serveDashboard)

	// 2. APIs
	http.HandleFunc("/api/templates", handleGetTemplates)
	http.HandleFunc("/api/broadcast", limitBody(handleBroadcast, 5<<20))
	http.HandleFunc("/api/reply", limitBody(handleReply, 1<<20))
	http.HandleFunc("/api/messages", handleGetMessages)
	http.HandleFunc("/api/contacts", handleGetContacts)
	http.HandleFunc("/api/statuses", handleGetStatuses)
	http.HandleFunc("/api/config", limitBody(handleConfig, 1<<20))
	http.HandleFunc("/api/settings", limitBody(handleSettings, 1<<20))
	http.HandleFunc("/api/broadcast/status", handleBroadcastStatus)
	http.HandleFunc("/api/readstate", limitBody(handleReadState, 1<<20))

	// 3. SSE endpoint for real-time updates
	http.HandleFunc("/api/events", handleSSE)

	// 4. Webhook (Public with its own Meta auth)
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/health", handleHealth)

	logger.Info("🚀 Server started", "port", cfg.Port, "webhookURL", cfg.WebhookURL)
	fmt.Printf("╔══════════════════════════════════════════╗\n")
	fmt.Printf("║  WhatsApp Business Hub                   ║\n")
	fmt.Printf("║  Local:   http://localhost:%s          ║\n", cfg.Port)
	fmt.Printf("║  Webhook: %s/webhook  ║\n", cfg.WebhookURL)
	fmt.Printf("╚══════════════════════════════════════════╝\n")

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db == nil {
		http.Error(w, "DB down", 503)
		return
	}
	if err := db.Ping(); err != nil {
		http.Error(w, "DB down", 503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func limitBody(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		mode := q.Get("hub.mode")
		token := q.Get("hub.verify_token")
		challenge := q.Get("hub.challenge")

		logger.Info("Webhook verification attempt", "mode", mode, "token_match", token == cfg.VerifyToken)

		if mode == "subscribe" && token == cfg.VerifyToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			logger.Info("✅ Webhook verified successfully")
			return
		}
		// Also support old-style without mode check
		if token == cfg.VerifyToken && challenge != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			logger.Info("✅ Webhook verified (legacy)")
			return
		}
		logger.Warn("❌ Webhook verification failed", "expected", cfg.VerifyToken, "got", token)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Body too large", http.StatusRequestEntityTooLarge)
			return
		}

		// IMPORTANT: Always respond 200 immediately to Meta
		w.WriteHeader(http.StatusOK)

		// Debug logging
		cfgMu.RLock()
		debug := cfg.Debug
		myPhoneID := cfg.PhoneID
		forwardPorts := cfg.ForwardPorts
		cfgMu.RUnlock()

		if debug == "true" {
			logger.Info("📩 Webhook POST received", "bodyLen", len(body), "body", string(body))
		} else {
			logger.Info("📩 Webhook POST received", "bodyLen", len(body))
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			logger.Error("Failed to parse webhook payload", "error", err)
			return
		}

		// Multi-number routing: check phone_number_id and forward if needed
		if len(forwardPorts) > 0 {
			incomingPhoneID := extractPhoneNumberID(payload)
			if incomingPhoneID != "" && incomingPhoneID != myPhoneID {
				if targetPort, ok := forwardPorts[incomingPhoneID]; ok {
					logger.Info("📡 Forwarding webhook to other container", "phoneID", incomingPhoneID, "port", targetPort)
					go forwardWebhook(body, targetPort)
					return
				}
				logger.Warn("⚠️ Unknown phone_number_id, no forward rule", "phoneID", incomingPhoneID)
			}
		}

		// Process in background so webhook responds instantly
		go handleIncoming(payload)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tplCacheMu.RLock()
	if time.Since(tplCacheTs) < 5*time.Minute && tplCache != nil {
		templates := tplCache
		tplCacheMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
		return
	}
	tplCacheMu.RUnlock()

	templates := fetchTemplates(cfg)

	tplCacheMu.Lock()
	tplCache = templates
	tplCacheTs = time.Now()
	tplCacheMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "DB not initialized", 500)
		return
	}
	if r.Method == http.MethodGet {
		rows, err := db.Query("SELECT key, value FROM settings")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		settings := make(map[string]string)
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				logger.Error("Settings Row scan failed", "error", err)
				continue
			}
			settings[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
		return
	}
	if r.Method == http.MethodPost {
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var allowedSettings = map[string]bool{
			"auto_reply_message":        true,
			"auto_reply_enabled":        true,
			"auto_reply_interval_hours": true,
			"auto_reply_delay_seconds":  true,
			"followup_enabled":          true,
			"followup_message":          true,
			"followup_delay_minutes":    true,
			"followup2_enabled":         true,
			"followup2_message":         true,
			"followup2_delay_minutes":   true,
		}

		for k, v := range req {
			if !allowedSettings[k] {
				logger.Warn("Rejected unknown setting key", "key", k)
				continue
			}
			_, err := db.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?", k, v, v)
			if err != nil {
				logger.Error("Settings update failed", "key", k, "error", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleReadState: GET returns map of phone->lastSeenISO, POST accepts {phone:isoTime} to persist
func handleReadState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodGet {
		if db == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}
		rows, err := db.Query("SELECT key, value FROM settings WHERE key LIKE 'readstate_%'")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}
		defer rows.Close()
		result := make(map[string]string)
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				continue
			}
			// Strip "readstate_" prefix to get phone number
			phone := strings.TrimPrefix(k, "readstate_")
			result[phone] = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method == http.MethodPost {
		if db == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for phone, ts := range req {
			key := "readstate_" + phone
			db.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?", key, ts, ts)
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		cfgMu.RLock()
		resp := map[string]string{
			"appName":        cfg.AppName,
			"apiVersion":     cfg.APIVer,
			"webhookUrl":     cfg.WebhookURL,
			"tunnelUrl":      getTunnelURL(),
			"verifyToken":    cfg.VerifyToken,
			"whatsappToken":  cfg.Token,
			"phoneNumberId":  cfg.PhoneID,
			"wabaId":         cfg.WABAID,
			"appId":          cfg.AppID,
			"debug":          cfg.Debug,
			"templateLang":   cfg.TplLang,
			"templateName":   cfg.TplName,
			"var1":           cfg.Var1,
			"var2":           cfg.Var2,
			"var3":           cfg.Var3,
			"var4":           cfg.Var4,
			"broadcastDelay": strconv.Itoa(cfg.BcastDelay),
		}
		cfgMu.RUnlock()
		json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		envMap, _ := godotenv.Read(".env")
		if envMap == nil {
			envMap = make(map[string]string)
		}

		cfgMu.Lock()
		if val, ok := req["appName"]; ok {
			envMap["APP_NAME"] = val
			cfg.AppName = val
		}
		if val, ok := req["apiVersion"]; ok {
			envMap["API_VERSION"] = val
			cfg.APIVer = val
		}
		if val, ok := req["webhookUrl"]; ok {
			envMap["WEBHOOK_URL"] = val
			cfg.WebhookURL = val
		}
		if val, ok := req["verifyToken"]; ok {
			envMap["WEBHOOK_VERIFY_TOKEN"] = val
			cfg.VerifyToken = val
		}
		if val, ok := req["whatsappToken"]; ok {
			envMap["WHATSAPP_TOKEN"] = val
			cfg.Token = val
		}
		if val, ok := req["phoneNumberId"]; ok {
			envMap["PHONE_NUMBER_ID"] = val
			cfg.PhoneID = val
		}
		if val, ok := req["wabaId"]; ok {
			envMap["WABA_ID"] = val
			cfg.WABAID = val
		}
		if val, ok := req["appId"]; ok {
			envMap["APP_ID"] = val
			cfg.AppID = val
		}
		if val, ok := req["debug"]; ok {
			envMap["DEBUG"] = val
			cfg.Debug = val
		}
		if val, ok := req["templateLang"]; ok {
			envMap["TEMPLATE_LANG"] = val
			cfg.TplLang = val
		}
		if val, ok := req["templateName"]; ok {
			envMap["TEMPLATE_NAME"] = val
			cfg.TplName = val
		}
		if val, ok := req["var1"]; ok {
			envMap["VAR1"] = val
			cfg.Var1 = val
		}
		if val, ok := req["var2"]; ok {
			envMap["VAR2"] = val
			cfg.Var2 = val
		}
		if val, ok := req["var3"]; ok {
			envMap["VAR3"] = val
			cfg.Var3 = val
		}
		if val, ok := req["var4"]; ok {
			envMap["VAR4"] = val
			cfg.Var4 = val
		}
		if val, ok := req["broadcastDelay"]; ok {
			envMap["BROADCAST_DELAY_MS"] = val
			if v, err := strconv.Atoi(val); err == nil && v > 0 {
				cfg.BcastDelay = v
			}
		}
		cfgMu.Unlock()

		godotenv.Write(envMap, ".env")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type BroadcastReq struct {
	Numbers  []string `json:"numbers"`
	Template string   `json:"template"`
	Language string   `json:"language"`
	Params   []string `json:"params"`
}

type BroadcastState struct {
	IsRunning bool   `json:"is_running"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Success   int    `json:"success"`
	Failed    int    `json:"failed"`
	StartTime string `json:"start_time"`
}

var bState BroadcastState
var bMu sync.Mutex

func handleBroadcastStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bMu.Lock()
	defer bMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bState)
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BroadcastReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	total := 0
	for _, n := range req.Numbers {
		if strings.TrimSpace(n) != "" {
			total++
		}
	}

	if total == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	bMu.Lock()
	if bState.IsRunning {
		bMu.Unlock()
		http.Error(w, "A broadcast is already running", http.StatusConflict)
		return
	}

	bState = BroadcastState{
		IsRunning: true,
		Total:     total,
		Processed: 0,
		Success:   0,
		Failed:    0,
		StartTime: time.Now().Format(time.RFC3339),
	}
	bMu.Unlock()

	go func() {
		defer func() {
			bMu.Lock()
			bState.IsRunning = false
			bMu.Unlock()
			// Notify UI
			broadcastSSE("broadcast_complete", `{"status":"done"}`)
		}()

		for _, num := range req.Numbers {
			num = strings.TrimSpace(num)
			if num == "" {
				continue
			}

			if !isValidPhone(num) {
				logger.Warn("Skipping invalid phone", "phone", num)
				bMu.Lock()
				bState.Processed++
				bState.Failed++
				bMu.Unlock()
				continue
			}

			success, _ := sendTemplateSync(cfg, num, req.Template, req.Language, req.Params)

			bMu.Lock()
			bState.Processed++
			if success {
				bState.Success++
			} else {
				bState.Failed++
			}
			bMu.Unlock()

			cfgMu.RLock()
			delay := cfg.BcastDelay
			cfgMu.RUnlock()
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

type ReplyReq struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func isValidPhone(phone string) bool {
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	for _, c := range phone {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func handleReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ReplyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.To == "" || req.Message == "" {
		http.Error(w, "Missing 'to' or 'message'", http.StatusBadRequest)
		return
	}
	if !isValidPhone(req.To) {
		http.Error(w, "Invalid phone number", http.StatusBadRequest)
		return
	}
	go sendText(cfg, req.To, req.Message)
	w.WriteHeader(http.StatusOK)
}

func sendTemplateSync(cfg *WebhookCfg, to, tpl, lang string, params []string) (bool, string) {
	cfgMu.RLock()
	apiVer := cfg.APIVer
	phoneID := cfg.PhoneID
	cfgMu.RUnlock()

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", apiVer, phoneID)

	compParams := []map[string]interface{}{}
	for _, p := range params {
		p = strings.ReplaceAll(p, "\r\n", " ")
		p = strings.ReplaceAll(p, "\n", " ")
		p = strings.ReplaceAll(p, "\t", " ")

		for strings.Contains(p, "     ") {
			p = strings.ReplaceAll(p, "     ", "    ")
		}

		compParams = append(compParams, map[string]interface{}{
			"type": "text",
			"text": p,
		})
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name":     tpl,
			"language": map[string]string{"code": lang},
		},
	}

	if len(compParams) > 0 {
		payload["template"].(map[string]interface{})["components"] = []map[string]interface{}{
			{
				"type":       "body",
				"parameters": compParams,
			},
		}
	}

	return doSendSync(cfg, url, payload, to, "Template")
}

// extractPhoneNumberID pulls metadata.phone_number_id from Meta webhook payload
func extractPhoneNumberID(payload map[string]interface{}) string {
	entries, _ := payload["entry"].([]interface{})
	if len(entries) == 0 {
		return ""
	}
	changes, _ := entries[0].(map[string]interface{})["changes"].([]interface{})
	if len(changes) == 0 {
		return ""
	}
	value, _ := changes[0].(map[string]interface{})["value"].(map[string]interface{})
	if value == nil {
		return ""
	}
	metadata, _ := value["metadata"].(map[string]interface{})
	if metadata == nil {
		return ""
	}
	phoneID, _ := metadata["phone_number_id"].(string)
	return phoneID
}

// forwardWebhook sends raw webhook body to another container's /webhook endpoint
func forwardWebhook(body []byte, port string) {
	url := fmt.Sprintf("http://localhost:%s/webhook", port)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Error("❌ Failed to forward webhook", "port", port, "error", err)
		return
	}
	defer resp.Body.Close()
	logger.Info("✅ Webhook forwarded successfully", "port", port, "status", resp.StatusCode)
}

func handleIncoming(payload map[string]interface{}) {
	if db == nil {
		return
	}
	entries, _ := payload["entry"].([]interface{})
	if len(entries) == 0 {
		logger.Warn("Webhook payload has no entries")
		return
	}
	changes, _ := entries[0].(map[string]interface{})["changes"].([]interface{})
	if len(changes) == 0 {
		logger.Warn("Webhook payload has no changes")
		return
	}
	value, _ := changes[0].(map[string]interface{})["value"].(map[string]interface{})

	// --- Handle delivery/read STATUSES ---
	if statuses, ok := value["statuses"].([]interface{}); ok {
		for _, rawSt := range statuses {
			st, ok := rawSt.(map[string]interface{})
			if !ok {
				continue
			}
			msgID, _ := st["id"].(string)
			recipient, _ := st["recipient_id"].(string)
			status, _ := st["status"].(string)
			metaTS := time.Now()
			if tsStr, ok := st["timestamp"].(string); ok {
				tsVal, _ := strconv.ParseInt(tsStr, 10, 64)
				if tsVal > 0 {
					metaTS = time.Unix(tsVal, 0)
				}
			}
			logger.Info("📋 Status update", "msgID", msgID, "to", recipient, "status", status)
			saveStatusDB(msgID, recipient, status, metaTS)

			// Notify UI via SSE
			sseData, _ := json.Marshal(map[string]string{
				"message_id": msgID,
				"phone":      recipient,
				"status":     status,
			})
			broadcastSSE("status_update", string(sseData))
		}
		return
	}

	msgs, _ := value["messages"].([]interface{})
	if len(msgs) == 0 {
		logger.Info("Webhook payload has no messages (may be a status-only update)")
		return
	}

	// Extract push_name from contacts
	pushName := ""
	if contacts, ok := value["contacts"].([]interface{}); ok && len(contacts) > 0 {
		if profile, ok := contacts[0].(map[string]interface{})["profile"].(map[string]interface{}); ok {
			pushName, _ = profile["name"].(string)
		}
	}

	for _, rawMsg := range msgs {
		m, ok := rawMsg.(map[string]interface{})
		if !ok {
			continue
		}
		from, _ := m["from"].(string)
		msgType, _ := m["type"].(string)
		msgID, _ := m["id"].(string)

		// Reply context
		contextMsgID := ""
		if ctx, ok := m["context"].(map[string]interface{}); ok {
			contextMsgID, _ = ctx["id"].(string)
		}

		// Meta actual timestamp
		metaTS := time.Now()
		if tsStr, ok := m["timestamp"].(string); ok {
			tsVal, _ := strconv.ParseInt(tsStr, 10, 64)
			if tsVal > 0 {
				metaTS = time.Unix(tsVal, 0)
			}
		}

		text := ""
		var locationLat, locationLng float64
		locationName := ""

		switch msgType {
		case "text":
			if t, ok := m["text"].(map[string]interface{}); ok {
				text, _ = t["body"].(string)
			}
		case "image", "video", "audio", "document", "sticker":
			if media, ok := m[msgType].(map[string]interface{}); ok {
				text, _ = media["caption"].(string)
				if text == "" {
					text = "[" + strings.ToUpper(msgType) + "]"
				}
			}
		case "location":
			if loc, ok := m["location"].(map[string]interface{}); ok {
				locationLat, _ = loc["latitude"].(float64)
				locationLng, _ = loc["longitude"].(float64)
				locationName, _ = loc["name"].(string)
				text = fmt.Sprintf("[LOCATION] %.6f, %.6f %s", locationLat, locationLng, locationName)
			}
		case "reaction":
			if r, ok := m["reaction"].(map[string]interface{}); ok {
				emoji, _ := r["emoji"].(string)
				reactMsgID, _ := r["message_id"].(string)
				text = fmt.Sprintf("[REACTION] %s on %s", emoji, reactMsgID)
			}
		case "button":
			if btn, ok := m["button"].(map[string]interface{}); ok {
				text, _ = btn["text"].(string)
			}
		case "interactive":
			if inter, ok := m["interactive"].(map[string]interface{}); ok {
				if reply, ok := inter["button_reply"].(map[string]interface{}); ok {
					text, _ = reply["title"].(string)
				} else if listReply, ok := inter["list_reply"].(map[string]interface{}); ok {
					text, _ = listReply["title"].(string)
				}
			}
		case "order":
			text = "[ORDER]"
		}

		if from == "" {
			continue
		}

		logger.Info("💬 Message received", "type", msgType, "from", from, "name", pushName, "text", text)

		saveMessageDB("incoming", from, pushName, text, msgType, msgID, contextMsgID, locationName, locationLat, locationLng, metaTS, time.Now())

		// Notify UI via SSE (real-time)
		sseData, _ := json.Marshal(map[string]interface{}{
			"direction":    "incoming",
			"phone":        from,
			"push_name":    pushName,
			"message":      text,
			"message_type": msgType,
			"message_id":   msgID,
			"timestamp":    time.Now().Format(time.RFC3339),
		})
		broadcastSSE("new_message", string(sseData))

		// --- SMART AUTO-REPLY LOGIC ---
		doReply := false
		var lastSent time.Time

		// Read configurable interval (default 12 hours)
		intervalHours := 12
		var intervalStr string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = 'auto_reply_interval_hours'").Scan(&intervalStr); err == nil {
			if v, e := strconv.Atoi(intervalStr); e == nil && v > 0 {
				intervalHours = v
			}
		}
		intervalDuration := time.Duration(intervalHours) * time.Hour

		err := db.QueryRow("SELECT last_sent FROM auto_replies WHERE phone = ?", from).Scan(&lastSent)
		if err == sql.ErrNoRows {
			doReply = true
		} else if err == nil {
			if time.Since(lastSent) > intervalDuration {
				doReply = true
			}
		}

		if doReply && msgType != "reaction" && msgType != "order" {
			var autoReplyMsg string
			var enabled string
			db.QueryRow("SELECT value FROM settings WHERE key = 'auto_reply_message'").Scan(&autoReplyMsg)
			db.QueryRow("SELECT value FROM settings WHERE key = 'auto_reply_enabled'").Scan(&enabled)

			if enabled == "true" && autoReplyMsg != "" {
				logger.Info("🤖 Auto-reply #1 triggered", "to", from, "interval_hours", intervalHours)
				db.Exec("INSERT INTO auto_replies (phone, last_sent) VALUES (?, ?) ON CONFLICT(phone) DO UPDATE SET last_sent = ?", from, time.Now(), time.Now())

				// Read delay from settings (in SECONDS, default 300 = 5min)
				delaySec := 300
				var delayStr string
				if err := db.QueryRow("SELECT value FROM settings WHERE key = 'auto_reply_delay_seconds'").Scan(&delayStr); err == nil {
					if v, e := strconv.Atoi(delayStr); e == nil && v >= 0 {
						delaySec = v
					}
				}
				logger.Info("⏳ Auto-reply #1 scheduled", "to", from, "delay_sec", delaySec)
				go func(to, m string, ds int) {
					time.Sleep(time.Duration(ds) * time.Second)
					sendText(cfg, to, m)
					saveMessageDB("outgoing", to, "", m, "text", "", "", "", 0, 0, time.Now(), time.Now())
					broadcastSSE("new_message", "{}")
				}(from, autoReplyMsg, delaySec)

				// Clear old pending followups for this phone
				db.Exec("DELETE FROM scheduled_followups WHERE phone = ? AND sent = 0", from)

				// Schedule Follow-Up #2
				var fu1Enabled, fu1Msg, fu1DelayStr string
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup_enabled'").Scan(&fu1Enabled)
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup_message'").Scan(&fu1Msg)
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup_delay_minutes'").Scan(&fu1DelayStr)
				if fu1Enabled == "true" && fu1Msg != "" {
					fu1Delay := 480 // default 8 hours
					if v, e := strconv.Atoi(fu1DelayStr); e == nil && v > 0 {
						fu1Delay = v
					}
					scheduled1 := time.Now().Add(time.Duration(fu1Delay) * time.Minute)
					db.Exec("INSERT INTO scheduled_followups (phone, message, scheduled_at) VALUES (?, ?, ?)", from, fu1Msg, scheduled1)
					logger.Info("📅 Follow-up #2 scheduled", "to", from, "at", scheduled1.Format("15:04"), "delay_min", fu1Delay)
				}

				// Schedule Follow-Up #3
				var fu2Enabled, fu2Msg, fu2DelayStr string
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup2_enabled'").Scan(&fu2Enabled)
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup2_message'").Scan(&fu2Msg)
				db.QueryRow("SELECT value FROM settings WHERE key = 'followup2_delay_minutes'").Scan(&fu2DelayStr)
				if fu2Enabled == "true" && fu2Msg != "" {
					fu2Delay := 720 // default 12 hours
					if v, e := strconv.Atoi(fu2DelayStr); e == nil && v > 0 {
						fu2Delay = v
					}
					scheduled2 := time.Now().Add(time.Duration(fu2Delay) * time.Minute)
					db.Exec("INSERT INTO scheduled_followups (phone, message, scheduled_at) VALUES (?, ?, ?)", from, fu2Msg, scheduled2)
					logger.Info("📅 Follow-up #3 scheduled", "to", from, "at", scheduled2.Format("15:04"), "delay_min", fu2Delay)
				}
			}
		}
	}
}

func fetchTemplates(cfg *WebhookCfg) []TemplateItem {
	cfgMu.RLock()
	apiVer := cfg.APIVer
	wabaID := cfg.WABAID
	token := cfg.Token
	cfgMu.RUnlock()

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?status=APPROVED&limit=1000", apiVer, wabaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create template request", "error", err)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("Meta API Templates Request Failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("Meta API Templates Error", "code", resp.StatusCode, "body", string(body))
		return nil
	}

	var res struct {
		Data []TemplateItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		logger.Error("Failed to decode template JSON", "error", err)
		return nil
	}

	logger.Info("Templates fetched", "count", len(res.Data))
	return res.Data
}

func sendText(cfg *WebhookCfg, to, msg string) {
	cfgMu.RLock()
	apiVer := cfg.APIVer
	phoneID := cfg.PhoneID
	cfgMu.RUnlock()

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", apiVer, phoneID)
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": msg},
	}
	success, msgID := doSendSync(cfg, url, payload, to, "Reply")
	if msgID == "" {
		if success {
			msgID = fmt.Sprintf("out-%d", time.Now().UnixNano())
		} else {
			msgID = fmt.Sprintf("out-fail-%d", time.Now().UnixNano())
		}
	}
	saveMessageDB("outgoing", to, "", msg, "text", msgID, "", "", 0, 0, time.Now(), time.Now())
	if !success {
		saveStatusDB(msgID, to, "failed", time.Now())
	} else {
		saveStatusDB(msgID, to, "sent", time.Now())
	}

	// Notify UI via SSE
	sseData, _ := json.Marshal(map[string]interface{}{
		"direction":    "outgoing",
		"phone":        to,
		"message":      msg,
		"message_type": "text",
		"message_id":   msgID,
		"timestamp":    time.Now().Format(time.RFC3339),
	})
	broadcastSSE("new_message", string(sseData))
}

func doSendSync(cfg *WebhookCfg, url string, payload map[string]interface{}, to, mode string) (bool, string) {
	b, err := json.Marshal(payload)
	if err != nil {
		logger.Error("JSON Marshal failed", "mode", mode, "to", to, "error", err)
		return false, ""
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		logger.Error("Request creation failed", "mode", mode, "to", to, "error", err)
		return false, ""
	}
	cfgMu.RLock()
	token := cfg.Token
	cfgMu.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("API call failed", "mode", mode, "to", to, "error", err)
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("Meta API call failed", "mode", mode, "to", to, "code", resp.StatusCode, "body", string(body))
		return false, ""
	}

	logger.Info("✅ Message sent", "mode", mode, "to", to)
	var r struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err == nil && len(r.Messages) > 0 {
		return true, r.Messages[0].ID
	}
	return true, ""
}
