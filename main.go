package main

import (
	"net/http"
)

func main() {

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", handlerReadiness)
	server := http.Server{Addr: ":8080", Handler: mux}
	server.ListenAndServe()
}
