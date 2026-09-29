package telegram

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/web"
)

// handleIncomingMessage processes all incoming user text messages and commands.
func (b *BotService) handleIncomingMessage(msg *tgbotapi.Message) {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return
	}

	chatID := msg.Chat.ID
	if !b.isAuthorized(chatID) {
		_, _ = b.sendMessage(chatID, "⛔ *Akses Ditolak*\nChat ID anda tidak dibenarkan mengakses bot miqa ini.")
		return
	}

	text := strings.TrimSpace(msg.Text)

	// If bot is currently awaiting path input from this user
	if b.IsAwaitingPath(chatID) {
		if text == "/cancel" {
			b.SetAwaitingPath(chatID, false)
			_, _ = b.sendMessage(chatID, "❌ *Penetapan laluan dibatalkan.*")
			b.sendDashboard(chatID)
			return
		}
		if text == "/start" || text == "/menu" {
			b.SetAwaitingPath(chatID, false)
			b.sendDashboard(chatID)
			return
		}

		cleanCandidate := text
		if strings.HasPrefix(strings.ToLower(cleanCandidate), "/path") {
			parts := strings.Fields(cleanCandidate)
			if len(parts) > 1 {
				cleanCandidate = strings.TrimSpace(strings.TrimPrefix(cleanCandidate, parts[0]))
			}
		}

		b.handlePathCommand(chatID, cleanCandidate)
		return
	}

	// Command routing
	if strings.HasPrefix(text, "/") {
		parts := strings.Fields(text)
		cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
		// Strip bot username if mentioned (e.g. /menu@mybot)
		if idx := strings.Index(cmd, "@"); idx != -1 {
			cmd = cmd[:idx]
		}

		args := ""
		if len(parts) > 1 {
			args = strings.TrimSpace(strings.TrimPrefix(text, parts[0]))
		}

		switch cmd {
		case "start", "menu":
			b.sendDashboard(chatID)
		case "model":
			b.sendModelSelection(chatID)
		case "usage":
			b.sendUsage(chatID)
		case "path":
			b.handlePathCommand(chatID, args)
		case "status":
			b.sendStatus(chatID)
		case "web":
			b.sendWebLink(chatID)
		case "cancel":
			b.handleCancel(chatID)
		case "help":
			b.sendHelp(chatID)
		default:
			// If text starts with / and points to a valid local directory (e.g. /home/maui/github/payvoucher)
			if validPath, err := ValidatePath(text); err == nil {
				b.handlePathCommand(chatID, validPath)
				return
			}
			_, _ = b.sendMessage(chatID, fmt.Sprintf("Arahan `/%s` tidak dikenali. Taip /menu atau /help.", cmd))
		}
		return
	}

	// Check if plain text is a valid directory (e.g. ~/github/payvoucher)
	if strings.HasPrefix(text, "~") {
		if validPath, err := ValidatePath(text); err == nil {
			b.handlePathCommand(chatID, validPath)
			return
		}
	}

	// Plain text message: Treat as Coding / AI Prompt
	b.executeCodingPrompt(chatID, text)
}

