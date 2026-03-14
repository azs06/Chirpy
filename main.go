package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/azs06/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	tokenSecret    string
	polkaKey       string
}

type userResp struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type chirpResp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    string    `json:"user_id"`
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
	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)

	mux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", cfg.handlerGetChirpByID)
	mux.HandleFunc("DELETE /api/chirps/{chirpId}", cfg.handlerDeleteChirp)

	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", cfg.handlerUpdateUser)

	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", cfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)

	mux.HandleFunc("POST /api/polka/webhooks", cfg.handlerWebhook)

	return &http.Server{
		Addr:    ":" + p,
		Handler: mux,
	}
}

func main() {
	godotenv.Load()
	platform, ok := os.LookupEnv("PLATFORM")
	if !ok {
		log.Fatal("PLATFORM not set")
	}
	tokenSecret, ok := os.LookupEnv("TOKEN_SECRET")
	if !ok {
		log.Fatal("Token not set")
	}
	dbURL, ok := os.LookupEnv("DB_URL")

	if !ok {
		log.Fatal("Database Url not set")
	}

	port := "8080"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	polkaKey := os.Getenv("POLKA_KEY")
	cfg := &apiConfig{
		platform:    platform,
		db:          database.New(db),
		tokenSecret: tokenSecret,
		polkaKey:    polkaKey,
	}
	fmt.Println("Starting Server on port " + port)
	s := newServer(port, cfg)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
