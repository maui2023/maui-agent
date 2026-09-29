package agy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ModelInfo represents an AI model available in agy.
type ModelInfo struct {
	ID   string
	Name string
}

// Client wraps the Antigravity CLI (agy).
type Client struct {
	BinPath string
}

// NewClient locates the agy binary and returns an initialized Client.
func NewClient() (*Client, error) {
	// Search candidates
	candidates := []string{
		"agy",
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "agy"),
			filepath.Join(home, "bin", "agy"),
		)
	}
	candidates = append(candidates, "/usr/local/bin/agy", "/usr/bin/agy")

	for _, cand := range candidates {
		if path, err := exec.LookPath(cand); err == nil {
			return &Client{BinPath: path}, nil
		}
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
			return &Client{BinPath: cand}, nil
		}
	}

	return nil, errors.New("perintah 'agy' tidak dijumpai dalam PATH atau ~/.local/bin/agy. Sila pastikan Antigravity CLI telah dipasang")
}

// VerifyStartup checks that agy is executable and supports --dangerously-skip-permissions.
func (c *Client) VerifyStartup(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, c.BinPath, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gagal menjalankan agy: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	if !strings.Contains(string(out), "--dangerously-skip-permissions") {
		return errors.New("versi agy ini tidak menyokong bendera --dangerously-skip-permissions")
	}

	return nil
}

// CheckAuth verifies if the user is authenticated in agy.
func (c *Client) CheckAuth(ctx context.Context) (bool, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, c.BinPath, "models")
	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if err != nil {
		if strings.Contains(strings.ToLower(outputStr), "login") ||
			strings.Contains(strings.ToLower(outputStr), "auth") ||
			strings.Contains(strings.ToLower(outputStr), "unauthorized") {
			return false, nil
		}
		return false, fmt.Errorf("ralat semasa menyemak status log masuk: %w\n%s", err, outputStr)
	}

	// If models output contains valid model lines, authentication succeeded
	lines := strings.Split(outputStr, "\n")
	hasModels := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "gemini-") || strings.HasPrefix(trimmed, "claude-") || strings.HasPrefix(trimmed, "gpt-") {
			hasModels = true
			break
		}
	}

	return hasModels, nil
}

// ListModels retrieves the list of available models from agy.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, c.BinPath, "models")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan senarai model: %w\n%s", err, string(out))
	}

	var models []ModelInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Fetching") || strings.HasPrefix(line, "⠋") || strings.HasPrefix(line, "⠙") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			id := parts[0]
			name := strings.TrimSpace(strings.TrimPrefix(line, id))
			models = append(models, ModelInfo{
				ID:   id,
				Name: name,
			})
		} else if len(parts) == 1 {
			models = append(models, ModelInfo{
				ID:   parts[0],
				Name: parts[0],
			})
		}
	}

	if len(models) == 0 {
		// Provide default fallback models in case parsing missed something
		models = []ModelInfo{
			{ID: "gemini-3.8-flash-high", Name: "Gemini 3.8 Flash (High)"},
			{ID: "gemini-3.8-flash-medium", Name: "Gemini 3.8 Flash (Medium)"},
			{ID: "gemini-3.7-flash-high", Name: "Gemini 3.7 Flash (High)"},
			{ID: "gemini-3.1-pro-high", Name: "Gemini 3.1 Pro (High)"},
			{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (Thinking)"},
			{ID: "claude-opus-4-6-thinking", Name: "Claude Opus 4.6 (Thinking)"},
		}
	}

	return models, nil
}

// OutputCallback is invoked whenever a line of stdout or stderr is received from agy.
type OutputCallback func(streamType string, line string)

// RunPrompt executes a single prompt using agy with --dangerously-skip-permissions.
func (c *Client) RunPrompt(ctx context.Context, prompt string, model string, workspaceDir string) (string, error) {
	return c.RunPromptStreaming(ctx, prompt, model, workspaceDir, nil)
}

// RunPromptStreaming executes an agy prompt, streaming output line-by-line via callback in real time.
func (c *Client) RunPromptStreaming(ctx context.Context, prompt string, model string, workspaceDir string, cb OutputCallback) (string, error) {
	if workspaceDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			workspaceDir = cwd
		}
	}

	args := []string{
		"--dangerously-skip-permissions",
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	if workspaceDir != "" {
		args = append(args, "--add-dir", workspaceDir)
	}

	args = append(args, "-p", prompt)

	cmd := exec.CommandContext(ctx, c.BinPath, args...)
	cmd.Dir = workspaceDir

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("gagal membuka stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("gagal membuka stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("gagal memulakan proses agy: %w", err)
	}

	var fullOutput strings.Builder
	var fullErr strings.Builder
	var wg sync.WaitGroup
	var outMu sync.Mutex

	// Read stdout line-by-line
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			outMu.Lock()
			fullOutput.WriteString(line)
			fullOutput.WriteString("\n")
			outMu.Unlock()
			if cb != nil {
				cb("stdout", line)
			}
		}
	}()

	// Read stderr line-by-line
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			outMu.Lock()
			fullErr.WriteString(line)
			fullErr.WriteString("\n")
			outMu.Unlock()
			if cb != nil {
				cb("stderr", line)
			}
		}
	}()

	wg.Wait()
	waitErr := cmd.Wait()

	stdoutStr := strings.TrimSpace(fullOutput.String())
	stderrStr := strings.TrimSpace(fullErr.String())

	if waitErr != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if stdoutStr != "" {
			return stdoutStr, fmt.Errorf("ralat pelaksanaan: %w (stderr: %s)", waitErr, stderrStr)
		}
		return "", fmt.Errorf("ralat agy: %w\n%s", waitErr, stderrStr)
	}

	if stdoutStr != "" {
		return stdoutStr, nil
	}
	return stderrStr, nil
}
