package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
		log.Fatalf("Error marshalling JSON: %s", err)
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(data)
}
