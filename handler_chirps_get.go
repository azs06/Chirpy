package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/azs06/Chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	author_id := r.URL.Query().Get("author_id")
	sort := r.URL.Query().Get("sort")
	var chirps []database.Chirp
	var err error
	var author_uuid uuid.UUID
	resp := make([]chirpResp, 0, len(chirps))

	if author_id != "" {
		author_uuid, err = uuid.Parse(author_id)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		chirps, err = cfg.db.GetChirpsByUserId(r.Context(), author_uuid)
	} else {
		chirps, err = cfg.db.GetChirps(r.Context())
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	if sort == "desc" {
		slices.Reverse(chirps)
	}

	for _, c := range chirps {
		resp = append(resp, chirpResp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt.Time,
			UpdatedAt: c.UpdatedAt.Time,
			Body:      c.Body.String,
			UserId:    c.UserID.String(),
		})
	}
	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.WriteHeader(200)
	w.Write(dat)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("chirpId")
	chirpUUId, err := uuid.Parse(chirpId)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println(err)
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
		return
	}

	chirp, err := cfg.db.GetChirpByID(r.Context(), chirpUUId)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	dat, _ := json.Marshal(chirpResp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt.Time,
		UpdatedAt: chirp.UpdatedAt.Time,
		Body:      chirp.Body.String,
		UserId:    chirp.UserID.String(),
	})
	w.WriteHeader(200)
	w.Write(dat)
}
