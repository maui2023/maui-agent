package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
	"github.com/maui2023/miqa/pkg/telegram"
	"github.com/maui2023/miqa/pkg/web"
)

// DashboardManager coordinates background services (WebUI, Telegram Bot) and interactive CLI navigation.
type DashboardManager struct {
	ctx       context.Context
	agyClient *agy.Client
	cfg       *config.Config
	mu        sync.Mutex

	// WebUI Service
	webRunning bool
	webCancel  context.CancelFunc
	webPort    int

	// Telegram Bot Service
	telegramRunning  bool
	telegramCancel   context.CancelFunc
	telegramUsername string
}

// RunDashboard starts the main interactive menu loop with background service management.
func RunDashboard(ctx context.Context, agyClient *agy.Client, cfg *config.Config) error {
	dm := &DashboardManager{
		ctx:       ctx,
		agyClient: agyClient,
		cfg:       cfg,
		webPort:   8080,
	}
	defer dm.ShutdownAll()

	return dm.Loop()
}

// Loop runs the main dashboard prompt loop.
func (dm *DashboardManager) Loop() error {
	for {
		select {
		case <-dm.ctx.Done():
			return nil
		default:
		}

		dm.mu.Lock()
		webStatusStr := styleMuted.Render("⚪ Tidak Aktif (Mati)")
		if dm.webRunning {
			webStatusStr = styleSuccess.Render(fmt.Sprintf("🟢 Aktif (http://localhost:%d)", dm.webPort))
		}

		tgStatusStr := styleMuted.Render("⚪ Tidak Aktif (Mati)")
		if dm.telegramRunning {
			tgStatusStr = styleSuccess.Render(fmt.Sprintf("🟢 Aktif (%s)", dm.telegramUsername))
		} else if !dm.cfg.IsTelegramConfigured() {
			tgStatusStr = styleWarning.Render("⚠️ Belum Dikonfigurasi (Perlu Token)")
		}

		infoItems := map[string]string{
			"Model AI Semasa":   dm.cfg.Model,
			"Laluan Folder":     dm.cfg.CodingPath,
			"🌐 Pelayan WebUI":  webStatusStr,
			"🤖 Bot Telegram":   tgStatusStr,
			"Pintasan Keizinan": "--dangerously-skip-permissions (Aktif)",
		}
		PrintStatusBox("PAPAN PEMUKA UTAMA MIQA", infoItems)

		// Dynamic menu actions based on service states
		webActionLabel := fmt.Sprintf("4. [MULA] Pelayan WebUI -> (Buka di http://localhost:%d)", dm.webPort)
		if dm.webRunning {
			webActionLabel = fmt.Sprintf("4. [HENTI] Pelayan WebUI -> (Hentikan http://localhost:%d)", dm.webPort)
		}

		tgActionLabel := "5. [MULA] Bot Telegram -> (Aktifkan bot di latar belakang)"
		if dm.telegramRunning {
			tgActionLabel = fmt.Sprintf("5. [HENTI] Bot Telegram -> (Hentikan perkhidmatan %s)", dm.telegramUsername)
		}
		dm.mu.Unlock()

		menuOptions := []string{
			"1. Pilih Model AI (Select Model)",
			"2. Semak Penggunaan (Check Usage)",
			"3. Tetapkan Laluan Pengekodan (Select Path for Coding)",
			webActionLabel,
			tgActionLabel,
			"6. Jalankan Arahan Kod Terus (Run Prompt in CLI)",
			"7. Konfigurasi Telegram Bot (Setup Telegram)",
			"8. Keluar (Exit)",
		}

		var choice string
		prompt := &survey.Select{
			Message:  "Pilih tindakan yang ingin dilakukan:",
			Options:  menuOptions,
			PageSize: 8,
		}

		if err := survey.AskOne(prompt, &choice); err != nil {
			return nil
		}

		switch choice {
		case menuOptions[0]: // Select Model
			_ = PromptSelectModel(dm.ctx, dm.agyClient, dm.cfg)

		case menuOptions[1]: // Check Usage
			DisplayUsage(dm.cfg)

		case menuOptions[2]: // Select Path
			_ = PromptSelectPath(dm.cfg)

		case menuOptions[3]: // Toggle WebUI (Start/Stop)
			dm.ToggleWebUI()

		case menuOptions[4]: // Toggle Telegram Bot (Start/Stop)
			dm.ToggleTelegramBot()

		case menuOptions[5]: // Run Prompt CLI
			_ = PromptExecutePrompt(dm.ctx, dm.agyClient, dm.cfg)

		case menuOptions[6]: // Config Telegram
			_ = PromptTelegramConfig(dm.cfg)

		case menuOptions[7]: // Exit
			dm.ShutdownAll()
			fmt.Println(stylePrimary.Render("\nTerima kasih kerana menggunakan miqa! Jumpa lagi.\n"))
			return nil
		}
	}
}

