package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/charmbracelet/lipgloss"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
	"github.com/maui2023/miqa/pkg/telegram"
	"github.com/maui2023/miqa/pkg/web"
)

// ShowLoginNotice displays instructions when the user has not logged into agy.
func ShowLoginNotice() {
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(errorColor).
		Padding(1, 2).
		Margin(1, 0)

	msg := lipgloss.NewStyle().Bold(true).Foreground(errorColor).Render("⛔ PENGESAHAN LOG MASUK DIPERLUKAN") + "\n\n" +
		"Sesi Antigravity anda belum disahkan (belum log masuk).\n\n" +
		lipgloss.NewStyle().Bold(true).Render("Langkah-langkah untuk log masuk:") + "\n" +
		"  1. Buka terminal dan jalankan arahan: " + lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("agy") + "\n" +
		"  2. Ikuti arahan pada skrin untuk log masuk ke akaun anda.\n" +
		"  3. Selepas selesai, jalankan semula " + lipgloss.NewStyle().Bold(true).Foreground(successColor).Render("miqa") + ".\n"

	fmt.Println(box.Render(msg))
}

// PromptTelegramConfig prompts the user for Telegram Bot Token and Chat ID.
func PromptTelegramConfig(cfg *config.Config) error {
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("⚙️  KONFIGURASI BOT TELEGRAM"))
	fmt.Println(styleMuted.Render("Tetapan ini membolehkan kawalan dua hala miqa melalui Telegram."))
	fmt.Println()

	var token string
	promptToken := &survey.Password{
		Message: "Masukkan Telegram Bot Token (daripada @BotFather):",
	}
	if cfg.TelegramToken != "" {
		promptToken.Message = fmt.Sprintf("Masukkan Telegram Bot Token [Tekan Enter untuk kekalkan sedia ada]:")
	}

	if err := survey.AskOne(promptToken, &token); err != nil {
		return err
	}

	token = strings.TrimSpace(token)
	if token != "" {
		// Validate token with Telegram API
		fmt.Print("⏳ Mengesahkan token Telegram... ")
		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			fmt.Println(styleError.Render("GAGAL"))
			fmt.Printf("❌ Token tidak sah: %v\n", err)
			return err
		}
		fmt.Println(styleSuccess.Render(fmt.Sprintf("BERJAYA! (@%s)", bot.Self.UserName)))
		cfg.TelegramToken = token
	}

	var chatIDInput string
	promptChatID := &survey.Input{
		Message: "Masukkan Chat ID Telegram (pilihan, kosongkan untuk tangkap secara automatik semasa /start):",
		Default: fmt.Sprintf("%d", cfg.TelegramChatID),
	}
	if cfg.TelegramChatID == 0 {
		promptChatID.Default = ""
	}

	if err := survey.AskOne(promptChatID, &chatIDInput); err != nil {
		return err
	}

	chatIDInput = strings.TrimSpace(chatIDInput)
	if chatIDInput != "" {
		var id int64
		if _, err := fmt.Sscanf(chatIDInput, "%d", &id); err == nil {
			cfg.TelegramChatID = id
		}
	}

	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("gagal menyimpan konfigurasi: %w", err)
	}

	fmt.Println(successBadge.Render("✓") + " Konfigurasi Telegram berjaya disimpan di " + config.GetConfigPath())
	fmt.Println()
	return nil
}

// PromptSelectModel allows the user to choose an AI model from the available agy models.
func PromptSelectModel(ctx context.Context, agyClient *agy.Client, cfg *config.Config) error {
	fmt.Println()
	fmt.Print("⏳ Memuatkan senarai model daripada agy... ")
	models, err := agyClient.ListModels(ctx)
	if err != nil {
		fmt.Println(styleError.Render("RALAT"))
		return err
	}
	fmt.Println(styleSuccess.Render("SIAP"))

	var options []string
	modelMap := make(map[string]string)
	defaultIndex := 0

	for i, m := range models {
		label := fmt.Sprintf("%s (%s)", m.Name, m.ID)
		options = append(options, label)
		modelMap[label] = m.ID
		if m.ID == cfg.Model {
			defaultIndex = i
		}
	}

	var selectedLabel string
	prompt := &survey.Select{
		Message: "Pilih Model AI untuk sesi pengekodan:",
		Options: options,
		Default: options[defaultIndex],
	}

	if err := survey.AskOne(prompt, &selectedLabel); err != nil {
		return err
	}

	cfg.Model = modelMap[selectedLabel]
	if err := cfg.SaveConfig(); err != nil {
		return err
	}

	fmt.Println(successBadge.Render("✓") + " Model aktif ditetapkan kepada: " + lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(cfg.Model))
	fmt.Println()
	return nil
}

