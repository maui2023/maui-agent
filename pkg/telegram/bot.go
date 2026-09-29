package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
)

// BotService manages the Telegram bot session and communication with agy.
type BotService struct {
	api             *tgbotapi.BotAPI
	cfg             *config.Config
	agyClient       *agy.Client
	startTime       time.Time
	mu              sync.Mutex
	isExecuting     bool
	cancelExecution context.CancelFunc
	currentTaskMsg  string
	awaitingPath    map[int64]bool
}

// NewBotService initializes a Telegram bot service with the provided config and agy client.
func NewBotService(cfg *config.Config, agyClient *agy.Client) (*BotService, error) {
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("token Telegram Bot tidak ditetapkan")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("gagal menyambung ke Telegram Bot API: %w", err)
	}

	return &BotService{
		api:          bot,
		cfg:          cfg,
		agyClient:    agyClient,
		startTime:    time.Now(),
		awaitingPath: make(map[int64]bool),
	}, nil
}

// GetBotUsername returns the authenticated bot's username.
func (b *BotService) GetBotUsername() string {
	if b.api != nil && b.api.Self.UserName != "" {
		return "@" + b.api.Self.UserName
	}
	return "miqa_bot"
}

// Start begins the Telegram bot update polling loop.
func (b *BotService) Start(ctx context.Context) error {
	log.Printf("[miqa-bot] Berjaya berhubung sebagai @%s (ID: %d)", b.api.Self.UserName, b.api.Self.ID)

	// Register bot command list in Telegram UI
	commands := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "menu", Description: "Papar Menu Interaktif Utama"},
		tgbotapi.BotCommand{Command: "model", Description: "Pilih model AI"},
		tgbotapi.BotCommand{Command: "usage", Description: "Semak statistik penggunaan"},
		tgbotapi.BotCommand{Command: "path", Description: "Tetapkan laluan folder kod"},
		tgbotapi.BotCommand{Command: "status", Description: "Semak status sistem & agy"},
		tgbotapi.BotCommand{Command: "cancel", Description: "Hentikan tugas agy yang sedang berjalan"},
		tgbotapi.BotCommand{Command: "help", Description: "Panduan arahan"},
	)
	_, _ = b.api.Request(commands)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("[miqa-bot] Menutup sambungan Telegram bot...")
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}

			// Handle Callback Queries (Button clicks)
			if update.CallbackQuery != nil {
				go b.handleCallbackQuery(update.CallbackQuery)
				continue
			}

			// Handle Messages (both new and edited messages)
			if update.Message != nil {
				go b.handleIncomingMessage(update.Message)
			} else if update.EditedMessage != nil {
				go b.handleIncomingMessage(update.EditedMessage)
			}
		}
	}
}

// SetAwaitingPath records whether a chat is in the middle of specifying a new path.
func (b *BotService) SetAwaitingPath(chatID int64, awaiting bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if awaiting {
		b.awaitingPath[chatID] = true
	} else {
		delete(b.awaitingPath, chatID)
	}
}

// IsAwaitingPath checks whether a chat is currently expected to enter a directory path.
func (b *BotService) IsAwaitingPath(chatID int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.awaitingPath[chatID]
}

// isAuthorized checks if the incoming chat ID is allowed.
func (b *BotService) isAuthorized(chatID int64) bool {
	if b.cfg.TelegramChatID == 0 {
		// Auto-bind to first chat ID if not set
		b.cfg.TelegramChatID = chatID
		_ = b.cfg.SaveConfig()
		log.Printf("[miqa-bot] Telegram Chat ID didaftarkan secara automatik: %d", chatID)
		return true
	}
	return b.cfg.TelegramChatID == chatID
}

// sendMessage sends a standard message formatted with Markdown.
func (b *BotService) sendMessage(chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	return b.api.Send(msg)
}

// sendLongMessage splits and sends long responses safely within Telegram's 4096 character limit.
func (b *BotService) sendLongMessage(chatID int64, text string) {
	const maxLen = 4000
	if len(text) <= maxLen {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		if _, err := b.api.Send(msg); err != nil {
			// Fallback without Markdown if markdown syntax fails
			msg.ParseMode = ""
			_, _ = b.api.Send(msg)
		}
		return
	}

	// Split by lines or chunks
	lines := strings.Split(text, "\n")
	var currentChunk strings.Builder

	for _, line := range lines {
		if currentChunk.Len()+len(line)+1 > maxLen {
			m := tgbotapi.NewMessage(chatID, currentChunk.String())
			m.ParseMode = ""
			_, _ = b.api.Send(m)
			currentChunk.Reset()
		}
		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
	}

	if currentChunk.Len() > 0 {
		m := tgbotapi.NewMessage(chatID, currentChunk.String())
		m.ParseMode = ""
		_, _ = b.api.Send(m)
	}
}

// sendTyping continuously sends typing action while a task is running.
func (b *BotService) sendTyping(ctx context.Context, chatID int64) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, _ = b.api.Send(action)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = b.api.Send(action)
		}
	}
}

// ValidatePath checks if directory exists and is a directory.
func ValidatePath(path string) (string, error) {
	return config.ValidatePath(path)
}
