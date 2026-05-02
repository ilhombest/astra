package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func ok200(w http.ResponseWriter) {
	writeJSON(w, 200, map[string]any{"status": true})
}

func idFromPath(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix+"/")
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}

func genID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli())
}

// ── streams ───────────────────────────────────────────────────────────────────

func handleStreamsAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")

	id := idFromPath(path, "streams")
	rest := strings.TrimPrefix(path, "streams/"+id)

	switch {
	case path == "streams" && r.Method == http.MethodGet:
		streamsListHandler(w, r)
	case path == "streams" && r.Method == http.MethodPost:
		streamsCreateHandler(w, r)
	case strings.HasPrefix(path, "streams/") && rest == "/restart":
		streamsRestartHandler(w, r, id)
	case strings.HasPrefix(path, "streams/") && r.Method == http.MethodPut:
		streamsUpdateHandler(w, r, id)
	case strings.HasPrefix(path, "streams/") && r.Method == http.MethodDelete:
		streamsDeleteHandler(w, r, id)
	default:
		http.Error(w, "not found", 404)
	}
}

func streamsListHandler(w http.ResponseWriter, _ *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	streams, _ := cfg["streams"].(map[string]any)
	out := make([]any, 0, len(streams))
	for id, v := range streams {
		s, _ := v.(map[string]any)
		row := copyMap(s)
		row["_id"] = id
		out = append(out, row)
	}
	writeJSON(w, 200, out)
}

func streamsCreateHandler(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	id := genID("stream")
	body["id"] = id

	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureStreams(cfg)[id] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	writeJSON(w, 200, map[string]any{"status": true, "id": id})
}

func streamsRestartHandler(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	streams := ensureStreams(cfg)
	s, ok := streams[id]
	if !ok {
		jsonError(w, "stream not found", 404)
		return
	}

	action := r.URL.Query().Get("action")
	sm, _ := s.(map[string]any)

	switch action {
	case "stop":
		sm["enable"] = false
		streams[id] = sm
		if err := saveConfig(cfg); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
	case "start":
		sm["enable"] = true
		streams[id] = sm
		if err := saveConfig(cfg); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
	default: // restart — just reload astra with current config
	}

	go restartAstra()
	ok200(w)
}

func streamsUpdateHandler(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	body["id"] = id

	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureStreams(cfg)[id] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func streamsDeleteHandler(w http.ResponseWriter, _ *http.Request, id string) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	delete(ensureStreams(cfg), id)
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func ensureStreams(cfg map[string]any) map[string]any {
	if m, ok := cfg["streams"].(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	cfg["streams"] = m
	return m
}

// ── adapters ──────────────────────────────────────────────────────────────────

func handleAdaptersAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")

	switch {
	case path == "adapters" && r.Method == http.MethodGet:
		adaptersListHandler(w, r)
	case path == "adapters" && r.Method == http.MethodPost:
		adaptersCreateHandler(w, r)
	case strings.HasPrefix(path, "adapters/"):
		id := idFromPath(path, "adapters")
		rest := strings.TrimPrefix(path, "adapters/"+id)
		switch {
		case rest == "/scan":
			handleScanAdapter(w, r, id)
		case rest == "" && r.Method == http.MethodPut:
			adaptersUpdateHandler(w, r, id)
		case rest == "" && r.Method == http.MethodDelete:
			adaptersDeleteHandler(w, r, id)
		default:
			http.Error(w, "not found", 404)
		}
	default:
		http.Error(w, "not found", 404)
	}
}

func adaptersListHandler(w http.ResponseWriter, _ *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	adapters, _ := cfg["adapters"].(map[string]any)
	out := make([]any, 0, len(adapters))
	for id, v := range adapters {
		a, _ := v.(map[string]any)
		row := copyMap(a)
		row["_id"] = id
		out = append(out, row)
	}
	writeJSON(w, 200, out)
}

func adaptersCreateHandler(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	// key is the adapter number
	adapterNum := fmt.Sprintf("%v", body["adapter"])
	if adapterNum == "" || adapterNum == "<nil>" || adapterNum == "0" {
		adapterNum = genID("a")
	}

	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureAdapters(cfg)[adapterNum] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	writeJSON(w, 200, map[string]any{"status": true, "id": adapterNum})
}

func adaptersUpdateHandler(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureAdapters(cfg)[id] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func adaptersDeleteHandler(w http.ResponseWriter, _ *http.Request, id string) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	delete(ensureAdapters(cfg), id)
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func ensureAdapters(cfg map[string]any) map[string]any {
	if m, ok := cfg["adapters"].(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	cfg["adapters"] = m
	return m
}

// ── cams (newcamd) ────────────────────────────────────────────────────────────

func handleCamsAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")

	switch {
	case path == "cams" && r.Method == http.MethodGet:
		camsListHandler(w, r)
	case path == "cams" && r.Method == http.MethodPost:
		camsCreateHandler(w, r)
	case strings.HasPrefix(path, "cams/") && r.Method == http.MethodPut:
		camsUpdateHandler(w, r, idFromPath(path, "cams"))
	case strings.HasPrefix(path, "cams/") && r.Method == http.MethodDelete:
		camsDeleteHandler(w, r, idFromPath(path, "cams"))
	default:
		http.Error(w, "not found", 404)
	}
}

func camsListHandler(w http.ResponseWriter, _ *http.Request) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	cams, _ := cfg["cams"].(map[string]any)
	out := make([]any, 0, len(cams))
	for id, v := range cams {
		c, _ := v.(map[string]any)
		row := copyMap(c)
		row["_id"] = id
		out = append(out, row)
	}
	writeJSON(w, 200, out)
}

func camsCreateHandler(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	body["type"] = "newcamd"
	id := genID("cam")

	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureCams(cfg)[id] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	writeJSON(w, 200, map[string]any{"status": true, "id": id})
}

func camsUpdateHandler(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad json", 400)
		return
	}
	body["type"] = "newcamd"
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	ensureCams(cfg)[id] = body
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func camsDeleteHandler(w http.ResponseWriter, _ *http.Request, id string) {
	cfg, err := loadConfig()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	delete(ensureCams(cfg), id)
	if err := saveConfig(cfg); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	go restartAstra()
	ok200(w)
}

func ensureCams(cfg map[string]any) map[string]any {
	if m, ok := cfg["cams"].(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	cfg["cams"] = m
	return m
}

// ── misc ──────────────────────────────────────────────────────────────────────

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
