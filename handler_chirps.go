package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Yishen1011/http_servers/internal/auth"
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
    }

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
    userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	type response struct {
		Chirp
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err = decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
    }

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
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
		UserID: userID,
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
	authorID, err := authorIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
		return
	}
	
	var dbChirps []database.Chirp

	if authorID != uuid.Nil {
		dbChirps, err = cfg.db.GetChirpsFromUserID(r.Context(), authorID)
	} else {
		dbChirps, err = cfg.db.GetChirps(r.Context())
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
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

func authorIDFromRequest(r *http.Request) (uuid.UUID, error) {
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString == "" {
		return uuid.Nil, nil
	}
	authorID, err := uuid.Parse(authorIDString)
	if err != nil {
		return uuid.Nil, err
	}
	return authorID, nil
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
		respondWithError(w, http.StatusInternalServerError, "Retrieving Chirp from ID", err)
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

func (cfg *apiConfig) handlerDeleteChirpFromID(w http.ResponseWriter, r *http.Request){
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	
	strChirpID := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(strChirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse UUID", err)
		return
	}

	chirp, err := cfg.db.GetChirpFromID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "ChirpID is not found in database", err)
		return
	}

	if userID != chirp.UserID {
		respondWithError(w, http.StatusForbidden, "UserID from token is different from API", err)
		return
	}

	err = cfg.db.DeleteChirpFromID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "ChirpID is not found in database", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
