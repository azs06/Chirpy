package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/azs06/Chirpy/internal/auth"
	"github.com/azs06/Chirpy/internal/database"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	defaultExpiresInSeconds := time.Hour

	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), sql.NullString{
		String: params.Email,
		Valid:  params.Email != "",
	})

	match, err := auth.CheckHashedPassword(params.Password, user.HashedPassword)
	if !match {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if params.ExpiresInSeconds > 0 {
		defaultExpiresInSeconds = time.Duration(params.ExpiresInSeconds) * time.Second
	}
	token, err := auth.MakeJWT(user.ID, cfg.tokenSecret, defaultExpiresInSeconds)
	refresh_token := auth.MakeRefreshToken()
	refresh_token_expiry := time.Now().Add(60 * 24 * time.Hour)
	tokenParams := database.CreateRefreshTokenParams{
		Token:  refresh_token,
		UserID: user.ID,
		ExpiresAt: sql.NullTime{
			Time:  refresh_token_expiry,
			Valid: true,
		},
		RevokedAt: sql.NullTime{},
	}
	tokenData, err := cfg.db.CreateRefreshToken(r.Context(), tokenParams)
	dat, _ := json.Marshal(userResp{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
		Email:        user.Email.String,
		Token:        token,
		RefreshToken: tokenData.Token,
		IsChirpyRed:  user.IsChirpyRed,
	})
	w.Write(dat)
	w.WriteHeader(http.StatusOK)
}
