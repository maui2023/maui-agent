package web

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
)

var (
	hubMu        sync.RWMutex
	activeServer *Server
)

// RegisterServer registers the running WebUI server as the global execution receiver.
func RegisterServer(s *Server) {
	hubMu.Lock()
	activeServer = s
	hubMu.Unlock()
}

// UnregisterServer removes the active server registration.
func UnregisterServer(s *Server) {
	hubMu.Lock()
	if activeServer == s {
		activeServer = nil
	}
	hubMu.Unlock()
}

// BroadcastExecutionStart notifies web viewers that an agent has begun executing a command.
func BroadcastExecutionStart(model, prompt, source string) {
	hubMu.RLock()
	s := activeServer
	hubMu.RUnlock()

	if s != nil {
		s.handleTaskStart(model, prompt, source)
	} else {
		go postInternalStream(map[string]interface{}{
			"action": "start",
			"model":  model,
			"prompt": prompt,
			"source": source,
		})
	}
}

// BroadcastExecutionLog streams a single output line to WebUI terminal in real time.
func BroadcastExecutionLog(logType, text string) {
	hubMu.RLock()
	s := activeServer
	hubMu.RUnlock()

	if s != nil {
		s.appendExecutionLog(logType, text)
	} else {
		go postInternalStream(map[string]interface{}{
			"action": "log",
			"type":   logType,
			"text":   text,
		})
	}
}

// BroadcastExecutionFinish notifies web viewers that execution ended.
func BroadcastExecutionFinish(model string, duration time.Duration, success bool) {
	hubMu.RLock()
	s := activeServer
	hubMu.RUnlock()

	if s != nil {
		s.handleTaskFinish(model, duration, success)
	} else {
		go postInternalStream(map[string]interface{}{
			"action":   "finish",
			"model":    model,
			"duration": duration.String(),
			"success":  success,
		})
	}
}

func postInternalStream(payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	port := "8080"
	if envPort := os.Getenv("MIQA_WEB_PORT"); envPort != "" {
		port = envPort
	}
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%s/api/internal/stream", port), "application/json", bytes.NewReader(data))
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}

//go:embed static/*
var embeddedStatic embed.FS

// Server represents the miqa WebUI HTTP and SSE server.
type Server struct {
	port       int
	host       string
	cfg        *config.Config
	agyClient  *agy.Client
	httpServer *http.Server

	mu             sync.RWMutex
	clients        map[chan string]bool
	isExecuting    bool
	currentTask    string
	activeModel    string
	modelStates    map[string]ModelState
	terminalLogs       []LogEntry
	executionStart     time.Time
	lastTranscriptPath string
	lastTranscriptPos  int64
}

// ModelState holds the operational and visual state of an AI model character.
type ModelState struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"` // "working", "idle", "sleeping", "exhausted"
	Credits     int     `json:"credits"`
	CurrentTask string  `json:"current_task"`
	TokensUsed  int     `json:"tokens_used"`
	Color       string  `json:"color"`
}

// LogEntry represents a terminal log line.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"` // "stdout", "stderr", "info", "system"
	Text      string `json:"text"`
}

// NewServer initializes the WebUI server.
func NewServer(host string, port int, cfg *config.Config, agyClient *agy.Client) *Server {
	if host == "" {
		host = "0.0.0.0"
	}
	if port <= 0 {
		port = 8080
	}

	s := &Server{
		host:         host,
		port:         port,
		cfg:          cfg,
		agyClient:    agyClient,
		clients:      make(map[chan string]bool),
		activeModel:  cfg.Model,
		modelStates:  make(map[string]ModelState),
		terminalLogs: make([]LogEntry, 0),
	}

	s.initModelStates()
	s.loadRecentTranscriptLogs()
	return s
}

