package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/sucamchi/Chirpy/internal/database"
)

const maxChirpLength = 140

var profaneWords = map[string]bool{
	"kerfuffle": true,
	"sharbert":  true,
	"fornax":    true,
}

func cleanChirpBody(body string) string {
	words := strings.Fields(body)
	for i, word := range words {
		if profaneWords[strings.ToLower(word)] {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	userID, ok := cfg.authenticate(w, r)
	if !ok {
		return
	}

	var params struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp body exceeds 140 characters")
		return
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanChirpBody(params.Body),
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}
	respondWithJSON(w, http.StatusCreated, newChirpResponse(chirp))
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	sortOrder := query.Get("sort")
	if sortOrder == "" {
		sortOrder = "asc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		respondWithError(w, http.StatusBadRequest, "Invalid sort")
		return
	}

	var (
		chirps []database.Chirp
		err    error
	)
	if authorIDs, hasAuthorID := query["author_id"]; !hasAuthorID {
		chirps, err = cfg.db.GetChirps(r.Context())
	} else {
		var authorID uuid.UUID
		if len(authorIDs) == 1 {
			authorID, err = uuid.Parse(authorIDs[0])
		} else {
			err = errors.New("multiple author_id values")
		}
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author_id")
			return
		}
		chirps, err = cfg.db.GetChirpsByAuthor(r.Context(), authorID)
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
		return
	}

	if sortOrder == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
	}

	responses := make([]chirpResponse, 0, len(chirps))
	for _, chirp := range chirps {
		responses = append(responses, newChirpResponse(chirp))
	}
	respondWithJSON(w, http.StatusOK, responses)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to get chirp")
		return
	}
	respondWithJSON(w, http.StatusOK, newChirpResponse(chirp))
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	userID, ok := cfg.authenticate(w, r)
	if !ok {
		return
	}

	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to get chirp")
		return
	}
	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You are not the author of this chirp")
		return
	}

	rowsAffected, err := cfg.db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     chirpID,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
		return
	}
	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
