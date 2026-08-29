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

	dbQueries := database.New(dbConn)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
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

	srv := &http.Server{
		Handler: srvMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
