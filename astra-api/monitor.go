package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StreamStat struct {
	OnAir   bool      `json:"on_air"`
	Bitrate int       `json:"bitrate"`
	CCErr   int       `json:"cc_error"`
	PESErr  int       `json:"pes_error"`
	Updated time.Time `json:"-"`
}

var (
	streamStats   = map[string]*StreamStat{}
	streamStatsMu sync.RWMutex

	reBitrate  = regexp.MustCompile(`\[(.+?)\] Bitrate:(\d+)Kbit/s`)
	reInputSfx = regexp.MustCompile(` #\d+$`)
	reCC      = regexp.MustCompile(`CC:(\d+)`)
	rePES     = regexp.MustCompile(`PES:(\d+)`)
)

func init() {
	go func() {
		var lastLen int
		for range time.Tick(2 * time.Second) {
			logMu.RLock()
			entries := logBuf[lastLen:]
			newLen := len(logBuf)
			logMu.RUnlock()

			for _, e := range entries {
				parseStatLine(e.Msg)
			}
			lastLen = newLen
		}
	}()
}

func parseStatLine(msg string) {
	m := reBitrate.FindStringSubmatch(msg)
	if m == nil {
		return
	}
	name := reInputSfx.ReplaceAllString(m[1], "")
	bitrate, _ := strconv.Atoi(m[2])

	offAir := strings.Contains(msg, "PES:") || strings.Contains(msg, "CC:") || strings.Contains(msg, "Scrambled")
	onAir := !offAir

	cc := 0
	if mc := reCC.FindStringSubmatch(msg); mc != nil {
		cc, _ = strconv.Atoi(mc[1])
	}
	pes := 0
	if mp := rePES.FindStringSubmatch(msg); mp != nil {
		pes, _ = strconv.Atoi(mp[1])
	}

	streamStatsMu.Lock()
	streamStats[name] = &StreamStat{
		OnAir:   onAir,
		Bitrate: bitrate,
		CCErr:   cc,
		PESErr:  pes,
		Updated: time.Now(),
	}
	streamStatsMu.Unlock()
}

// getStatByName returns the latest stat for a stream name.
// on_air=true states are kept indefinitely (astra only logs on state change).
// on_air=false states expire after 2 minutes (stream may have recovered).
func getStatByName(name string) *StreamStat {
	streamStatsMu.RLock()
	defer streamStatsMu.RUnlock()
	s, ok := streamStats[name]
	if !ok {
		return nil
	}
	if !s.OnAir && time.Since(s.Updated) > 2*time.Minute {
		return nil
	}
	return s
}

// getStatByID looks up a stream's name from config then returns its stat.
func getStatByID(id string) *StreamStat {
	cfg, err := loadConfig()
	if err != nil {
		return nil
	}
	streams, _ := cfg["streams"].(map[string]any)
	s, ok := streams[id]
	if !ok {
		return nil
	}
	sm, _ := s.(map[string]any)
	name, _ := sm["name"].(string)
	if name == "" {
		return nil
	}
	return getStatByName(name)
}
