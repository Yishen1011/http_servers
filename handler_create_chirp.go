package main

import (
	"encoding/json"
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
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
    }

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
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
		respondWithError(w, http.StatusInternalServerError, "User can't be created")
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