func (s *Server) initModelStates() {
	defaultModels := []struct {
		ID    string
		Name  string
		Role  string
		Color string
	}{
		{"gemini-3.8-flash-high", "Gemini 3.8 Flash", "Lead Developer", "#00D7FF"},
		{"claude-sonnet-4-6", "Claude Sonnet 4.6", "Code Reviewer", "#FFB86C"},
		{"gpt-oss-120b-medium", "GPT-OSS 120B", "DevOps Engine", "#50FA7B"},
		{"ollama-local", "Local Ollama", "Offline Privacy", "#BD93F9"},
	}

	for _, m := range defaultModels {
		status := "sleeping"
		if m.ID == s.cfg.Model {
			status = "idle"
		}
		credits := 100

		s.modelStates[m.ID] = ModelState{
			ID:         m.ID,
			Name:       m.Name,
			Role:       m.Role,
			Status:     status,
			Credits:    credits,
			Color:      m.Color,
			TokensUsed: 0,
		}
	}
}

// Start launches the HTTP server and background metric publisher.
func (s *Server) Start(ctx context.Context) error {
	RegisterServer(s)
	defer UnregisterServer(s)

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/set-model", s.handleSetModel)
	mux.HandleFunc("/api/set-path", s.handleSetPath)
	mux.HandleFunc("/api/set-state", s.handleSetState)
	mux.HandleFunc("/api/run", s.handleRunPrompt)
	mux.HandleFunc("/api/events", s.handleSSE)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/internal/stream", s.handleInternalStream)

	// Static Assets
	// Dynamic lookup: check local directories first (for live edits), fallback to embeddedStatic
	subFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return fmt.Errorf("gagal memuat aset terbenam: %w", err)
	}
	embeddedHandler := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		// Try candidate dirs
		candidates := []string{
			"web/static",
			"../web/static",
			"pkg/web/static",
			"../pkg/web/static",
			"bin/web/static",
		}
		for _, dir := range candidates {
			fp := filepath.Join(dir, p)
			if fi, err := os.Stat(fp); err == nil && !fi.IsDir() {
				http.ServeFile(w, r, fp)
				return
			}
		}
		embeddedHandler.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}

	// Start background metric broadcaster
	go s.broadcastStatsLoop(ctx)

	s.mu.RLock()
	hasLogs := len(s.terminalLogs) > 0
	s.mu.RUnlock()
	if !hasLogs {
		s.appendLog("system", "system: info | [MIQA] Sedia menerima arahan dari Telegram (@miqa_agy_bot) atau CLI")
	}

	log.Printf("[miqa-web] Pelayan WebUI sedia pada http://localhost:%d (alamat: %s)", s.port, addr)

	// Run server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleStatus returns current system, agy, and session status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	activeModel := s.activeModel
	codingPath := s.cfg.CodingPath
	isExec := s.isExecuting
	currentTask := s.currentTask
	s.mu.RUnlock()

	stats, _ := agy.GetUsageStats(codingPath)

	resp := map[string]interface{}{
		"active_model":      activeModel,
		"coding_path":       codingPath,
		"is_executing":      isExec,
		"current_task":      currentTask,
		"telegram_status":   s.cfg.IsTelegramConfigured(),
		"skip_permissions":  true,
		"stats":             stats,
		"agy_authenticated": true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleModels returns the list of models and their states.
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []ModelState
	for _, m := range s.modelStates {
		list = append(list, m)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

// handleSetModel changes active model.
func (s *Server) handleSetModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model == "" {
		http.Error(w, "Model tidak sah", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.activeModel = body.Model
	s.cfg.Model = body.Model
	_ = s.cfg.SaveConfig()

	// Update model states
	for id, m := range s.modelStates {
		if id == body.Model {
			if s.isExecuting {
				m.Status = "working"
			} else {
				m.Status = "idle"
			}
		} else {
			if m.Status == "working" || m.Status == "idle" {
				m.Status = "sleeping"
			}
		}
		s.modelStates[id] = m
	}
	s.mu.Unlock()

	s.broadcastEvent("model_changed", map[string]string{"model": body.Model})
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "active_model": body.Model})
}

// handleSetPath changes workspace coding directory.
func (s *Server) handleSetPath(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "Laluan tidak sah", http.StatusBadRequest)
		return
	}

	cleanPath, err := config.ValidatePath(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.cfg.CodingPath = cleanPath
	_ = s.cfg.SaveConfig()
	s.mu.Unlock()

	s.broadcastEvent("path_changed", map[string]string{"path": cleanPath})
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": cleanPath})
}

