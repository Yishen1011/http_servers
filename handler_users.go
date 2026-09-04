package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/Yishen1011/http_servers/internal/auth"
	"github.com/Yishen1011/http_servers/internal/database"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request){
    type parameters struct {
		Password string `json:"password"`
        Email string `json:"email"`
    }

	type response struct {
		User
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
    }

	hashedPW, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Password can't be hashed", err)
		return
	}

	args := database.CreateUserParams{
		HashedPassword:  hashedPW,
		Email:           params.Email,
	}

	user, err := cfg.db.CreateUser(r.Context(), args)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "User can't be created", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		User: User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request){
    type parameters struct {
		Password         string `json:"password"`
        Email            string `json:"email"`
    }

	type response struct {
		User
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
    }

	dbUser, err := cfg.db.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusUnauthorized, "No user found matching that email", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Retrieving User from email", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Checking matching password in database", err)
		return
	}
	if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect password for this email", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create access JWT", err)
		return
	}

	args := database.CreateRefreshTokenParams{
		Token:     auth.MakeRefreshToken(),
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	}

	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), args)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Refresh Token can't be created", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:          dbUser.ID,
			CreatedAt:   dbUser.CreatedAt,
			UpdatedAt:   dbUser.UpdatedAt,
			Email:       dbUser.Email,
			IsChirpyRed: dbUser.IsChirpyRed,
		},
		Token: accessToken,
		RefreshToken: refreshToken.Token,
	})
}

func (cfg *apiConfig) handlerUpdateUserPW(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password         string `json:"password"`
        Email            string `json:"email"`
    }

	type response struct {
		User
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

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err = decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
    }

	hashedPW, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Password can't be hashed", err)
		return
	}
	args := database.UpdateUserPasswordParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: hashedPW,
	}

	user, err := cfg.db.UpdateUserPassword(r.Context(), args)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "User password can't be updated", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	})
}

func (cfg *apiConfig) handlerUpdateUserChirpyRed(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		UserID string `json:"user_id"`
	}

	type parameters struct {
		Event string `json:"event"`
        Data  Data   `json:"data"`
    }

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find ApiKey", err)
		return
	}

	if apiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "API key is invalid", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err = decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
    }

	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id", err)
		return
	}

	args := database.UpdateChirpyRedFromIDParams{
		ID:          userID,
		IsChirpyRed: true,
	}

	_, err = cfg.db.UpdateChirpyRedFromID(r.Context(), args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "User not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "User can't be updated", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
