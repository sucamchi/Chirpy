package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
