package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jja42/chirpy/internal/database"
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
	if cfg.platform != "dev" {
		respondWithError(writer, 403, "Access Forbidden", nil)
		return
	}
	err := cfg.db.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Unable to Delete Users", err)
	}
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) handlerCreateChirp(writer http.ResponseWriter, req *http.Request) {

	//Ported Validate Chirp Logic
	type Request struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	r := Request{}
	err := decoder.Decode(&r)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	if len(r.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long", err)
		return
	}

	//Chirp is Valid and will be Cleaned

	cleaned_body := replaceProfaneText(r.Body)

	//Now we need to touch the database

	params := database.CreateChirpParams{Body: cleaned_body, UserID: r.UserID}

	chirp, err := cfg.db.CreateChirp(req.Context(), params)
	if err != nil {
		respondWithError(writer, 500, "Unable to Create Chirp", err)
		return
	}

	response := Chirp{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}

	respondWithJSON(writer, 201, response)
}

func (cfg *apiConfig) handlerCreateUser(writer http.ResponseWriter, req *http.Request) {
	type Parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := Parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	user, err := cfg.db.CreateUser(req.Context(), params.Email)
	if err != nil {
		respondWithError(writer, 500, "Unable to Create User", err)
		return
	}

	user_response := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}

	respondWithJSON(writer, 201, user_response)
}

func (cfg *apiConfig) handlerGetChirps(writer http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.db.GetChirps(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Unable to Get Chirps", err)
		return
	}

	var Chirps []Chirp

	for _, chirp := range chirps {
		Chirps = append(Chirps, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	respondWithJSON(writer, 200, Chirps)
}
