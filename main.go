package main

import (
	"fmt"
	"log"
	"encoding/json"
	"net/http"
	// "strconv"
	"strings"
	"sync/atomic"
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
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", int(cfg.fileserverHits.Load()))
	w.Write([]byte(html))
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

	srvMux.HandleFunc("GET /api/healthz", handlerFunc)
	srvMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	srvMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	srvMux.HandleFunc("POST /api/validate_chirp", handlerDecode)

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

func handlerDecode(w http.ResponseWriter, r *http.Request){
    type parameters struct {
        Body string `json:"body"`
    }

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
    }

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	} else {
		type returnVals struct {
			Cleaned_body string `json:"cleaned_body"`
		}

		cleaned, err := filterBadWords(params.Body)
		if err != nil {
			log.Printf("Error filtering body: %s", err)
			return
		}

		respBody := returnVals{
			Cleaned_body: cleaned,
		}
		respondWithJSON(w, 200, respBody)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
        Error string `json:"error"`
    }

	respBody := returnVals{
		Error: msg,
	}
    
    data, err := json.Marshal(respBody)
	if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(data)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    data, err := json.Marshal(payload)
	if err != nil {
			// respondWithError(w, 400, "Something went wrong")
			log.Printf("Error marshalling JSON: %s", err)
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(data)
}

func filterBadWords(msg string) (string, error) {

	profound := []string{ 
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	badMap := make(map[string]bool)
	for _, bad := range profound {
		badMap[strings.ToLower(bad)] = true
	}

	lists := strings.Split(msg, " ")
	filtered := []string{}

	for _, word := range lists {
		cleanedWord := strings.Trim(word, ".,!?;:()\"'")

		if badMap[strings.ToLower(cleanedWord)] {
			censored := strings.Replace(strings.ToLower(word), strings.ToLower(cleanedWord), "****", 1)
			filtered = append(filtered, censored)
		} else {
			filtered = append(filtered, word)
		}
	}

	final := strings.Join(filtered, " ")

	return final, nil
}