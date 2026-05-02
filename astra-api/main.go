package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var (
	flagPort     = flag.String("port", "8000", "HTTP port")
	flagLogin    = flag.String("login", "admin", "Basic auth login")
	flagPassword = flag.String("password", "admin", "Basic auth password")
	flagConfig   = flag.String("config", "/etc/astra/astra.lua", "Astra config path")
	flagAstraBin = flag.String("astra-bin", "/usr/bin/astra", "Astra binary path")
	flagPidFile  = flag.String("pid-file", "/var/run/astra.pid", "Astra PID file path")
)

//go:embed ui/index.html
var uiHTML string

func main() {
	flag.Parse()
	initLogger()
	go attachOrStartAstra()

	mux := http.NewServeMux()

	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return basicAuth(*flagLogin, *flagPassword, h)
	}

	// Web UI
	mux.HandleFunc("/", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, uiHTML)
	}))

	// Existing API bridge
	mux.HandleFunc("/control/", auth(handleControl))

	// API: system, stream-status, adapter-status, log + new CRUD
	mux.HandleFunc("/api/", auth(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/")
		// allow CORS preflight
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch {
		case path == "streams-status":
			w.Header().Set("Content-Type", "application/json")
			handleStreamsStatus(w, r)
		case path == "streams" || strings.HasPrefix(path, "streams/"):
			handleStreamsAPI(w, r)
		case path == "adapters" || strings.HasPrefix(path, "adapters/"):
			handleAdaptersAPI(w, r)
		case path == "cams" || strings.HasPrefix(path, "cams/"):
			handleCamsAPI(w, r)
		case path == "interfaces":
			handleInterfaces(w, r)
		default:
			handleAPI(w, r)
		}
	}))

	addr := fmt.Sprintf(":%s", *flagPort)
	log.Printf("astra-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
