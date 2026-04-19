package main

import (
	"net/http"
)

func handleAPI(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
