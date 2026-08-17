package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"strconv"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hitsInputs := "Hits: " + strconv.Itoa(int(cfg.fileserverHits.Load()))
	w.Write([]byte(hitsInputs))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset"))
}

func main(){
	const filepathRoot = "."
	const port = "8080"

	apiCfg := apiConfig{}

	srvMux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	srvMux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	srvMux.HandleFunc("GET /healthz", handlerFunc)
	srvMux.HandleFunc("GET /metrics", apiCfg.handlerMetrics)
	srvMux.HandleFunc("POST /reset", apiCfg.handlerReset)

	srv := &http.Server{
		Handler: srvMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}