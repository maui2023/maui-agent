package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maui2023/miqa/pkg/agy"
)

// MainDashboardKeyboard returns the interactive dashboard inline keyboard.
func MainDashboardKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Pilih Model", "menu:model"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Semak Usage", "menu:usage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 Laluan Path", "menu:path"),
			tgbotapi.NewInlineKeyboardButtonData("⚡ Status Sistem", "menu:status"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌐 WebUI Pixel Office", "menu:web"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Bantuan Arahan", "menu:help"),
		),
	)
}

// ModelSelectionKeyboard generates inline buttons for selecting AI models.
func ModelSelectionKeyboard(models []agy.ModelInfo, currentModel string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, m := range models {
		label := m.Name
		if m.ID == currentModel {
			label = "✅ " + m.Name
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("model:set:%s", m.ID))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Back button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Kembali ke Menu Utama", "menu:main"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// PathSelectionKeyboard generates inline buttons for sibling/sub-projects around currentPath.
func PathSelectionKeyboard(currentPath string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if abs, err := filepath.Abs(currentPath); err == nil {
		currentPath = abs
	}

	// Add parent folder's subdirectories as quick choices
	parent := filepath.Dir(currentPath)
	if entries, err := os.ReadDir(parent); err == nil {
		count := 0
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				fullPath := filepath.Join(parent, entry.Name())
				label := "📁 " + entry.Name()
				if fullPath == currentPath {
					label = "✅ " + entry.Name()
				}
				cbData := "path:set:" + fullPath
				if len(cbData) > 64 {
					cbData = "path:sub:" + entry.Name()
				}
				btn := tgbotapi.NewInlineKeyboardButtonData(label, cbData)
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
				count++
				if count >= 8 { // limit to top 8 items
					break
				}
			}
		}
	}

	// Back button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Kembali ke Menu Utama", "menu:main"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// BackToMenuKeyboard returns an inline keyboard with just a back button.
func BackToMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Kembali ke Menu Utama", "menu:main"),
		),
	)
}