// sendDashboard sends the main interactive menu dashboard.
func (b *BotService) sendDashboard(chatID int64) {
	text := fmt.Sprintf(
		"👋 *Selamat Datang ke miqa AI Coding Agent!*\n\n"+
			"Sistem ejen pintar terminal & Telegram sedia membantu anda dalam sesi pengekodan pantas dengan mod `--dangerously-skip-permissions` aktif.\n\n"+
			"🔹 *Model Aktif:* `%s`\n"+
			"📁 *Laluan Kod:* `%s`\n\n"+
			"💡 _Hantar sebarang arahan atau kod untuk mula menjana terus melalui agy, atau pilih menu di bawah:_",
		b.cfg.Model,
		b.cfg.CodingPath,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = MainDashboardKeyboard()
	_, _ = b.api.Send(msg)
}

// sendModelSelection displays available AI models as inline buttons.
func (b *BotService) sendModelSelection(chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	models, err := b.agyClient.ListModels(ctx)
	if err != nil {
		_, _ = b.sendMessage(chatID, "❌ Gagal memuatkan senarai model: "+err.Error())
		return
	}

	text := fmt.Sprintf("🤖 *PILIH MODEL AI*\n\nModel semasa: `%s`\n\nSila pilih model pilihan anda dari senarai di bawah:", b.cfg.Model)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = ModelSelectionKeyboard(models, b.cfg.Model)
	_, _ = b.api.Send(msg)
}

// sendUsage displays usage and system metrics.
func (b *BotService) sendUsage(chatID int64) {
	stats, err := agy.GetUsageStats(b.cfg.CodingPath)
	if err != nil {
		_, _ = b.sendMessage(chatID, "❌ Gagal mendapatkan statistik: "+err.Error())
		return
	}

	text := stats.FormatTelegramMarkdown(b.cfg.Model, b.cfg.CodingPath)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = BackToMenuKeyboard()
	_, _ = b.api.Send(msg)
}

// handlePathCommand displays or changes the coding workspace path.
func (b *BotService) handlePathCommand(chatID int64, newPath string) {
	newPath = strings.TrimSpace(newPath)
	if newPath == "" {
		b.SetAwaitingPath(chatID, true)
		text := fmt.Sprintf(
			"📁 *TETAPKAN LALUAN PENGEKODAN*\n\n"+
				"Laluan semasa:\n`%s`\n\n"+
				"💡 _Pilih direktori cadangan di bawah atau hantar laluan folder baharu dalam chat:_\n"+
				"Contoh: `/path /home/maui/github/payvoucher`",
			b.cfg.CodingPath,
		)
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = PathSelectionKeyboard(b.cfg.CodingPath)
		_, _ = b.api.Send(msg)
		return
	}

	validPath, err := ValidatePath(newPath)
	if err != nil {
		_, _ = b.sendMessage(chatID, "❌ "+err.Error()+"\n\nSila pastikan folder wujud di mesin hos.")
		return
	}

	b.cfg.CodingPath = validPath
	_ = b.cfg.SaveConfig()
	b.SetAwaitingPath(chatID, false)
	log.Printf("[miqa-bot] Laluan pengekodan berjaya dikemaskini kepada: %s", validPath)

	text := fmt.Sprintf(
		"✅ *Laluan Berjaya Dikemaskini!*\n\n"+
			"📁 *Laluan Kod Baharu:*\n`%s`\n\n"+
			"Sistem kini bersedia menjana atau menyunting kod di direktori ini.",
		validPath,
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = MainDashboardKeyboard()
	_, _ = b.api.Send(msg)
}

// sendStatus displays overall status of miqa and agy.
func (b *BotService) sendStatus(chatID int64) {
	uptime := time.Since(b.startTime).Round(time.Second)

	text := fmt.Sprintf(
		"⚡ *STATUS SISTEM MIQA*\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"🤖 *Bot Telegram:* `%s` (Aktif)\n"+
			"🧠 *Model Pilihan:* `%s`\n"+
			"📁 *Laluan Kod:* `%s`\n"+
			"🛡️ *Mod Pintasan:* `--dangerously-skip-permissions` (Aktif)\n"+
			"⏱️ *Masa Aktif Bot:* %s\n"+
			"🟢 *Status Enjin:* agy bersedia menerima arahan",
		b.GetBotUsername(),
		b.cfg.Model,
		b.cfg.CodingPath,
		uptime.String(),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = BackToMenuKeyboard()
	_, _ = b.api.Send(msg)
}

// sendHelp displays user instructions and command guide.
func (b *BotService) sendHelp(chatID int64) {
	text := "📖 *PANDUAN PENGGUNAAN MIQA*\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"🔹 */menu* - Membuka menu papan pemuka interaktif\n" +
		"🔹 */model* - Memilih atau menukar model AI pilihan\n" +
		"🔹 */usage* - Semak penggunaan token, sesi & sumber sistem\n" +
		"🔹 */path <folder>* - Melihat atau menukar folder kerja pengekodan\n" +
		"🔹 */status* - Semak status semasa bot dan enjin agy\n" +
		"🔹 */cancel* - Membatalkan tugas agy yang sedang berlangsung\n" +
		"🔹 */help* - Paparkan panduan arahan ini\n\n" +
		"💬 *Cara Menulis Kod Melalui Telegram:*\n" +
		"Anda hanya perlu hantar sebarang teks atau arahan seperti:\n" +
		"_\"Buat fail hello.go dan cetak hello world\"_\n" +
		"miqa akan menghantar arahan tersebut terus kepada `agy` dengan mod automatik!"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = BackToMenuKeyboard()
	_, _ = b.api.Send(msg)
}

// sendWebLink sends the WebUI URL to the user.
func (b *BotService) sendWebLink(chatID int64) {
	text := "🌐 *MIQA WEBUI DASHBOARD*\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Saksikan *Animated Pixel Office*, graf telemetri sistem secara langsung, dan konsol terminal agy masa nyata:\n\n" +
		"🔗 [Buka WebUI MIQA](http://localhost:8080)\n`http://localhost:8080`\n\n" +
		"_(Pastikan pelayan web dimulakan dengan arahan `miqa web` atau melalui menu CLI)_"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = BackToMenuKeyboard()
	_, _ = b.api.Send(msg)
}

// handleCancel cancels any currently executing prompt.
func (b *BotService) handleCancel(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isExecuting || b.cancelExecution == nil {
		_, _ = b.sendMessage(chatID, "ℹ️ Tiada tugasan agy yang sedang berjalan ketika ini.")
		return
	}

	b.cancelExecution()
	b.isExecuting = false
	_, _ = b.sendMessage(chatID, "🛑 *Tugasan dibatalkan:* Arahan agy telah dihentikan oleh pengguna.")
}

// executeCodingPrompt runs an agy coding prompt with background typing notification.
func (b *BotService) executeCodingPrompt(chatID int64, prompt string) {
	b.mu.Lock()
	if b.isExecuting {
		b.mu.Unlock()
		_, _ = b.sendMessage(chatID, "⚠️ Sila tunggu, terdapat satu tugasan lain yang sedang diproses. Gunakan /cancel jika ingin membatalkannya.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	b.cancelExecution = cancel
	b.isExecuting = true
	b.currentTaskMsg = prompt
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.isExecuting = false
		b.cancelExecution = nil
		b.mu.Unlock()
	}()

	// 1. Broadcast Task Start to WebUI (Live stream & Pixel Office Agent!)
	sourceName := fmt.Sprintf("Telegram (@%s)", strings.TrimPrefix(b.GetBotUsername(), "@"))
	web.BroadcastExecutionStart(b.cfg.Model, prompt, sourceName)

	// Notify user that task execution started
	initialMsg, _ := b.sendMessage(chatID, fmt.Sprintf("⏳ *Sedang melaksanakan tugas melalui agy...*\n\n_Model:_ `%s`\n_Laluan:_ `%s`\n\nArahan: _%s_", b.cfg.Model, b.cfg.CodingPath, prompt))

	// Send typing actions periodically
	typingCtx, cancelTyping := context.WithCancel(ctx)
	go b.sendTyping(typingCtx, chatID)

	start := time.Now()
	result, err := b.agyClient.RunPromptStreaming(ctx, prompt, b.cfg.Model, b.cfg.CodingPath, func(streamType, line string) {
		web.BroadcastExecutionLog(streamType, line)
	})
	cancelTyping()
	duration := time.Since(start).Round(time.Millisecond)

	// 2. Broadcast Task Finish to WebUI (Pixel Office agent returns to idle)
	web.BroadcastExecutionFinish(b.cfg.Model, duration, err == nil)

	if err != nil {
		if ctx.Err() == context.Canceled {
			return
		}
		errMsg := fmt.Sprintf("❌ *Ralat Pelaksanaan agy:*\n%s", err.Error())
		if result != "" {
			errMsg += fmt.Sprintf("\n\n*Output:*\n%s", result)
		}
		b.sendLongMessage(chatID, errMsg)
		return
	}

	// Delete initial waiting message or keep it clean
	if initialMsg.MessageID != 0 {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(chatID, initialMsg.MessageID))
	}

	header := fmt.Sprintf("✅ *Selesai (%s)*\n━━━━━━━━━━━━━━━━━━━━━━━━━\n\n", duration.Round(time.Second))
	b.sendLongMessage(chatID, header+result)
}

// handleCallbackQuery handles inline button presses.
func (b *BotService) handleCallbackQuery(cq *tgbotapi.CallbackQuery) {
	chatID := cq.Message.Chat.ID
	if !b.isAuthorized(chatID) {
		resp := tgbotapi.NewCallback(cq.ID, "Akses ditolak")
		_, _ = b.api.Request(resp)
		return
	}

	data := cq.Data
	switch {
	case data == "menu:main":
		text := fmt.Sprintf(
			"👋 *Menu Papan Pemuka miqa*\n\n"+
				"🔹 *Model Aktif:* `%s`\n"+
				"📁 *Laluan Kod:* `%s`\n\n"+
				"Pilih menu di bawah:",
			b.cfg.Model,
			b.cfg.CodingPath,
		)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(MainDashboardKeyboard())
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))

	case data == "menu:model":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, _ := b.agyClient.ListModels(ctx)
		text := fmt.Sprintf("🤖 *PILIH MODEL AI*\n\nModel semasa: `%s`\n\nSila pilih model pilihan anda:", b.cfg.Model)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(ModelSelectionKeyboard(models, b.cfg.Model))
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))

	case strings.HasPrefix(data, "model:set:"):
		newModel := strings.TrimPrefix(data, "model:set:")
		b.cfg.Model = newModel
		_ = b.cfg.SaveConfig()
		log.Printf("[miqa-bot] Model AI ditukar kepada: %s", newModel)

		// Refresh model keyboard to show new active model
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, _ := b.agyClient.ListModels(ctx)
		text := fmt.Sprintf("🤖 *PILIH MODEL AI*\n\n✅ *Model berjaya ditukar kepada:* `%s`\n\nPilih model lain jika perlu:", newModel)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(ModelSelectionKeyboard(models, b.cfg.Model))
		_, _ = b.api.Request(edit)

		resp := tgbotapi.NewCallback(cq.ID, "Model ditukar kepada "+newModel)
		_, _ = b.api.Request(resp)

	case data == "menu:usage":
		stats, _ := agy.GetUsageStats(b.cfg.CodingPath)
		text := stats.FormatTelegramMarkdown(b.cfg.Model, b.cfg.CodingPath)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(BackToMenuKeyboard())
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))

	case data == "menu:path":
		b.SetAwaitingPath(chatID, true)
		text := fmt.Sprintf(
			"📁 *TETAPKAN LALUAN PENGEKODAN*\n\n"+
				"Laluan semasa:\n`%s`\n\n"+
				"💡 _Pilih direktori cadangan di bawah atau hantar laluan folder baharu dalam chat:_\n"+
				"Contoh: `/path /home/maui/github/payvoucher`",
			b.cfg.CodingPath,
		)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(PathSelectionKeyboard(b.cfg.CodingPath))
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))

	case strings.HasPrefix(data, "path:set:") || strings.HasPrefix(data, "path:sub:"):
		var targetPath string
		if strings.HasPrefix(data, "path:set:") {
			targetPath = strings.TrimPrefix(data, "path:set:")
		} else {
			folderName := strings.TrimPrefix(data, "path:sub:")
			parent := filepath.Dir(b.cfg.CodingPath)
			targetPath = filepath.Join(parent, folderName)
		}
		validPath, err := ValidatePath(targetPath)
		if err != nil {
			resp := tgbotapi.NewCallback(cq.ID, "Ralat: "+err.Error())
			_, _ = b.api.Request(resp)
			return
		}
		b.cfg.CodingPath = validPath
		_ = b.cfg.SaveConfig()
		b.SetAwaitingPath(chatID, false)
		log.Printf("[miqa-bot] Laluan pengekodan ditukar via butang inline: %s", validPath)

		text := fmt.Sprintf(
			"✅ *Laluan Berjaya Dikemaskini!*\n\n"+
				"📁 *Laluan Kod Baharu:*\n`%s`\n\n"+
				"Pilih menu di bawah:",
			validPath,
		)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(MainDashboardKeyboard())
		_, _ = b.api.Request(edit)
		resp := tgbotapi.NewCallback(cq.ID, "Laluan ditukar ke "+filepath.Base(validPath))
		_, _ = b.api.Request(resp)

	case data == "menu:status":
		uptime := time.Since(b.startTime).Round(time.Second)
		text := fmt.Sprintf(
			"⚡ *STATUS SISTEM MIQA*\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"🤖 *Bot Telegram:* `%s` (Aktif)\n"+
				"🧠 *Model Pilihan:* `%s`\n"+
				"📁 *Laluan Kod:* `%s`\n"+
				"🛡️ *Mod Pintasan:* `--dangerously-skip-permissions` (Aktif)\n"+
				"⏱️ *Masa Aktif Bot:* %s\n"+
				"🟢 *Status Enjin:* agy bersedia",
			b.GetBotUsername(),
			b.cfg.Model,
			b.cfg.CodingPath,
			uptime.String(),
		)
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(BackToMenuKeyboard())
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))
	case data == "menu:web":
		text := "🌐 *MIQA WEBUI DASHBOARD*\n" +
			"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
			"Saksikan *Animated Pixel Office*, graf telemetri sistem secara langsung, dan konsol terminal agy masa nyata:\n\n" +
			"🔗 [Buka WebUI MIQA](http://localhost:8080)\n`http://localhost:8080`\n\n" +
			"_(Pastikan pelayan web dimulakan dengan arahan `miqa web` atau melalui menu CLI)_"
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(BackToMenuKeyboard())
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))

	case data == "menu:help":
		text := "📖 *PANDUAN PENGGUNAAN MIQA*\n" +
			"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
			"🔹 */menu* - Papan pemuka interaktif\n" +
			"🔹 */model* - Pilih model AI\n" +
			"🔹 */usage* - Semak penggunaan\n" +
			"🔹 */path <folder>* - Tukar direktori kerja\n" +
			"🔹 */status* - Status sistem & agy\n" +
			"🔹 */cancel* - Batal tugas agy\n" +
			"🔹 */help* - Bantuan arahan\n\n" +
			"💬 Hantar teks terus untuk menulis kod secara automatik!"
		edit := tgbotapi.NewEditMessageText(chatID, cq.Message.MessageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = ptrMarkup(BackToMenuKeyboard())
		_, _ = b.api.Request(edit)
		_, _ = b.api.Request(tgbotapi.NewCallback(cq.ID, ""))
	}
}

func ptrMarkup(markup tgbotapi.InlineKeyboardMarkup) *tgbotapi.InlineKeyboardMarkup {
	return &markup
}