// handleSetState allows toggling model states (e.g. simulate exhausted/hungry or working).
func (s *Server) handleSetState(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model   string `json:"model"`
		Status  string `json:"status"` // "working", "idle", "sleeping", "exhausted"
		Credits int    `json:"credits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Ralat data", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if m, ok := s.modelStates[body.Model]; ok {
		if body.Status != "" {
			m.Status = body.Status
		}
		if body.Credits >= 0 {
			m.Credits = body.Credits
			if m.Credits == 0 {
				m.Status = "exhausted"
			}
		}
		s.modelStates[body.Model] = m
	}
	s.mu.Unlock()

	s.broadcastEvent("models_updated", s.modelStates)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleLogs returns stored terminal logs.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.terminalLogs)
}

// handleRunPrompt is disabled on public WebUI for host security.
func (s *Server) handleRunPrompt(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Input WebUI dimatikan atas faktor keselamatan. Halaman ini adalah paparan umum. Sila hantar arahan melalui Telegram (@miqa_agy_bot) atau CLI.", http.StatusForbidden)
}

// handleTaskStart updates agent state to working and logs initiation.
func (s *Server) handleTaskStart(model, prompt, source string) {
	s.mu.Lock()
	s.isExecuting = true
	s.currentTask = prompt
	s.executionStart = time.Now()
	if model != "" {
		s.activeModel = model
	}

	for id, m := range s.modelStates {
		if id == s.activeModel {
			m.Status = "working"
			m.CurrentTask = prompt
		} else if m.Status == "working" {
			m.Status = "sleeping"
			m.CurrentTask = ""
		}
		s.modelStates[id] = m
	}
	s.mu.Unlock()

	srcLabel := "Telegram"
	if source != "" {
		srcLabel = source
	}
	s.appendLog("prompt", fmt.Sprintf("user: prompt | %s (%s)", prompt, srcLabel))
	s.appendLog("thinking", fmt.Sprintf("%s: thinking | Menganalisis tugasan: %s", s.activeModel, prompt))

	s.broadcastEvent("agent_started", map[string]interface{}{
		"model":  s.activeModel,
		"task":   prompt,
		"source": srcLabel,
	})
	s.broadcastEvent("models_updated", s.modelStates)
}

// handleTaskFinish restores agent state to idle and logs completion.
func (s *Server) handleTaskFinish(model string, duration time.Duration, success bool) {
	s.mu.Lock()
	s.isExecuting = false
	s.currentTask = ""
	for id, m := range s.modelStates {
		if id == s.activeModel {
			m.Status = "idle"
			m.CurrentTask = ""
		}
		s.modelStates[id] = m
	}
	s.mu.Unlock()

	statusIcon := "✅ Selesai"
	if !success {
		statusIcon = "❌ Ralat"
	}
	s.appendLog("result", fmt.Sprintf("%s: result | %s (Masa: %s)", s.activeModel, statusIcon, duration.Round(time.Millisecond)))

	s.broadcastEvent("agent_finished", map[string]interface{}{
		"model":    s.activeModel,
		"duration": duration.String(),
		"success":  success,
	})
	s.broadcastEvent("models_updated", s.modelStates)
}

// handleInternalStream processes stream events from other processes (CLI or bot).
func (s *Server) handleInternalStream(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action   string `json:"action"`
		Model    string `json:"model"`
		Prompt   string `json:"prompt"`
		Source   string `json:"source"`
		Type     string `json:"type"`
		Text     string `json:"text"`
		Duration string `json:"duration"`
		Success  bool   `json:"success"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	switch body.Action {
	case "start":
		s.handleTaskStart(body.Model, body.Prompt, body.Source)
	case "log":
		s.appendExecutionLog(body.Type, body.Text)
	case "finish":
		d, _ := time.ParseDuration(body.Duration)
		s.handleTaskFinish(body.Model, d, body.Success)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) appendLog(logType, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Type:      logType,
		Text:      trimmed,
	}

	s.mu.Lock()
	s.terminalLogs = append(s.terminalLogs, entry)
	if len(s.terminalLogs) > 300 {
		s.terminalLogs = s.terminalLogs[len(s.terminalLogs)-300:]
	}
	s.mu.Unlock()

	s.broadcastEvent("log", entry)
}

