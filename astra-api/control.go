package main

import (
	"encoding/json"
	"net/http"
)

func handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
	default:
		http.Error(w, "Unknown command", http.StatusBadRequest)
	}
}
