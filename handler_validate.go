package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handlerValidateChrip(w http.ResponseWriter, r *http.Request){
    type parameters struct {
        Body string `json:"body"`
    }

	type returnVals struct {
		Cleaned_body string `json:"cleaned_body"`
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

	respondWithJSON(w, http.StatusOK, returnVals{
		Cleaned_body: cleaned,
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
