package agy

import (
	"context"
	"testing"
	"time"
)

func TestNewClientAndVerify(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("expected NewClient to succeed: %v", err)
	}

	if client.BinPath == "" {
		t.Fatalf("expected BinPath to be non-empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.VerifyStartup(ctx); err != nil {
		t.Fatalf("VerifyStartup failed: %v", err)
	}
}

func TestCheckAuthAndListModels(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	isLoggedIn, err := client.CheckAuth(ctx)
	if err != nil {
		t.Fatalf("CheckAuth failed: %v", err)
	}
	if !isLoggedIn {
		t.Fatalf("expected user to be authenticated in test environment")
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Fatalf("expected at least one model, got 0")
	}

	hasGemini := false
	for _, m := range models {
		if m.ID != "" && m.Name != "" {
			hasGemini = true
			break
		}
	}
	if !hasGemini {
		t.Errorf("expected valid model IDs and names in list")
	}
}

func TestGetUsageStats(t *testing.T) {
	stats, err := GetUsageStats(".")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if stats.MemTotalGB <= 0 {
		t.Errorf("expected MemTotalGB > 0, got %f", stats.MemTotalGB)
	}
	if stats.DiskTotGB <= 0 {
		t.Errorf("expected DiskTotGB > 0, got %f", stats.DiskTotGB)
	}

	formatted := stats.FormatTelegramMarkdown("gemini-3.8-flash-high", "/tmp")
	if formatted == "" {
		t.Errorf("expected formatted markdown string, got empty")
	}
}
