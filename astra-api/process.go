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

// ── process tracking ─────────────────────────────────────────────

var (
	astraProc   *os.Process
	astraProcMu sync.Mutex
	astraStart  time.Time
)

func setAstraProc(p *os.Process) {
	astraProcMu.Lock()
	astraProc = p
	astraStart = time.Now()
	astraProcMu.Unlock()
}

func isAstraAlive() bool {
	astraProcMu.Lock()
	p := astraProc
	astraProcMu.Unlock()
	if p != nil {
		if p.Signal(syscall.Signal(0)) == nil {
			return true
		}
	}
	// fallback: pidfile
	pid, err := readPID()
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func astraUptimeMin() int {
	astraProcMu.Lock()
	p := astraProc
	t := astraStart
	astraProcMu.Unlock()
	if p == nil || !t.IsZero() == false {
		return 0
	}
	if p.Signal(syscall.Signal(0)) != nil {
		return 0
	}
	return int(time.Since(t).Minutes())
}

// ── process management ────────────────────────────────────────────

// attachOrStartAstra is called once on startup.
// If astra is already running (pidfile + process alive), restart it through
// astra-api so its output is captured in the log ring buffer.
// If not running, start it fresh.
func attachOrStartAstra() {
	time.Sleep(500 * time.Millisecond) // let HTTP server start first
	pid, err := readPID()
	if err != nil {
		addLog("info", "astra not running, starting…")
		if err := startAstra(); err != nil {
			addLog("error", "failed to start astra: "+err.Error())
		}
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		addLog("info", "stale pidfile, starting astra…")
		_ = os.Remove(*flagPidFile)
		if err := startAstra(); err != nil {
			addLog("error", "failed to start astra: "+err.Error())
		}
		return
	}
	// astra is running but not captured — restart to capture logs
	addLog("info", fmt.Sprintf("astra running (pid=%d), restarting to capture logs…", pid))
	if err := stopAstra(); err != nil {
		addLog("error", "stop failed: "+err.Error())
	}
	time.Sleep(800 * time.Millisecond)
	if err := startAstra(); err != nil {
		addLog("error", "failed to restart astra: "+err.Error())
	}
}

// restartAstra does a full stop+start so the Lua config is fully reloaded.
// SIGHUP in this astra version does not re-read the Lua script.
func restartAstra() error {
	if err := stopAstra(); err != nil {
		addLog("error", "stop failed: "+err.Error())
	}
	time.Sleep(600 * time.Millisecond)
	return startAstra()
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
	setAstraProc(cmd.Process)
	addLog("info", fmt.Sprintf("astra started pid=%d", cmd.Process.Pid))
	go func() {
		if err := cmd.Wait(); err != nil {
			addLog("error", fmt.Sprintf("astra exited: %v", err))
		}
		astraProcMu.Lock()
		if astraProc == cmd.Process {
			astraProc = nil
		}
		astraProcMu.Unlock()
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
