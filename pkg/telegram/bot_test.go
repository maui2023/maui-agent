package telegram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maui2023/miqa/pkg/agy"
)

func TestValidatePath(t *testing.T) {
	tempDir := t.TempDir()

	validPath, err := ValidatePath(tempDir)
	if err != nil {
		t.Fatalf("expected valid path, got error: %v", err)
	}
	if validPath != tempDir {
		t.Errorf("expected %s, got %s", tempDir, validPath)
	}

	// Test non-existent path
	nonExistent := filepath.Join(tempDir, "does-not-exist-12345")
	if _, err := ValidatePath(nonExistent); err == nil {
		t.Errorf("expected error for non-existent path, got nil")
	}

	// Test file instead of dir
	tempFile := filepath.Join(tempDir, "sample.txt")
	_ = os.WriteFile(tempFile, []byte("hello"), 0644)
	if _, err := ValidatePath(tempFile); err == nil {
		t.Errorf("expected error for file path, got nil")
	}
}

func TestKeyboards(t *testing.T) {
	tempDir := t.TempDir()
	mainKb := MainDashboardKeyboard()
	if len(mainKb.InlineKeyboard) < 3 {
		t.Errorf("expected at least 3 rows in main keyboard, got %d", len(mainKb.InlineKeyboard))
	}

	models := []agy.ModelInfo{
		{ID: "gemini-3.8-flash-high", Name: "Gemini 3.8 Flash (High)"},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6"},
	}

	modelKb := ModelSelectionKeyboard(models, "gemini-3.8-flash-high")
	if len(modelKb.InlineKeyboard) != 3 { // 2 models + 1 back button
		t.Errorf("expected 3 rows in model keyboard, got %d", len(modelKb.InlineKeyboard))
	}

	// Verify active checkmark
	if modelKb.InlineKeyboard[0][0].Text != "✅ Gemini 3.8 Flash (High)" {
		t.Errorf("expected active model checkmark, got '%s'", modelKb.InlineKeyboard[0][0].Text)
	}

	// Test PathSelectionKeyboard
	subDir1 := filepath.Join(tempDir, "proj1")
	subDir2 := filepath.Join(tempDir, "proj2")
	_ = os.MkdirAll(subDir1, 0755)
	_ = os.MkdirAll(subDir2, 0755)

	pathKb := PathSelectionKeyboard(subDir1)
	if len(pathKb.InlineKeyboard) < 2 {
		t.Errorf("expected at least 2 rows in path keyboard, got %d", len(pathKb.InlineKeyboard))
	}
}

func TestValidatePathCleaning(t *testing.T) {
	tempDir := t.TempDir()

	// Path with quotes
	quoted := `"` + tempDir + `"`
	p, err := ValidatePath(quoted)
	if err != nil || p != tempDir {
		t.Errorf("failed to clean quoted path: got %s, err %v", p, err)
	}

	// Path with trailing spaces and backticks
	backticked := "`" + tempDir + " ` "
	p, err = ValidatePath(backticked)
	if err != nil || p != tempDir {
		t.Errorf("failed to clean backticked path: got %s, err %v", p, err)
	}
}

func TestBotAwaitingPath(t *testing.T) {
	bot := &BotService{
		awaitingPath: make(map[int64]bool),
	}

	chatID := int64(123456)
	if bot.IsAwaitingPath(chatID) {
		t.Errorf("expected false initially")
	}

	bot.SetAwaitingPath(chatID, true)
	if !bot.IsAwaitingPath(chatID) {
		t.Errorf("expected true after SetAwaitingPath true")
	}

	bot.SetAwaitingPath(chatID, false)
	if bot.IsAwaitingPath(chatID) {
		t.Errorf("expected false after SetAwaitingPath false")
	}
}
