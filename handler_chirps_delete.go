package main

import (
	"net/http"

	"github.com/azs06/Chirpy/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chirpId := r.PathValue("chirpId")
	chirpUUId, err := uuid.Parse(chirpId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	userId, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)

	if err != nil {
		w.WriteHeader(403)
		return
	}

	chirp, err := cfg.db.GetChirpByID(r.Context(), chirpUUId)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	if userId != chirp.UserID {
		w.WriteHeader(403)
		return
	}

	err = cfg.db.DeleteChirpById(r.Context(), chirpUUId)

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(204)
}
