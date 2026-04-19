package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// restartAstra sends SIGHUP to the running astra process so it reloads its config.
// If no process is running, it starts one.
func restartAstra() error {
	pid, err := readPID()
	if err != nil {
		// no running process — start fresh
		return startAstra()
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return startAstra()
	}
	// check process is actually alive
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return startAstra()
	}
	log.Printf("sending SIGHUP to astra pid=%d", pid)
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start astra: %w", err)
	}
	log.Printf("astra started pid=%d", cmd.Process.Pid)
	// detach — let astra manage its own lifecycle
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("astra exited: %v", err)
		}
	}()
	return nil
}

func stopAstra() error {
	pid, err := readPID()
	if err != nil {
		return nil // already stopped
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	log.Printf("sending SIGTERM to astra pid=%d", pid)
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	// wait up to 5 s for clean exit
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break // process gone
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
