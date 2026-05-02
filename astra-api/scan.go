package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ScannedService holds a DVB service found during a transponder scan.
type ScannedService struct {
	SID      int    `json:"sid"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	TSID     int    `json:"tsid"`
}

var (
	reScanTSID     = regexp.MustCompile(`SDT:\s*tsid:\s*(\d+)`)
	reScanSID      = regexp.MustCompile(`SDT:\s*sid:\s*(\d+)`)
	reScanSvcName  = regexp.MustCompile(`Service:\s*(.+)`)
	reScanProvider = regexp.MustCompile(`Provider:\s*(.+)`)
)

// handleScanAdapter: POST /api/adapters/{id}/scan
// Starts astra in stream mode with a temp config on the given adapter,
// waits for SDT output (up to 25 s), and returns the list of services.
func handleScanAdapter(w http.ResponseWriter, r *http.Request, adapterID string) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", 405)
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	adapters, _ := cfg["adapters"].(map[string]any)
	adRaw, ok := adapters[adapterID]
	if !ok {
		jsonError(w, "adapter not found: "+adapterID, 404)
		return
	}
	a, _ := adRaw.(map[string]any)

	// Write temp Lua config
	lua := buildScanLua(adapterID, a)
	tmp, err := os.CreateTemp("", "astra-scan-*.lua")
	if err != nil {
		jsonError(w, "tmp file: "+err.Error(), 500)
		return
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString(lua)
	tmp.Close()

	// DVB adapters use exclusive O_RDWR locking — stop main astra to release the adapter
	addLog("info", fmt.Sprintf("[scan] adapter %s: stopping main astra to release DVB lock", adapterID))
	stopAstra()
	time.Sleep(800 * time.Millisecond)
	defer func() {
		addLog("info", fmt.Sprintf("[scan] adapter %s: restarting main astra", adapterID))
		time.Sleep(300 * time.Millisecond)
		startAstra()
	}()

	addLog("info", fmt.Sprintf("[scan] adapter %s: starting scan", adapterID))

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, *flagAstraBin, "--stream", tmp.Name(), "--debug")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		jsonError(w, "astra start: "+err.Error(), 500)
		return
	}

	// Wait until SDT crc32 line appears (scan done) or timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		cmd.Wait()
	}()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(40 * time.Second)

	sdtDone := false
outer:
	for {
		select {
		case <-done:
			break outer
		case <-deadline:
			cmd.Process.Kill()
			<-done
			break outer
		case <-ticker.C:
			if strings.Contains(buf.String(), "SDT: crc32:") {
				// SDT fully parsed — wait a bit more then kill
				if !sdtDone {
					sdtDone = true
					time.AfterFunc(2*time.Second, func() {
						cmd.Process.Kill()
					})
				}
			}
		}
	}

	output := buf.String()
	addLog("info", fmt.Sprintf("[scan] adapter %s: scan finished, parsing output", adapterID))
	if len(output) > 0 {
		preview := output
		if len(preview) > 3000 {
			preview = preview[:3000]
		}
		addLog("info", fmt.Sprintf("[scan] adapter %s raw output:\n%s", adapterID, preview))
	}

	services := parseSDTOutput(output)
	addLog("info", fmt.Sprintf("[scan] adapter %s: found %d services", adapterID, len(services)))

	resp := map[string]any{
		"adapter":  adapterID,
		"services": services,
	}
	if len(services) == 0 && len(output) > 0 {
		preview := output
		if len(preview) > 2000 {
			preview = preview[:2000]
		}
		resp["debug_output"] = preview
	}
	writeJSON(w, 200, resp)
}

// buildScanLua generates a minimal --stream Lua config for a single adapter,
// pointing make_channel at the full MPTS (no PNR filter) so SDT is received.
func buildScanLua(adapterNum string, a map[string]any) string {
	n := adapterNum
	dvbType := strings.TrimPrefix(anyStr(a["dvb_type"]), "DVB-")
	if dvbType == "" {
		dvbType = "S2"
	}

	var sb strings.Builder
	sb.WriteString("-- Auto-generated transponder scan\n\n")

	sb.WriteString(fmt.Sprintf("dvb%s = dvb_tune({\n", n))
	sb.WriteString(fmt.Sprintf("  type = %q,\n", dvbType))

	isSat := dvbType == "S" || dvbType == "S2"
	isTer := dvbType == "T" || dvbType == "T2" || dvbType == "ATSC" || dvbType == "ISDB-T"

	if isSat {
		sb.WriteString(fmt.Sprintf("  tp = %q,\n",
			fmt.Sprintf("%s:%s:%s", anyStr(a["frequency"]), anyStr(a["polarization"]), anyStr(a["symbolrate"]))))
		sb.WriteString(fmt.Sprintf("  lnb = %q,\n",
			fmt.Sprintf("%s:%s:%s", anyStr(a["lof1"]), anyStr(a["lof2"]), anyStr(a["slof"]))))
	} else if isTer {
		sb.WriteString(fmt.Sprintf("  tp = %q,\n",
			fmt.Sprintf("%s:%s", anyStr(a["frequency"]), anyStr(a["bandwidth"]))))
	} else { // DVB-C
		sb.WriteString(fmt.Sprintf("  tp = %q,\n",
			fmt.Sprintf("%s:%s", anyStr(a["frequency"]), anyStr(a["symbolrate"]))))
	}

	sb.WriteString(fmt.Sprintf("  adapter = %s,\n", anyStr(a["adapter"])))
	if dev := anyStr(a["device"]); dev != "" && dev != "0" {
		sb.WriteString(fmt.Sprintf("  device = %s,\n", dev))
	}
	// budget=true delivers all PIDs (including SDT PID 17) without hardware filtering
	sb.WriteString("  budget = true,\n")
	sb.WriteString("})\n\n")

	// Full MPTS (no PNR) — output to loopback UDP so the channel actually starts
	sb.WriteString("make_channel({\n")
	sb.WriteString(fmt.Sprintf("  name = %q,\n", "scan_"+n))
	sb.WriteString(fmt.Sprintf("  id   = %q,\n", "scan_"+n))
	sb.WriteString("  enable = true,\n")
	sb.WriteString(fmt.Sprintf("  input  = {\"dvb://dvb%s\"},\n", n))
	sb.WriteString("  output = {\"udp://127.0.0.1:29876\"},\n")
	sb.WriteString("})\n")

	return sb.String()
}

// parseSDTOutput extracts ScannedService entries from astra log output.
// Expected patterns (anywhere in the line):
//
//	SDT: tsid: 1234
//	SDT: sid: 5601
//	SDT:    Service: Channel Name
//	SDT:    Provider: Provider Name
func parseSDTOutput(output string) []ScannedService {
	var services []ScannedService
	var cur *ScannedService
	tsid := 0

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if m := reScanTSID.FindStringSubmatch(line); m != nil {
			tsid, _ = strconv.Atoi(m[1])
			continue
		}
		if m := reScanSID.FindStringSubmatch(line); m != nil {
			if cur != nil && cur.SID > 0 {
				services = append(services, *cur)
			}
			sid, _ := strconv.Atoi(m[1])
			cur = &ScannedService{SID: sid, TSID: tsid}
			continue
		}
		if cur != nil {
			if m := reScanSvcName.FindStringSubmatch(line); m != nil {
				cur.Name = strings.TrimSpace(m[1])
			}
			if m := reScanProvider.FindStringSubmatch(line); m != nil {
				cur.Provider = strings.TrimSpace(m[1])
			}
		}
	}
	if cur != nil && cur.SID > 0 {
		services = append(services, *cur)
	}
	return services
}
