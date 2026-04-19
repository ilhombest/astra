package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// --- background CPU sampler ---

type cpuStat struct {
	total, idle uint64
}

var (
	cpuMu     sync.RWMutex
	lastCPU   cpuStat
	sysCPUPct float64
	startTime = time.Now()
)

func init() {
	lastCPU, _ = readCPUStat()
	go func() {
		for range time.Tick(5 * time.Second) {
			cur, err := readCPUStat()
			if err != nil {
				continue
			}
			cpuMu.Lock()
			dt := cur.total - lastCPU.total
			di := cur.idle - lastCPU.idle
			if dt > 0 {
				sysCPUPct = float64(dt-di) / float64(dt) * 100
			}
			lastCPU = cur
			cpuMu.Unlock()
		}
	}()
}

func readCPUStat() (cpuStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)[1:] // skip "cpu"
	var vals [8]uint64
	for i := 0; i < len(vals) && i < len(fields); i++ {
		vals[i], _ = strconv.ParseUint(fields[i], 10, 64)
	}
	total := vals[0] + vals[1] + vals[2] + vals[3] + vals[4] + vals[5] + vals[6] + vals[7]
	idle := vals[3] + vals[4]
	return cpuStat{total, idle}, nil
}

// --- /proc readers ---

func readLoadAvg() (la1, la5, la15 int) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	var f1, f5, f15 float64
	fmt.Sscanf(string(data), "%f %f %f", &f1, &f5, &f15)
	// frontend divides by 100 to show float
	return int(f1 * 100), int(f5 * 100), int(f15 * 100)
}

func readMemInfo() (appMemKB int, sysMemPct float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}
	vals := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			v, _ := strconv.ParseUint(fields[1], 10, 64)
			vals[key] = v
		}
	}
	total := vals["MemTotal"]
	avail := vals["MemAvailable"]
	if total > 0 {
		used := total - avail
		sysMemPct = float64(used) / float64(total) * 100
	}
	// app mem: read our own RSS from /proc/self/status
	selfData, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range strings.Split(string(selfData), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					v, _ := strconv.ParseUint(fields[1], 10, 64)
					appMemKB = int(v)
				}
			}
		}
	}
	return
}

func astraProcMemKB() int {
	pid, err := readPID()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseUint(fields[1], 10, 64)
				return int(v)
			}
		}
	}
	return 0
}

func astraAppCPU() float64 {
	pid, err := readPID()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	// field 14 = utime, 15 = stime (1-indexed, space-separated after name field)
	// format: pid (name) state ... utime stime
	s := string(data)
	// skip past the closing ')' of process name
	end := strings.LastIndex(s, ")")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(s[end+2:])
	if len(fields) < 13 {
		return 0
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	ticks := float64(utime+stime) / 100.0 // USER_HZ=100
	uptime := time.Since(startTime).Seconds()
	if uptime > 0 {
		return ticks / uptime * 100
	}
	return 0
}

// --- handlers ---

func handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	w.Header().Set("Content-Type", "application/json")

	switch {
	case path == "system-status":
		handleSystemStatus(w, r)
	case strings.HasPrefix(path, "stream-status/"):
		id := strings.TrimPrefix(path, "stream-status/")
		handleStreamStatus(w, r, id)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func handleSystemStatus(w http.ResponseWriter, _ *http.Request) {
	cpuMu.RLock()
	sysCPU := sysCPUPct
	cpuMu.RUnlock()

	la1, la5, la15 := readLoadAvg()
	_, sysMemPct := readMemInfo()
	appMemKB := astraProcMemKB()
	appCPU := astraAppCPU()

	pid, _ := readPID()
	uptimeMin := 0
	if pid > 0 {
		if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
			// field 22 = starttime in clock ticks since system boot
			s := string(data)
			end := strings.LastIndex(s, ")")
			if end >= 0 {
				fields := strings.Fields(s[end+2:])
				if len(fields) >= 20 {
					startTicks, _ := strconv.ParseUint(fields[19], 10, 64)
					bootData, _ := os.ReadFile("/proc/uptime")
					var bootSec float64
					fmt.Sscanf(string(bootData), "%f", &bootSec)
					procStartSec := float64(startTicks) / 100.0
					runSec := bootSec - procStartSec
					if runSec > 0 {
						uptimeMin = int(runSec / 60)
					}
				}
			}
		}
	}

	// check astra online
	isOnline := false
	if pid > 0 {
		if proc, err := os.FindProcess(pid); err == nil {
			isOnline = proc.Signal(syscall.Signal(0)) == nil
		}
	}
	if !isOnline {
		appCPU = 0
		appMemKB = 0
		uptimeMin = 0
	}

	json.NewEncoder(w).Encode(map[string]any{
		"app_cpu_usage": fmt.Sprintf("%.1f", appCPU),
		"sys_cpu_usage": fmt.Sprintf("%.1f", sysCPU),
		"la1":           la1,
		"la5":           la5,
		"la15":          la15,
		"app_mem_kb":    appMemKB,
		"sys_mem_usage": fmt.Sprintf("%.1f", sysMemPct),
		"app_uptime":    uptimeMin,
	})
}

func handleStreamStatus(w http.ResponseWriter, _ *http.Request, id string) {
	cfg, err := loadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	streams, _ := cfg["streams"].(map[string]any)
	stream, ok := streams[id]
	if !ok {
		http.Error(w, "stream not found", http.StatusNotFound)
		return
	}
	s, _ := stream.(map[string]any)
	inputs, _ := s["input"].([]any)
	inputStatus := map[string]any{
		"bitrate":   0,
		"cc_error":  0,
		"pes_error": 0,
	}
	if len(inputs) > 0 {
		inputStatus["url"] = inputs[0]
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id":        id,
		"bitrate":   0,
		"cc_error":  0,
		"pes_error": 0,
		"input":     inputStatus,
		"output":    []any{},
	})
}
