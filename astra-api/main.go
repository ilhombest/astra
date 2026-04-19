package main

import (
	"flag"
	"log"
	"net/http"
	"fmt"
)

var (
	flagPort     = flag.String("port", "8000", "HTTP port")
	flagLogin    = flag.String("login", "admin", "Basic auth login")
	flagPassword = flag.String("password", "admin", "Basic auth password")
	flagConfig   = flag.String("config", "/etc/astra/astra.lua", "Astra config path")
	flagAstraBin = flag.String("astra-bin", "/usr/bin/astra", "Astra binary path")
	flagPidFile  = flag.String("pid-file", "/var/run/astra.pid", "Astra PID file path")
)

func main() {
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/control/", basicAuth(*flagLogin, *flagPassword, handleControl))
	mux.HandleFunc("/api/", basicAuth(*flagLogin, *flagPassword, handleAPI))

	addr := fmt.Sprintf(":%s", *flagPort)
	log.Printf("astra-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