// appendExecutionLog categorizes live stream output lines into thinking, develop, result
func (s *Server) appendExecutionLog(logType, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}

	s.mu.RLock()
	model := s.activeModel
	s.mu.RUnlock()
	if model == "" {
		model = "gemini-3.8-flash-high"
	}

	lower := strings.ToLower(trimmed)
	var cat string
	var msg string

	if strings.Contains(lower, "thinking") || strings.Contains(lower, "analyz") || strings.Contains(lower, "merancang") || strings.Contains(lower, "menganalisis") {
		cat = "thinking"
		msg = fmt.Sprintf("%s: thinking | %s", model, trimmed)
	} else if strings.Contains(lower, "run_command") || strings.Contains(lower, "write_to_file") || strings.Contains(lower, "replace_file") || strings.Contains(lower, "git ") || strings.Contains(lower, "edit") || strings.Contains(lower, "build") || strings.Contains(lower, "compile") {
		cat = "develop"
		msg = fmt.Sprintf("%s: develop | %s", model, trimmed)
	} else if strings.Contains(lower, "exited with code") || strings.Contains(lower, "berjaya") || strings.Contains(lower, "success") || strings.Contains(lower, "completed") || strings.Contains(lower, "result") {
		cat = "result"
		msg = fmt.Sprintf("%s: result | %s", model, trimmed)
	} else if logType == "stderr" {
		cat = "result"
		msg = fmt.Sprintf("%s: result | [RALAT] %s", model, trimmed)
	} else {
		cat = "develop"
		msg = fmt.Sprintf("%s: develop | %s", model, trimmed)
	}

	s.appendLog(cat, msg)
}

// findNewestTranscript searches Antigravity brain directories for the newest transcript.jsonl.
func findNewestTranscript() (string, int64) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", 0
	}
	searchDirs := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "brain"),
		filepath.Join(home, ".gemini", "antigravity-ide", "brain"),
	}

	var newestFile string
	var newestTime int64

	for _, dir := range searchDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "transcript.jsonl" {
				if info.ModTime().UnixNano() > newestTime {
					newestTime = info.ModTime().UnixNano()
					newestFile = path
				}
			}
			return nil
		})
	}
	return newestFile, newestTime
}

