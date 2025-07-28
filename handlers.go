package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func handlerReadiness(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(writer, req)
	})
}

func (cfg *apiConfig) handlerMetrics(writer http.ResponseWriter, req *http.Request) {
	str := fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", cfg.fileserverHits.Load())
	writer.Header().Set("Content-Type", "text/html")
	writer.Write([]byte(str))
}

func (cfg *apiConfig) handlerReset(writer http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
}

func handlerValidateChirp(writer http.ResponseWriter, req *http.Request) {
	type Chirp struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	chirp := Chirp{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	type Response struct {
		CleanedBody string `json:"cleaned_body"`
	}

	if len(chirp.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long", err)
		return
	}

	cleaned_body := replaceProfaneText(chirp.Body)

	response := Response{
		CleanedBody: cleaned_body,
	}

	respondWithJSON(writer, 200, response)
}

func handlerCreateUser(writer http.ResponseWriter, req *http.Request) {
}
