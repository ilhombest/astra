package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ── log ring buffer ──────────────────────────────────────────────

type LogEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

var (
	logBuf   []LogEntry
	logMu    sync.RWMutex
	maxLines = 1000
)

func addLog(level, msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	logBuf = append(logBuf, LogEntry{
		Time:  time.Now().Format("2006-01-02 15:04:05"),
		Level: level,
		Msg:   strings.TrimSpace(msg),
	})
	if len(logBuf) > maxLines {
		logBuf = logBuf[len(logBuf)-maxLines:]
	}
}

// logWriter wraps io.Writer and copies to ring buffer
type logWriter struct{ w io.Writer }

func (l logWriter) Write(p []byte) (int, error) {
	line := strings.TrimSpace(string(p))
	if line != "" {
		level := "info"
		low := strings.ToLower(line)
		if strings.Contains(low, "error") || strings.Contains(low, "fatal") {
			level = "error"
		} else if strings.Contains(low, "warn") {
			level = "warning"
		}
		addLog(level, line)
	}
	return l.w.Write(p)
}

func initLogger() {
	lw := logWriter{w: os.Stderr}
	log.SetOutput(lw)
}

// captureProcess pipes cmd stdout/stderr into ring buffer
func captureProcess(cmd *exec.Cmd) {
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	go func() {
		defer pr.Close()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			level := "info"
			low := strings.ToLower(line)
			if strings.Contains(low, "error") || strings.Contains(low, "fatal") {
				level = "error"
			} else if strings.Contains(low, "warn") {
				level = "warning"
			}
			addLog(level, line)
		}
	}()
}

// ── process management ────────────────────────────────────────────

func restartAstra() error {
	pid, err := readPID()
	if err != nil {
		return startAstra()
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return startAstra()
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return startAstra()
	}
	addLog("info", fmt.Sprintf("sending SIGHUP to astra pid=%d", pid))
	return proc.Signal(syscall.SIGHUP)
}

func startAstra() error {
	if err := os.MkdirAll(dirOf(*flagPidFile), 0755); err != nil {
		return fmt.Errorf("pid dir: %w", err)
	}
	cmd := exec.Command(
		*flagAstraBin,
		"--stream",
		*flagConfig,
		"--pid", *flagPidFile,
	)
	captureProcess(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start astra: %w", err)
	}
	addLog("info", fmt.Sprintf("astra started pid=%d", cmd.Process.Pid))
	go func() {
		if err := cmd.Wait(); err != nil {
			addLog("error", fmt.Sprintf("astra exited: %v", err))
		}
	}()
	return nil
}

func stopAstra() error {
	pid, err := readPID()
	if err != nil {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	addLog("info", fmt.Sprintf("sending SIGTERM to astra pid=%d", pid))
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
	}
	_ = os.Remove(*flagPidFile)
	return nil
}

func getSessions() (map[string]any, error) {
	pid, err := readPID()
	if err != nil {
		return map[string]any{"sessions": []any{}, "status": "offline"}, nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		return map[string]any{"sessions": []any{}, "status": "offline"}, nil
	}
	return map[string]any{"sessions": []any{}, "status": "online"}, nil
}

func readPID() (int, error) {
	data, err := os.ReadFile(*flagPidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

// ── /api/log endpoint ──────────────────────────────────────────────

func handleLog(w http.ResponseWriter, r *http.Request) {
	logMu.RLock()
	entries := make([]LogEntry, len(logBuf))
	copy(entries, logBuf)
	logMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"lines": entries,
	})
}