// ToggleWebUI starts or stops the WebUI background server.
func (dm *DashboardManager) ToggleWebUI() {
	dm.mu.Lock()
	if dm.webRunning {
		// Stop WebUI
		if dm.webCancel != nil {
			dm.webCancel()
		}
		dm.webRunning = false
		dm.mu.Unlock()

		fmt.Println()
		notice := cardStyle.Render(
			styleWarning.Render("🛑 PELAYAN WEBUI BERJAYA DIHENTIKAN") + "\n\n" +
				styleMuted.Render("Pelayan WebUI di latar belakang telah dimatikan."),
		)
		fmt.Println(notice)
		time.Sleep(1400 * time.Millisecond)
		return
	}

	// Start WebUI
	webCtx, cancel := context.WithCancel(context.Background())
	dm.webCancel = cancel
	dm.webRunning = true
	port := dm.webPort
	dm.mu.Unlock()

	srv := web.NewServer("0.0.0.0", port, dm.cfg, dm.agyClient)
	go func() {
		if err := srv.Start(webCtx); err != nil {
			dm.mu.Lock()
			dm.webRunning = false
			dm.mu.Unlock()
		}
	}()

	fmt.Println()
	notice := cardStyle.Render(
		styleSuccess.Render("✅ PELAYAN WEBUI BERJAYA DIMULAKAN DI LATAR BELAKANG!") + "\n\n" +
			lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(fmt.Sprintf("🌐 Layari: http://localhost:%d", port)) + "\n" +
			styleMuted.Render("🏢 Animated Pixel Office, Graf Sistem, dan Terminal Langsung kini aktif.") + "\n" +
			styleMuted.Render("Kembali ke menu utama dalam seketika..."),
	)
	fmt.Println(notice)
	time.Sleep(1800 * time.Millisecond)
}

// ToggleTelegramBot starts or stops the Telegram Bot background service.
func (dm *DashboardManager) ToggleTelegramBot() {
	dm.mu.Lock()
	if dm.telegramRunning {
		// Stop Telegram Bot
		if dm.telegramCancel != nil {
			dm.telegramCancel()
		}
		dm.telegramRunning = false
		username := dm.telegramUsername
		dm.mu.Unlock()

		fmt.Println()
		notice := cardStyle.Render(
			styleWarning.Render("🛑 BOT TELEGRAM BERJAYA DIHENTIKAN") + "\n\n" +
				styleMuted.Render(fmt.Sprintf("Perkhidmatan bot %s telah dimatikan.", username)),
		)
		fmt.Println(notice)
		time.Sleep(1400 * time.Millisecond)
		return
	}

	// Check if Telegram is configured
	if !dm.cfg.IsTelegramConfigured() {
		dm.mu.Unlock()
		fmt.Println()
		fmt.Println(styleWarning.Render("⚠️  Telegram Bot belum dikonfigurasi. Sila masukkan token terlebih dahulu."))
		if err := PromptTelegramConfig(dm.cfg); err != nil {
			return
		}
		dm.mu.Lock()
	}

	// Start Telegram Bot
	botService, err := telegram.NewBotService(dm.cfg, dm.agyClient)
	if err != nil {
		dm.mu.Unlock()
		fmt.Println()
		fmt.Println(styleError.Render("❌ Gagal memulakan Telegram bot: " + err.Error()))
		time.Sleep(1500 * time.Millisecond)
		return
	}

	tgCtx, cancel := context.WithCancel(context.Background())
	dm.telegramCancel = cancel
	dm.telegramRunning = true
	dm.telegramUsername = botService.GetBotUsername()
	username := dm.telegramUsername
	dm.mu.Unlock()

	go func() {
		if err := botService.Start(tgCtx); err != nil {
			dm.mu.Lock()
			dm.telegramRunning = false
			dm.mu.Unlock()
		}
	}()

	fmt.Println()
	notice := cardStyle.Render(
		styleSuccess.Render("✅ BOT TELEGRAM BERJAYA DIMULAKAN DI LATAR BELAKANG!") + "\n\n" +
			lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(fmt.Sprintf("🤖 Bot Aktif: %s", username)) + "\n" +
			styleMuted.Render("Sedia menerima arahan pengekodan dan pertanyaan terus di Telegram.") + "\n" +
			styleMuted.Render("Kembali ke menu utama dalam seketika..."),
	)
	fmt.Println(notice)
	time.Sleep(1800 * time.Millisecond)
}

// ShutdownAll cleanly terminates all active background services.
func (dm *DashboardManager) ShutdownAll() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.webRunning && dm.webCancel != nil {
		dm.webCancel()
		dm.webRunning = false
	}
	if dm.telegramRunning && dm.telegramCancel != nil {
		dm.telegramCancel()
		dm.telegramRunning = false
	}
}
