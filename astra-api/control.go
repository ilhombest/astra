package main

import (
	"encoding/json"
	"net/http"
)

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"status": false, "error": msg})
}

func handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}

	cmd, _ := req["cmd"].(string)
	w.Header().Set("Content-Type", "application/json")

	switch cmd {
	case "version":
		json.NewEncoder(w).Encode(map[string]string{
			"version": "4.4.199",
			"commit":  "opensource",
		})

	case "load":
		cfg, err := loadConfig()
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(cfg)

	case "upload":
		cfg, ok := req["config"].(map[string]any)
		if !ok {
			jsonError(w, `"config" field required`, http.StatusBadRequest)
			return
		}
		if err := saveConfig(cfg); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"status": true})

	case "restart":
		if err := restartAstra(); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"status": true})

	case "sessions":
		out, err := getSessions()
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(out)

	default:
		jsonError(w, "unknown command", http.StatusBadRequest)
	}
}
