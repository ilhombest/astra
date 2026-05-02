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
	"unsafe"
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
	case path == "log":
		handleLog(w, r)
		return
	case path == "system-status":
		handleSystemStatus(w, r)
	case strings.HasPrefix(path, "stream-status/"):
		id := strings.TrimPrefix(path, "stream-status/")
		handleStreamStatus(w, r, id)
	case strings.HasPrefix(path, "adapter-status/"):
		parts := strings.SplitN(strings.TrimPrefix(path, "adapter-status/"), "/", 2)
		adapterN, deviceN := 0, 0
		if len(parts) >= 1 {
			adapterN, _ = strconv.Atoi(parts[0])
		}
		if len(parts) >= 2 {
			deviceN, _ = strconv.Atoi(parts[1])
		}
		handleAdapterStatus(w, r, adapterN, deviceN)
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

func handleAdapterStatus(w http.ResponseWriter, _ *http.Request, adapter, device int) {
	lock, signal, snr, ber := readDVBStatus(adapter, device)
	json.NewEncoder(w).Encode(map[string]any{
		"lock":    lock,
		"signal":  signal,
		"snr":     snr,
		"ber":     ber,
		"bitrate": 0,
	})
}

// readDVBStatus reads signal stats via DVB ioctl API from /dev/dvb/adapterN/frontendN.
// Ioctl numbers: _IOR('o', N, size) = (2<<30)|(size<<16)|('o'<<8)|N
const (
	feReadStatus         = 0x80046F45 // _IOR('o',69,uint32)  FE_READ_STATUS
	feReadBER            = 0x80046F46 // _IOR('o',70,uint32)  FE_READ_BER
	feReadSignalStrength = 0x80026F47 // _IOR('o',71,uint16)  FE_READ_SIGNAL_STRENGTH
	feReadSNR            = 0x80026F48 // _IOR('o',72,uint16)  FE_READ_SNR
)

func readDVBStatus(adapter, device int) (lock bool, signal, snr, ber int) {
	path := fmt.Sprintf("/dev/dvb/adapter%d/frontend%d", adapter, device)
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	defer f.Close()
	fd := f.Fd()

	var st uint32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, feReadStatus, uintptr(unsafe.Pointer(&st))); e == 0 {
		lock = (st & 0x10) != 0 // FE_HAS_LOCK
	}
	var sig uint16
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, feReadSignalStrength, uintptr(unsafe.Pointer(&sig))); e == 0 {
		signal = int(sig) * 100 / 65535
	}
	var snrVal uint16
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, feReadSNR, uintptr(unsafe.Pointer(&snrVal))); e == 0 {
		snr = int(snrVal) * 100 / 65535
	}
	var berVal uint32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, feReadBER, uintptr(unsafe.Pointer(&berVal))); e == 0 {
		ber = int(berVal)
	}
	return
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

	bitrate, ccErr, pesErr, onAir := 0, 0, 0, false
	if stat := getStatByID(id); stat != nil {
		bitrate = stat.Bitrate
		ccErr = stat.CCErr
		pesErr = stat.PESErr
		onAir = stat.OnAir
	}

	inputStatus := map[string]any{
		"bitrate":   bitrate,
		"cc_error":  ccErr,
		"pes_error": pesErr,
		"on_air":    onAir,
	}
	if len(inputs) > 0 {
		inputStatus["url"] = inputs[0]
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id":        id,
		"bitrate":   bitrate,
		"cc_error":  ccErr,
		"pes_error": pesErr,
		"on_air":    onAir,
		"input":     inputStatus,
		"output":    []any{},
	})
}

func handleStreamsStatus(w http.ResponseWriter, _ *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	streams, _ := cfg["streams"].(map[string]any)
	result := map[string]any{}
	for id, v := range streams {
		s, _ := v.(map[string]any)
		name, _ := s["name"].(string)
		if stat := getStatByName(name); stat != nil {
			result[id] = map[string]any{
				"on_air":    stat.OnAir,
				"bitrate":   stat.Bitrate,
				"cc_error":  stat.CCErr,
				"pes_error": stat.PESErr,
			}
		} else {
			// null on_air means no monitoring data yet
			result[id] = map[string]any{"on_air": nil}
		}
	}
	json.NewEncoder(w).Encode(result)
}
