package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/maui2023/miqa/pkg/agy"
	"github.com/maui2023/miqa/pkg/config"
	"github.com/maui2023/miqa/pkg/telegram"
	"github.com/maui2023/miqa/pkg/tui"
	"github.com/maui2023/miqa/pkg/web"
	"github.com/spf13/cobra"
)

var (
	flagModel     string
	flagPath      string
	flagToken     string
	flagChatID    int64
	flagSkipPerms bool
)

// Execute runs the root miqa command.
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "miqa",
		Short: "miqa - AI Coding Agent CLI & Telegram Bridge",
		Long: lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7FF")).Render(`
███╗   ███╗██╗ ██████╗  █████╗ 
████╗ ████║██║██╔═══██╗██╔══██╗
██╔████╔██║██║██║   ██║███████║
██║╚██╔╝██║██║██║▄▄ ██║██╔══██║
██║ ╚═╝ ██║██║╚██████╔╝██║  ██║
╚═╝     ╚═╝╚═╝ ╚══▀▀═╝ ╚═╝  ╚═╝`) + "\n\n" +
			"miqa ialah sistem ejen CLI pintar berasaskan Go untuk sesi pengekodan,\n" +
			"konfigurasi bot Telegram, pemilihan model AI, dan semakan penggunaan.\n" +
			"Dilengkapi dengan mod pintasan kebenaran lanjutan (--dangerously-skip-permissions).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(cmd.Context())
		},
	}

	rootCmd.PersistentFlags().StringVarP(&flagModel, "model", "m", "", "Model AI untuk digunakan (cth: gemini-3.8-flash-high)")
	rootCmd.PersistentFlags().StringVarP(&flagPath, "path", "p", "", "Laluan direktori kerja pengekodan")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Token Bot Telegram")
	rootCmd.PersistentFlags().Int64Var(&flagChatID, "chat-id", 0, "Chat ID Telegram yang dibenarkan")
	rootCmd.PersistentFlags().BoolVar(&flagSkipPerms, "dangerously-skip-permissions", true, "Luluskan semua kebenaran alat agy secara automatik")

	// Subcommand: bot
	botCmd := &cobra.Command{
		Use:   "bot",
		Short: "Mulakan perkhidmatan Bot Telegram secara terus",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBotOnly(cmd.Context())
		},
	}

	// Subcommand: models
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "Papar senarai model AI yang tersedia dalam agy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsList(cmd.Context())
		},
	}

	// Subcommand: usage
	usageCmd := &cobra.Command{
		Use:   "usage",
		Short: "Semak statistik penggunaan sesi dan sumber sistem",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsageOnly(cmd.Context())
		},
	}

	// Subcommand: run
	promptCmd := &cobra.Command{
		Use:   "run [prompt]",
		Short: "Laksanakan arahan pengekodan agy secara terus dalam terminal",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromptDirect(cmd.Context(), args[0])
		},
	}

	// Subcommand: web
	var flagWebPort int
	var flagWebHost string
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Mulakan pelayan WebUI (Animated Pixel Office & System Graphs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebOnly(cmd.Context(), flagWebHost, flagWebPort)
		},
	}
	webCmd.Flags().IntVarP(&flagWebPort, "port", "P", 8080, "Port untuk pelayan WebUI")
	webCmd.Flags().StringVar(&flagWebHost, "host", "0.0.0.0", "Hos alamat untuk pelayan WebUI")

	rootCmd.AddCommand(botCmd, modelsCmd, usageCmd, promptCmd, webCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runWorkflow executes the full PRD user flow.
func runWorkflow(ctx context.Context) error {
	tui.PrintBanner()

	// 1. Initialize & verify agy startup command with --dangerously-skip-permissions
	fmt.Print("⏳ Mengesahkan enjin agy (--dangerously-skip-permissions)... ")
	agyClient, err := agy.NewClient()
	if err != nil {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("GAGAL"))
		return err
	}

	if err := agyClient.VerifyStartup(ctx); err != nil {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("RALAT"))
		return err
	}
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("BERJAYA"))

	// 2. Auth Verification (Check Login Status)
	fmt.Print("⏳ Menyemak status log masuk sesi Antigravity... ")
	isLoggedIn, err := agyClient.CheckAuth(ctx)
	if err != nil || !isLoggedIn {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("BELUM LOG MASUK"))
		tui.ShowLoginNotice()
		return nil
	}
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render("DISAHKAN (SUDAH LOG MASUK)"))

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("ralat konfigurasi: %w", err)
	}

	// Apply CLI flags if supplied
	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagPath != "" {
		cfg.CodingPath = flagPath
	}
	if flagToken != "" {
		cfg.TelegramToken = flagToken
	}
	if flagChatID != 0 {
		cfg.TelegramChatID = flagChatID
	}

	// 3. Telegram Bot Configuration prompt (if not already set)
	if !cfg.IsTelegramConfigured() {
		fmt.Println()
		if err := tui.PromptTelegramConfig(cfg); err != nil {
			return err
		}
	}

	// 4. Interactive Dashboard
	return tui.RunDashboard(ctx, agyClient, cfg)
}

func runBotOnly(ctx context.Context) error {
	tui.PrintBanner()

	agyClient, err := agy.NewClient()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagPath != "" {
		cfg.CodingPath = flagPath
	}
	if flagToken != "" {
		cfg.TelegramToken = flagToken
	}
	if flagChatID != 0 {
		cfg.TelegramChatID = flagChatID
	}

	if !cfg.IsTelegramConfigured() {
		if err := tui.PromptTelegramConfig(cfg); err != nil {
			return err
		}
	}

	botService, err := telegram.NewBotService(cfg, agyClient)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Bot aktif sebagai %s\n", botService.GetBotUsername())
	fmt.Println("Menunggu arahan pengekodan daripada Telegram... (Tekan Ctrl+C untuk berhenti)")
	fmt.Println()
	return botService.Start(ctx)
}

func runModelsList(ctx context.Context) error {
	agyClient, err := agy.NewClient()
	if err != nil {
		return err
	}

	models, err := agyClient.ListModels(ctx)
	if err != nil {
		return err
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D7FF")).Render("🤖 MODEL AI TERSEDIA DALAM AGY:"))
	for i, m := range models {
		fmt.Printf("  %2d. %-26s %s\n", i+1, lipgloss.NewStyle().Bold(true).Render(m.ID), m.Name)
	}
	return nil
}

func runUsageOnly(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	tui.DisplayUsage(cfg)
	return nil
}

func runPromptDirect(ctx context.Context, prompt string) error {
	agyClient, err := agy.NewClient()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagPath != "" {
		cfg.CodingPath = flagPath
	}

	fmt.Printf("⏳ Menjalankan arahan dalam '%s'...\n", cfg.CodingPath)
	web.BroadcastExecutionStart(cfg.Model, prompt, "CLI (Terminal)")
	start := time.Now()
	_, err = agyClient.RunPromptStreaming(ctx, prompt, cfg.Model, cfg.CodingPath, func(streamType, line string) {
		web.BroadcastExecutionLog(streamType, line)
		fmt.Println(line)
	})
	dur := time.Since(start)
	web.BroadcastExecutionFinish(cfg.Model, dur, err == nil)
	if err != nil {
		return err
	}
	return nil
}

func runWebOnly(ctx context.Context, host string, port int) error {
	tui.PrintBanner()

	agyClient, err := agy.NewClient()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagPath != "" {
		cfg.CodingPath = flagPath
	}

	srv := web.NewServer(host, port, cfg, agyClient)
	fmt.Printf("\n🚀 Memulakan WebUI MIQA pada: %s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D7FF")).Render(fmt.Sprintf("http://localhost:%d", port)))
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Animated Pixel Office, Graf Sistem & Terminal Langsung aktif."))
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Tekan Ctrl+C untuk berhenti."))
	fmt.Println()

	return srv.Start(ctx)
}
