package main

import (
	"net/http"
)

func basicAuth(login, password string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != login || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="astra-api"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
