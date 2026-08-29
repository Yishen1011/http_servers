package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Yishen1011/http_servers/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request){
    type parameters struct {
        Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
    }

	type response struct {
		Chirp
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
    }

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", err)
		return
	}

	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	cleaned := filterBadWords(params.Body, badWords)

	args := database.CreateChirpParams{
		Body:   cleaned,
		UserID: params.UserID,
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), args)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Chirp can't be created", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Chirp: Chirp{
			ID:          chirp.ID,
			CreatedAt:   chirp.CreatedAt,
			UpdatedAt:   chirp.UpdatedAt,
			Body:        chirp.Body,
			UserID:      chirp.UserID,
		},
	})
}

func filterBadWords(msg string, badWords map[string]struct{}) string {
	words := strings.Split(msg, " ")
	for i, word := range words {
		cleanedWord := strings.Trim(word, ".,!?;:()\"'")
		loweredWord := strings.ToLower(cleanedWord)
		if _, ok := badWords[loweredWord]; ok {
			words[i] = "****"
		}
	}

	final := strings.Join(words, " ")
	return final
}

func (cfg *apiConfig) handlerListChirps(w http.ResponseWriter, r *http.Request){
	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: Listing Chirps", err)
		return
	}

	responseChirps := []Chirp{}

	for _, dbChirp := range dbChirps {
		responseChirps = append(responseChirps, Chirp{
			ID:          dbChirp.ID,
			CreatedAt:   dbChirp.CreatedAt,
			UpdatedAt:   dbChirp.UpdatedAt,
			Body:        dbChirp.Body,
			UserID:      dbChirp.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, responseChirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request){
	strChirpID := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(strChirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse UUID", err)
		return
	}

	dbChirp, err := cfg.db.GetChirpFromID(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "No Chirp found matching that ID", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error: Retrieving Chirp from ID", err)
		return
	}

	respondWithJSON(w, http.StatusOK, Chirp{
			ID:          dbChirp.ID,
			CreatedAt:   dbChirp.CreatedAt,
			UpdatedAt:   dbChirp.UpdatedAt,
			Body:        dbChirp.Body,
			UserID:      dbChirp.UserID,
		})
}
