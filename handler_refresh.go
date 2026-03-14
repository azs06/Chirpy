package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/azs06/Chirpy/internal/auth"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type respParams struct {
		Token string `json:"token"`
	}
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	refresh_token, err := cfg.db.GetRefreshToken(r.Context(), bearerToken)

	if err != nil || refresh_token.RevokedAt.Valid || refresh_token.ExpiresAt.Time.Before(time.Now()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := cfg.db.GetUserById(r.Context(), refresh_token.UserID)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.tokenSecret, time.Hour)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := json.Marshal(respParams{
		Token: token,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	refresh_token, err := cfg.db.GetRefreshToken(r.Context(), bearerToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = cfg.db.RevokeRefreshToken(r.Context(), refresh_token.Token)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
