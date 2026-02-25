package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/azs06/Chirpy/internal/auth"
	"github.com/azs06/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}
type chirpResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    string    `json:"user_id"`
}

type userResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) resetMetrics() {
	cfg.fileserverHits.Store(0)
}

func sanitize(s string) string {
	strSlice := strings.Split(s, " ")
	rtSlice := []string{}
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	for _, v := range strSlice {
		clean := strings.ToLower(strings.Trim(v, ".,!?"))
		if slices.Contains(badWords, clean) {
			rtSlice = append(rtSlice, "****")
		} else {
			rtSlice = append(rtSlice, v)
		}
	}
	return strings.Join(rtSlice, " ")
}

func newServer(p string, cfg *apiConfig) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", cfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		//w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if cfg.platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		cfg.resetMetrics()
		err := cfg.db.DeleteUsers(r.Context())
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Write([]byte("Metrics reset\n"))
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body   string    `json:"body"`
			UserId uuid.UUID `json:"user_id"`
		}
		type errResp struct {
			Error string `json:"error"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
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
			UserID: params.UserId,
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
	})
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := cfg.db.GetChirps(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			return
		}

		type chirp struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    string    `json:"user_id"`
		}
		chirpResp := []chirp{}

		for _, c := range chirps {
			chirpResp = append(chirpResp, chirp{
				ID:        c.ID,
				CreatedAt: c.CreatedAt.Time,
				UpdatedAt: c.UpdatedAt.Time,
				Body:      c.Body.String,
				UserId:    c.UserID.String(),
			})
		}
		dat, _ := json.Marshal(chirpResp)
		w.WriteHeader(200)
		w.Write(dat)

	})

	mux.HandleFunc("GET /api/chirps/{chirpId}", func(w http.ResponseWriter, r *http.Request) {
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

	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
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
			ID:        user.ID,
			CreatedAt: user.CreatedAt.Time,
			UpdatedAt: user.UpdatedAt.Time,
			Email:     user.Email.String,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)

	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
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
		dat, _ := json.Marshal(userResp{
			ID:        user.ID,
			CreatedAt: user.CreatedAt.Time,
			UpdatedAt: user.UpdatedAt.Time,
			Email:     user.Email.String,
		})
		w.Write(dat)
		w.WriteHeader(http.StatusOK)
	})
	return &http.Server{
		Addr:    ":" + p,
		Handler: mux,
	}
}

func main() {
	godotenv.Load()
	platform, ok := os.LookupEnv("PLATFORM")
	dbURL, _ := os.LookupEnv("DB_URL")
	if !ok {
		log.Fatal("PLATFORM not set")
	}
	port := "8080"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	cfg := &apiConfig{
		platform: platform,
		db:       database.New(db),
	}
	fmt.Println("Starting Server on port " + port)
	s := newServer(port, cfg)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
