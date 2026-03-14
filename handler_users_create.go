package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/azs06/Chirpy/internal/auth"
	"github.com/azs06/Chirpy/internal/database"
)

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type errResp struct {
		Error string `json:"error"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	hPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	userData := database.CreateUserParams{
		Email: sql.NullString{
			String: params.Email,
			Valid:  params.Email != "",
		},
		HashedPassword: hPassword,
	}
	user, err := cfg.db.CreateUser(r.Context(), userData)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}

	dat, _ := json.Marshal(userResp{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:   user.UpdatedAt.Time,
		Email:       user.Email.String,
		IsChirpyRed: user.IsChirpyRed,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)
}
