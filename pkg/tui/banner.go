package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#00D7FF") // Cyan
	secondaryColor = lipgloss.Color("#BD93F9") // Purple
	successColor   = lipgloss.Color("#50FA7B") // Green
	warningColor   = lipgloss.Color("#FFB86C") // Orange
	errorColor     = lipgloss.Color("#FF5555") // Red
	mutedColor     = lipgloss.Color("#6272A4") // Gray

	// Text Styles
	stylePrimary   = lipgloss.NewStyle().Foreground(primaryColor)
	styleSecondary = lipgloss.NewStyle().Foreground(secondaryColor)
	styleSuccess   = lipgloss.NewStyle().Foreground(successColor)
	styleWarning   = lipgloss.NewStyle().Foreground(warningColor)
	styleError     = lipgloss.NewStyle().Foreground(errorColor)
	styleMuted     = lipgloss.NewStyle().Foreground(mutedColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(0)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#FF5555")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	successBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	warningBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(warningColor)
)

// PrintBanner renders the miqa ASCII header and safety warnings.
func PrintBanner() {
	banner := `
███╗   ███╗██╗ ██████╗  █████╗ 
████╗ ████║██║██╔═══██╗██╔══██╗
██╔████╔██║██║██║   ██║███████║
██║╚██╔╝██║██║██║▄▄ ██║██╔══██║
██║ ╚═╝ ██║██║╚██████╔╝██║  ██║
╚═╝     ╚═╝╚═╝ ╚══▀▀═╝ ╚═╝  ╚═╝`

	fmt.Println(titleStyle.Render(banner))
	fmt.Println(subtitleStyle.Render("  Intelligent AI Coding Agent CLI & Telegram Bridge"))
	fmt.Println()

	// Safety warning regarding --dangerously-skip-permissions
	securityNote := warningBadge.Render("⚠️  KESELAMATAN:") + " " +
		styleMuted.Render("Mod ") +
		lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render("--dangerously-skip-permissions") +
		styleMuted.Render(" aktif. Semua tindakan ejen akan diluluskan secara automatik.")
	fmt.Println(securityNote)
	fmt.Println()
}

// PrintStatusBox renders a key-value status box.
func PrintStatusBox(title string, items map[string]string) {
	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(title) + "\n\n"
	for k, v := range items {
		content += lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render(k+": ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(v) + "\n"
	}
	fmt.Println(cardStyle.Render(content))
}
