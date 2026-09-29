package agy

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

// SessionSummary represents a recent agy conversation session.
type SessionSummary struct {
	Title        string
	StepCount    int
	LastModified string
}

// UsageStats holds aggregated usage metrics from both agy and the local system.
type UsageStats struct {
	TotalSessions  int
	TotalSteps     int
	RecentSessions []SessionSummary

	// System metrics
	MemTotalGB float64
	MemUsedGB  float64
	MemFreeGB  float64
	DiskFreeGB float64
	DiskTotGB  float64
	CPULoad1m  string
}

// GetUsageStats queries SQLite DB and host system metrics.
func GetUsageStats(codingPath string) (*UsageStats, error) {
	stats := &UsageStats{}

	// 1. Check Antigravity DB
	home, err := os.UserHomeDir()
	if err == nil {
		dbPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
		if _, err := os.Stat(dbPath); err == nil {
			_ = readSQLiteStats(dbPath, stats)
		}
	}

	// 2. Read System Memory from /proc/meminfo
	readMemInfo(stats)

	// 3. Read CPU Load from /proc/loadavg
	readCPULoad(stats)

	// 4. Read Disk Space at codingPath
	readDiskStats(codingPath, stats)

	return stats, nil
}

func readSQLiteStats(dbPath string, stats *UsageStats) error {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	// Query counts
	row := db.QueryRow("SELECT COUNT(*), COALESCE(SUM(step_count), 0) FROM conversation_summaries")
	_ = row.Scan(&stats.TotalSessions, &stats.TotalSteps)

	// Query recent sessions
	rows, err := db.Query("SELECT title, step_count, last_modified_time FROM conversation_summaries ORDER BY last_modified_time DESC LIMIT 5")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s SessionSummary
			var rawTime string
			if err := rows.Scan(&s.Title, &s.StepCount, &rawTime); err == nil {
				if t, err := time.Parse(time.RFC3339Nano, rawTime); err == nil {
					s.LastModified = t.Format("02 Jan 15:04")
				} else {
					s.LastModified = rawTime
				}
				stats.RecentSessions = append(stats.RecentSessions, s)
			}
		}
	}

	return nil
}

func readMemInfo(stats *UsageStats) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}

	var memTotalKB, memAvailKB float64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			switch fields[0] {
			case "MemTotal:":
				_, _ = fmt.Sscanf(fields[1], "%f", &memTotalKB)
			case "MemAvailable:":
				_, _ = fmt.Sscanf(fields[1], "%f", &memAvailKB)
			}
		}
	}

	if memTotalKB > 0 {
		stats.MemTotalGB = memTotalKB / (1024 * 1024)
		stats.MemFreeGB = memAvailKB / (1024 * 1024)
		stats.MemUsedGB = (memTotalKB - memAvailKB) / (1024 * 1024)
	}
}

func readCPULoad(stats *UsageStats) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		stats.CPULoad1m = "N/A"
		return
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 1 {
		stats.CPULoad1m = fields[0]
	}
}

func readDiskStats(path string, stats *UsageStats) {
	if path == "" {
		path = "."
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err == nil {
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bavail * uint64(stat.Bsize)
		stats.DiskTotGB = float64(totalBytes) / (1024 * 1024 * 1024)
		stats.DiskFreeGB = float64(freeBytes) / (1024 * 1024 * 1024)
	}
}

// FormatTelegramMarkdown returns a message formatted for Telegram.
func (s *UsageStats) FormatTelegramMarkdown(activeModel, activePath string) string {
	var sb strings.Builder
	sb.WriteString("📊 *STATISTIK PENGGUNAAN MIQA*\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString("🧠 *Model AI Aktif:* `" + activeModel + "`\n")
	sb.WriteString(fmt.Sprintf("📂 *Sesi Antigravity:* %d sesi direkodkan\n", s.TotalSessions))
	sb.WriteString(fmt.Sprintf("⚡ *Jumlah Langkah (Steps):* %d langkah\n\n", s.TotalSteps))

	if len(s.RecentSessions) > 0 {
		sb.WriteString("🕒 *Sesi Terkini:*\n")
		for i, sess := range s.RecentSessions {
			title := sess.Title
			if title == "" {
				title = "(Tiada Tajuk)"
			}
			if len(title) > 28 {
				title = title[:25] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %d. %s (%d langkah) - %s\n", i+1, title, sess.StepCount, sess.LastModified))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("💻 *Sumber Sistem Semasa:*\n")
	if s.MemTotalGB > 0 {
		sb.WriteString(fmt.Sprintf("  • RAM Digunakan: %.1f GB / %.1f GB\n", s.MemUsedGB, s.MemTotalGB))
	}
	if s.DiskTotGB > 0 {
		sb.WriteString(fmt.Sprintf("  • Storan Bebas: %.1f GB / %.1f GB\n", s.DiskFreeGB, s.DiskTotGB))
	}
	if s.CPULoad1m != "" {
		sb.WriteString(fmt.Sprintf("  • Beban CPU (1m): %s\n", s.CPULoad1m))
	}
	sb.WriteString("  • Laluan Aktif: `" + activePath + "`\n")

	return sb.String()
}
