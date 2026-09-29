package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
)

func TestWebServerEndpoints(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{
		Model:      "gemini-3.8-flash-high",
		CodingPath: ".",
	}
	client, err := agy.NewClient()
	if err != nil {
		t.Fatalf("failed to create agy client: %v", err)
	}

	server := NewServer("127.0.0.1", 8080, cfg, client)

	// Test /api/status
	reqStatus := httptest.NewRequest("GET", "/api/status", nil)
	wStatus := httptest.NewRecorder()
	server.handleStatus(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", wStatus.Code)
	}

	var statusResp map[string]interface{}
	if err := json.Unmarshal(wStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to parse status json: %v", err)
	}
	if statusResp["active_model"] != "gemini-3.8-flash-high" {
		t.Errorf("expected active model gemini-3.8-flash-high, got %v", statusResp["active_model"])
	}

	// Test /api/models
	reqModels := httptest.NewRequest("GET", "/api/models", nil)
	wModels := httptest.NewRecorder()
	server.handleModels(wModels, reqModels)

	if wModels.Code != http.StatusOK {
		t.Errorf("expected status 200 for models, got %d", wModels.Code)
	}

	var modelsList []ModelState
	if err := json.Unmarshal(wModels.Body.Bytes(), &modelsList); err != nil {
		t.Fatalf("failed to parse models json: %v", err)
	}
	if len(modelsList) == 0 {
		t.Errorf("expected at least 1 model in states, got %d", len(modelsList))
	}

	// Test /api/set-model
	setModelBody, _ := json.Marshal(map[string]string{"model": "claude-sonnet-4-6"})
	reqSetModel := httptest.NewRequest("POST", "/api/set-model", bytes.NewBuffer(setModelBody))
	wSetModel := httptest.NewRecorder()
	server.handleSetModel(wSetModel, reqSetModel)

	if wSetModel.Code != http.StatusOK {
		t.Errorf("expected status 200 for set-model, got %d", wSetModel.Code)
	}
	if server.activeModel != "claude-sonnet-4-6" {
		t.Errorf("expected active model claude-sonnet-4-6, got %s", server.activeModel)
	}

	// Test /api/set-state (simulate exhausted/hungry state)
	setStateBody, _ := json.Marshal(map[string]interface{}{
		"model":   "claude-sonnet-4-6",
		"status":  "exhausted",
		"credits": 0,
	})
	reqSetState := httptest.NewRequest("POST", "/api/set-state", bytes.NewBuffer(setStateBody))
	wSetState := httptest.NewRecorder()
	server.handleSetState(wSetState, reqSetState)

	if wSetState.Code != http.StatusOK {
		t.Errorf("expected status 200 for set-state, got %d", wSetState.Code)
	}
	if server.modelStates["claude-sonnet-4-6"].Status != "exhausted" {
		t.Errorf("expected model state exhausted, got %s", server.modelStates["claude-sonnet-4-6"].Status)
	}

	// Test /api/run is forbidden for public safety
	runBody, _ := json.Marshal(map[string]string{"prompt": "echo test"})
	reqRun := httptest.NewRequest("POST", "/api/run", bytes.NewBuffer(runBody))
	wRun := httptest.NewRecorder()
	server.handleRunPrompt(wRun, reqRun)
	if wRun.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden on /api/run for public security, got %d", wRun.Code)
	}

	// Test direct task start and finish
	server.handleTaskStart("claude-sonnet-4-6", "refactor code", "telegram")
	if server.modelStates["claude-sonnet-4-6"].Status != "working" {
		t.Errorf("expected working status after task start, got %s", server.modelStates["claude-sonnet-4-6"].Status)
	}

	server.handleTaskFinish("claude-sonnet-4-6", 500*1000*1000, true)
	if server.modelStates["claude-sonnet-4-6"].Status != "idle" {
		t.Errorf("expected idle status after task finish, got %s", server.modelStates["claude-sonnet-4-6"].Status)
	}

	// Test internal stream handler for cross-process events
	streamStartBody, _ := json.Marshal(map[string]interface{}{
		"action": "start",
		"model":  "claude-sonnet-4-6",
		"prompt": "test prompt",
		"source": "cli",
	})
	reqStreamStart := httptest.NewRequest("POST", "/api/internal/stream", bytes.NewBuffer(streamStartBody))
	wStreamStart := httptest.NewRecorder()
	server.handleInternalStream(wStreamStart, reqStreamStart)
	if wStreamStart.Code != http.StatusOK {
		t.Errorf("expected 200 on internal stream start, got %d", wStreamStart.Code)
	}

	streamLogBody, _ := json.Marshal(map[string]interface{}{
		"action": "log",
		"type":   "stdout",
		"text":   "hello streaming output",
	})
	reqStreamLog := httptest.NewRequest("POST", "/api/internal/stream", bytes.NewBuffer(streamLogBody))
	wStreamLog := httptest.NewRecorder()
	server.handleInternalStream(wStreamLog, reqStreamLog)
	if wStreamLog.Code != http.StatusOK {
		t.Errorf("expected 200 on internal stream log, got %d", wStreamLog.Code)
	}

	streamFinishBody, _ := json.Marshal(map[string]interface{}{
		"action":   "finish",
		"model":    "claude-sonnet-4-6",
		"duration": "1.5s",
		"success":  true,
	})
	reqStreamFinish := httptest.NewRequest("POST", "/api/internal/stream", bytes.NewBuffer(streamFinishBody))
	wStreamFinish := httptest.NewRecorder()
	server.handleInternalStream(wStreamFinish, reqStreamFinish)
	if wStreamFinish.Code != http.StatusOK {
		t.Errorf("expected 200 on internal stream finish, got %d", wStreamFinish.Code)
	}

	// Test Global Broadcast functions with active server registered
	RegisterServer(server)
	defer UnregisterServer(server)

	BroadcastExecutionStart("claude-sonnet-4-6", "running test task", "cli")
	BroadcastExecutionLog("stdout", "test streaming log line")
	BroadcastExecutionFinish("claude-sonnet-4-6", 500*1000*1000, true)
}


