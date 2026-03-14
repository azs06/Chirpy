package main

import (
	"fmt"
	"net/http"
)

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
}
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	cfg.resetMetrics()
	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write([]byte("Metrics reset\n"))
}