// parseTranscriptLine turns a raw transcript JSON line into clean log components.
func parseTranscriptLine(line []byte) (timestamp string, cat string, text string, ok bool) {
	var item struct {
		CreatedAt string `json:"created_at"`
		Type      string `json:"type"`
		Content   string `json:"content"`
		ToolCalls []struct {
			Name string                 `json:"name"`
			Args map[string]interface{} `json:"args"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal(line, &item); err != nil {
		return "", "", "", false
	}

	ts := time.Now().Format("15:04:05")
	if item.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err == nil {
			ts = t.Local().Format("15:04:05")
		} else if t, err := time.Parse(time.RFC3339, item.CreatedAt); err == nil {
			ts = t.Local().Format("15:04:05")
		}
	}

	model := "gemini-3.8-flash-high"

	if item.Type == "USER_INPUT" {
		c := strings.TrimSpace(item.Content)
		if len(c) > 95 {
			c = c[:92] + "..."
		}
		return ts, "prompt", fmt.Sprintf("user: prompt | %s", c), true
	}

	if len(item.ToolCalls) > 0 {
		tc := item.ToolCalls[0]
		desc := tc.Name
		if s, ok := tc.Args["toolSummary"].(string); ok && s != "" {
			desc = s
			if a, ok := tc.Args["toolAction"].(string); ok && a != "" && a != s {
				desc = fmt.Sprintf("%s (%s)", s, a)
			}
		} else if cmd, ok := tc.Args["CommandLine"].(string); ok && cmd != "" {
			if len(cmd) > 55 {
				cmd = cmd[:52] + "..."
			}
			desc = fmt.Sprintf("%s [%s]", tc.Name, cmd)
		}
		return ts, "develop", fmt.Sprintf("%s: develop | %s", model, desc), true
	}

	if item.Content != "" {
		lines := strings.Split(strings.TrimSpace(item.Content), "\n")
		var firstLine string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "Created At:") && !strings.HasPrefix(l, "Completed At:") {
				firstLine = l
				break
			}
		}
		if firstLine != "" {
			if len(firstLine) > 110 {
				firstLine = firstLine[:107] + "..."
			}
			if item.Type == "PLANNER_RESPONSE" {
				return ts, "thinking", fmt.Sprintf("%s: thinking | %s", model, firstLine), true
			}
			return ts, "result", fmt.Sprintf("%s: result | %s", model, firstLine), true
		}
	}

	return "", "", "", false
}

func (s *Server) loadRecentTranscriptLogs() {
	newestFile, _ := findNewestTranscript()
	if newestFile == "" {
		s.appendLog("system", "system: info | [MIQA] Sedia menerima arahan dari Telegram (@miqa_agy_bot) atau CLI")
		return
	}

	fi, err := os.Stat(newestFile)
	if err != nil {
		return
	}
	s.lastTranscriptPath = newestFile
	s.lastTranscriptPos = fi.Size()

	f, err := os.Open(newestFile)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 128*1024)
	scanner.Buffer(buf, 1024*1024)

	var parsedEntries []LogEntry
	for scanner.Scan() {
		line := scanner.Bytes()
		if ts, cat, txt, ok := parseTranscriptLine(line); ok {
			parsedEntries = append(parsedEntries, LogEntry{
				Timestamp: ts,
				Type:      cat,
				Text:      txt,
			})
		}
	}

	if len(parsedEntries) > 30 {
		parsedEntries = parsedEntries[len(parsedEntries)-30:]
	}

	s.mu.Lock()
	s.terminalLogs = append(s.terminalLogs, parsedEntries...)
	s.mu.Unlock()
}

func (s *Server) pollTranscriptUpdates() {
	newestFile, _ := findNewestTranscript()
	if newestFile == "" {
		return
	}

	fi, err := os.Stat(newestFile)
	if err != nil {
		return
	}

	currentSize := fi.Size()

	if newestFile != s.lastTranscriptPath {
		s.lastTranscriptPath = newestFile
		s.lastTranscriptPos = 0
	}

	if currentSize <= s.lastTranscriptPos {
		return
	}

	f, err := os.Open(newestFile)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.Seek(s.lastTranscriptPos, io.SeekStart); err != nil {
		return
	}

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 128*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if ts, cat, txt, ok := parseTranscriptLine(line); ok {
			entry := LogEntry{
				Timestamp: ts,
				Type:      cat,
				Text:      txt,
			}
			s.mu.Lock()
			s.terminalLogs = append(s.terminalLogs, entry)
			if len(s.terminalLogs) > 300 {
				s.terminalLogs = s.terminalLogs[len(s.terminalLogs)-300:]
			}
			s.mu.Unlock()
			s.broadcastEvent("log", entry)
		}
	}
	s.lastTranscriptPos = currentSize
}

// handleSSE manages Server-Sent Events subscriptions.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak disokong", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgChan := make(chan string, 32)
	s.mu.Lock()
	s.clients[msgChan] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, msgChan)
		close(msgChan)
		s.mu.Unlock()
	}()

	// Send initial handshake
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			_, _ = fmt.Fprint(w, msg)
			flusher.Flush()
		}
	}
}

func (s *Server) broadcastEvent(eventType string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(payload))

	s.mu.RLock()
	defer s.mu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) broadcastStatsLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			path := s.cfg.CodingPath
			s.mu.RUnlock()

			// Check for live transcript updates
			s.pollTranscriptUpdates()

			stats, err := agy.GetUsageStats(path)
			if err == nil {
				s.broadcastEvent("stats_tick", stats)
			}
		}
	}
}
