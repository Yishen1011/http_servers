package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Yishen1011/http_servers/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

func main(){
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening dbURL: %s", err)
    }

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("In .env file PLATFORM must be set")
	}

	jwt_secret := os.Getenv("JWT_SECRET")
	if jwt_secret == "" {
		log.Fatal("In .env file JWT_SECRET must be set")
	}

	polka_key := os.Getenv("POLKA_KEY")
	if polka_key == "" {
		log.Fatal("In .env file JWT_SECRET must be set")
	}

	dbQueries := database.New(dbConn)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
		jwtSecret:      jwt_secret,
		polkaKey:       polka_key,
	}

	srvMux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	srvMux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	srvMux.HandleFunc("GET /api/healthz", handlerReadiness)
	srvMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	srvMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	srvMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	srvMux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	srvMux.HandleFunc("GET /api/chirps", apiCfg.handlerListChirps)
	srvMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
	srvMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	srvMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefreshToken)
	srvMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeToken)
	srvMux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUserPW)
	srvMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirpFromID)
	srvMux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerUpdateUserChirpyRed)

	srv := &http.Server{
		Handler: srvMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
