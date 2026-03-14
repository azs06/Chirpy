package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/azs06/Chirpy/internal/auth"
	"github.com/azs06/Chirpy/internal/database"
)

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type errResp struct {
		Error string `json:"error"`
	}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userId, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)

	if err != nil {
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		dat, _ := json.Marshal(errResp{
			Error: "Something went wrong",
		})
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		w.Write(dat)
		return
	}
	if len(params.Body) > 140 {
		dat, _ := json.Marshal(errResp{
			Error: "Chirp is too long",
		})
		w.WriteHeader(400)
		w.Write(dat)
		return
	}
	chirpParam := database.CreateChirpParams{
		Body: sql.NullString{
			String: sanitize(params.Body),
			Valid:  true,
		},
		UserID: userId,
	}
	chirp, err := cfg.db.CreateChirp(r.Context(), chirpParam)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}

	dat, _ := json.Marshal(chirpResp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt.Time,
		UpdatedAt: chirp.UpdatedAt.Time,
		Body:      chirp.Body.String,
		UserId:    chirp.UserID.String(),
	})
	w.WriteHeader(201)
	w.Write(dat)
}
