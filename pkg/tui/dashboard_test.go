package tui

import (
	"context"
	"testing"
	"time"

	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
)

func TestDashboardManagerToggles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{
		Model:      "gemini-3.8-flash-high",
		CodingPath: ".",
	}
	client, err := agy.NewClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dm := &DashboardManager{
		ctx:       ctx,
		agyClient: client,
		cfg:       cfg,
		webPort:   8999, // Use non-standard port for test
	}

	if dm.webRunning {
		t.Errorf("expected WebUI not running initially")
	}

	// Test starting WebUI in background
	go func() {
		// Run ToggleWebUI which has a short sleep
		dm.ToggleWebUI()
	}()

	time.Sleep(300 * time.Millisecond)
	dm.mu.Lock()
	running := dm.webRunning
	dm.mu.Unlock()

	if !running {
		t.Errorf("expected WebUI to be running after ToggleWebUI")
	}

	// Test stopping WebUI
	go func() {
		dm.ToggleWebUI()
	}()

	time.Sleep(300 * time.Millisecond)
	dm.mu.Lock()
	runningAfterStop := dm.webRunning
	dm.mu.Unlock()

	if runningAfterStop {
		t.Errorf("expected WebUI to be stopped after second ToggleWebUI")
	}

	// Clean shutdown
	dm.ShutdownAll()
}
