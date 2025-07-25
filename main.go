package main

import (
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {

	var apiCfg apiConfig
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /api/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /api/reset", apiCfg.handlerReset)
	server := http.Server{Addr: ":8080", Handler: mux}
	server.ListenAndServe()
}
