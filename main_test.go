package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sucamchi/Chirpy/internal/auth"
)

func TestUpdateUserRequiresValidBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing token"},
		{name: "malformed token", authorization: "Bearer not-a-jwt"},
		{name: "malformed authorization header", authorization: "Basic credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &apiConfig{jwtSecret: "test-secret"}
			req := httptest.NewRequest(http.MethodPut, "/api/users", strings.NewReader(`{"email":"new@example.com","password":"new-password"}`))
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			recorder := httptest.NewRecorder()

			cfg.updateUser(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("updateUser() status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestDeleteChirpRequiresValidToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing token"},
		{name: "malformed token", authorization: "Bearer invalid"},
		{name: "malformed authorization header", authorization: "Basic credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &apiConfig{jwtSecret: "test-secret"}
			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirp)
			req := httptest.NewRequest(http.MethodDelete, "/api/chirps/"+uuid.NewString(), nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("DELETE /api/chirps status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestDeleteChirpRejectsMalformedID(t *testing.T) {
	const secret = "test-secret"
	token, err := auth.MakeJWT(uuid.New(), secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	cfg := &apiConfig{jwtSecret: secret}
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirp)
	req := httptest.NewRequest(http.MethodDelete, "/api/chirps/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("DELETE /api/chirps status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