// PromptSelectPath allows the user to specify or select a coding directory.
func PromptSelectPath(cfg *config.Config) error {
	fmt.Println()
	var newPath string
	prompt := &survey.Input{
		Message: "Masukkan laluan direktori pengekodan (workspace path):",
		Default: cfg.CodingPath,
	}

	if err := survey.AskOne(prompt, &newPath); err != nil {
		return err
	}

	newPath = strings.TrimSpace(newPath)
	validPath, err := telegram.ValidatePath(newPath)
	if err != nil {
		fmt.Println(styleError.Render("❌ " + err.Error()))
		return err
	}

	absPath, err := filepath.Abs(validPath)
	if err == nil {
		validPath = absPath
	}

	cfg.CodingPath = validPath
	if err := cfg.SaveConfig(); err != nil {
		return err
	}

	fmt.Println(successBadge.Render("✓") + " Laluan pengekodan aktif: " + lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(cfg.CodingPath))
	fmt.Println()
	return nil
}

// DisplayUsage prints current usage and system resources to the terminal.
func DisplayUsage(cfg *config.Config) {
	fmt.Println()
	stats, err := agy.GetUsageStats(cfg.CodingPath)
	if err != nil {
		fmt.Println(styleError.Render("❌ Gagal membaca statistik: " + err.Error()))
		return
	}

	items := map[string]string{
		"Model Aktif":         cfg.Model,
		"Laluan Folder":       cfg.CodingPath,
		"Sesi Antigravity":    fmt.Sprintf("%d sesi", stats.TotalSessions),
		"Jumlah Langkah":      fmt.Sprintf("%d langkah", stats.TotalSteps),
		"Memori (RAM)":        fmt.Sprintf("%.1f GB digunakan / %.1f GB jumlah", stats.MemUsedGB, stats.MemTotalGB),
		"Storan Cakera Bebas": fmt.Sprintf("%.1f GB bebas / %.1f GB jumlah", stats.DiskFreeGB, stats.DiskTotGB),
		"Beban CPU (1m)":      stats.CPULoad1m,
	}

	PrintStatusBox("📊 PENGGUNAAN & SUMBER SISTEM", items)

	if len(stats.RecentSessions) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("🕒 Sesi Terkini:"))
		for i, s := range stats.RecentSessions {
			fmt.Printf("  %d. %-32s (%d langkah) - %s\n", i+1, s.Title, s.StepCount, s.LastModified)
		}
		fmt.Println()
	}
}

// PromptExecutePrompt allows immediate command execution from CLI.
func PromptExecutePrompt(ctx context.Context, agyClient *agy.Client, cfg *config.Config) error {
	fmt.Println()
	var promptText string
	prompt := &survey.Input{
		Message: "Masukkan arahan pengekodan (prompt untuk agy):",
	}

	if err := survey.AskOne(prompt, &promptText); err != nil {
		return err
	}

	promptText = strings.TrimSpace(promptText)
	if promptText == "" {
		return nil
	}

	fmt.Println()
	fmt.Println(styleMuted.Render(fmt.Sprintf("⏳ Menjalankan arahan dalam '%s' menggunakan model '%s'...", cfg.CodingPath, cfg.Model)))
	web.BroadcastExecutionStart(cfg.Model, promptText, "TUI Dashboard (CLI)")
	start := time.Now()

	result, err := agyClient.RunPromptStreaming(ctx, promptText, cfg.Model, cfg.CodingPath, func(streamType, line string) {
		web.BroadcastExecutionLog(streamType, line)
		fmt.Println(line)
	})
	duration := time.Since(start).Round(time.Millisecond)
	web.BroadcastExecutionFinish(cfg.Model, duration, err == nil)

	if err != nil {
		fmt.Println(styleError.Render("❌ Ralat pelaksanaan agy: " + err.Error()))
		return err
	}

	fmt.Println()
	fmt.Println(styleSuccess.Render(fmt.Sprintf("✅ Selesai dalam %s:", duration)))
	if result != "" {
		fmt.Println(cardStyle.Render(result))
	}
	return nil
}
