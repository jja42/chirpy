package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jja42/chirpy/internal/auth"
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

	cfg.fileserverHits.Store(0)

	err := cfg.db.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Unable to Delete Users", err)
		return
	}
}

func (cfg *apiConfig) handlerCreateUser(writer http.ResponseWriter, req *http.Request) {
	type Parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := Parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		respondWithError(writer, 500, "Unable to Create User", err)
		return
	}

	user_params := database.CreateUserParams{Email: params.Email, HashedPassword: hashed_password}

	user, err := cfg.db.CreateUser(req.Context(), user_params)
	if err != nil {
		respondWithError(writer, 500, "Unable to Create User", err)
		return
	}

	user_response := UserResponse{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}

	respondWithJSON(writer, 201, user_response)
}

func (cfg *apiConfig) handlerUpdateUser(writer http.ResponseWriter, req *http.Request) {
	//Get Access Token
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "Unable to Get Client Token", err)
		return
	}

	user_id, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		respondWithError(writer, 401, "Unauthorized Request", err)
		return
	}

	//New Params
	type Parameters struct {
		NewPassword string `json:"password"`
		NewEmail    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := Parameters{}

	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	hashed_password, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		respondWithError(writer, 500, "Unable to Update User", err)
		return
	}

	//Update User
	user, err := cfg.db.UpdateUser(req.Context(), database.UpdateUserParams{ID: user_id, Email: params.NewEmail, HashedPassword: hashed_password})
	if err != nil {
		respondWithError(writer, 401, "Incorrect email or password", err)
		return
	}

	user_response := UserResponse{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}

	respondWithJSON(writer, 200, user_response)
}

func (cfg *apiConfig) handlerLogin(writer http.ResponseWriter, req *http.Request) {
	type Parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := Parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		respondWithError(writer, 500, "Unable to Decode JSON", err)
		return
	}

	user, err := cfg.db.GetUser(req.Context(), params.Email)
	if err != nil {
		respondWithError(writer, 401, "Incorrect email or password", err)
		return
	}

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(writer, 401, "Incorrect email or password", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwt_secret, time.Hour)
	if err != nil {
		respondWithError(writer, 401, "Unable to Make JWT", err)
		return
	}

	refresh_token := auth.MakeRefreshToken()

	cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token: refresh_token, UserID: user.ID, ExpiresAt: time.Now().AddDate(0, 0, 60),
	})

	user_response := UserResponse{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, Token: token, RefreshToken: refresh_token}

	respondWithJSON(writer, 200, user_response)
}

func (cfg *apiConfig) handlerCreateChirp(writer http.ResponseWriter, req *http.Request) {

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "Unable to Get Client Token", err)
		return
	}

	user_id, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		respondWithError(writer, 401, "Unauthorized Request", err)
		return
	}

	//Ported Validate Chirp Logic
	type Request struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	r := Request{}
	err = decoder.Decode(&r)
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

	params := database.CreateChirpParams{Body: cleaned_body, UserID: user_id}

	chirp, err := cfg.db.CreateChirp(req.Context(), params)
	if err != nil {
		respondWithError(writer, 500, "Unable to Create Chirp", err)
		return
	}

	response := ChirpResponse{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}

	respondWithJSON(writer, 201, response)
}

func (cfg *apiConfig) handlerGetChirps(writer http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.db.GetChirps(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Unable to Get Chirps", err)
		return
	}

	var Chirps []ChirpResponse

	for _, chirp := range chirps {
		Chirps = append(Chirps, ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	respondWithJSON(writer, 200, Chirps)
}

func (cfg *apiConfig) handlerGetChirp(writer http.ResponseWriter, req *http.Request) {
	id_string := req.PathValue("chirpID")
	id, err := uuid.Parse(id_string)
	if err != nil {
		respondWithError(writer, 500, "Could Not Parse Chirp ID from Path Value", err)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), id)
	if err != nil {
		respondWithError(writer, 404, "Chirp Not Found", err)
		return
	}

	response := ChirpResponse{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.CreatedAt, Body: chirp.Body, UserID: chirp.UserID}

	respondWithJSON(writer, 200, response)
}

func (cfg *apiConfig) handlerDeleteChirp(writer http.ResponseWriter, req *http.Request) {
	id_string := req.PathValue("chirpID")
	id, err := uuid.Parse(id_string)
	if err != nil {
		respondWithError(writer, 500, "Could Not Parse Chirp ID from Path Value", err)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), id)
	if err != nil {
		respondWithError(writer, 404, "Chirp Not Found", err)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "Unable to Get Client Token", err)
		return
	}

	user_id, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		respondWithError(writer, 401, "Unauthorized Request", err)
		return
	}

	if chirp.UserID != user_id {
		respondWithError(writer, 403, "Unauthorized Request", err)
		return
	}

	cfg.db.DeleteChirp(req.Context(), chirp.ID)

	respondWithJSON(writer, 204, nil)
}

func (cfg *apiConfig) handlerRefresh(writer http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 500, "Unable to Get Refresh Token from Header", err)
		return
	}

	refresh_token, err := cfg.db.GetRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(writer, 401, "Unable to Get Refresh Token from Database", err)
		return
	}

	expired := refresh_token.ExpiresAt.Before(time.Now())
	if expired {
		respondWithError(writer, 401, "Refresh Token is Expired", err)
		return
	}

	revoked := refresh_token.RevokedAt.Valid
	if revoked {
		respondWithError(writer, 401, "Refresh Token is Revoked", err)
		return
	}

	access_token, err := auth.MakeJWT(refresh_token.UserID, cfg.jwt_secret, time.Hour)
	if err != nil {
		respondWithError(writer, 401, "Unable to Make JWT", err)
		return
	}

	type Response struct {
		Token string `json:"token"`
	}

	response := Response{Token: access_token}

	respondWithJSON(writer, 200, response)
}

func (cfg *apiConfig) handlerRevoke(writer http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 500, "Unable to Get Refresh Token from Header", err)
		return
	}

	err = cfg.db.RevokeRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(writer, 401, "Unable to Revoke Token. Does Not Exist.", err)
		return
	}

	respondWithJSON(writer, 204, nil)
}
