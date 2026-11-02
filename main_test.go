package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sucamchi/Chirpy/internal/auth"
	"github.com/sucamchi/Chirpy/internal/database"
)

type webhookDB struct {
	rowsAffected int64
	execCalls    int
}

func (db *webhookDB) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	db.execCalls++
	return webhookResult(db.rowsAffected), nil
}

func (*webhookDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, nil
}

func (*webhookDB) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (*webhookDB) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return nil
}

type webhookResult int64

func (webhookResult) LastInsertId() (int64, error) { return 0, nil }
func (result webhookResult) RowsAffected() (int64, error) {
	return int64(result), nil
}

func TestUpdateUserRequiresValidBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing token"},
		{name: "malformed token", authorization: "******"},
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

func TestPolkaWebhookIgnoresOtherEvents(t *testing.T) {
	db := &webhookDB{}
	cfg := &apiConfig{db: database.New(db)}
	req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", strings.NewReader(`{"event":"payment.failed"}`))
	recorder := httptest.NewRecorder()

	cfg.handlePolkaWebhook(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("handlePolkaWebhook() status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("handlePolkaWebhook() body = %q, want empty", recorder.Body.String())
	}
	if db.execCalls != 0 {
		t.Errorf("database called %d times for unrelated event, want 0", db.execCalls)
	}
}

func TestPolkaWebhookUpgradesUser(t *testing.T) {
	tests := []struct {
		name         string
		rowsAffected int64
		wantStatus   int
	}{
		{name: "upgraded", rowsAffected: 1, wantStatus: http.StatusNoContent},
		{name: "user not found", rowsAffected: 0, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &webhookDB{rowsAffected: tt.rowsAffected}
			cfg := &apiConfig{db: database.New(db)}
			body := `{"event":"user.upgraded","data":{"user_id":"` + uuid.NewString() + `"}}`
			req := httptest.NewRequest(http.MethodPost, "/api/polka/webhooks", strings.NewReader(body))
			recorder := httptest.NewRecorder()

			cfg.handlePolkaWebhook(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("handlePolkaWebhook() status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if recorder.Body.Len() != 0 {
				t.Errorf("handlePolkaWebhook() body = %q, want empty", recorder.Body.String())
			}
			if db.execCalls != 1 {
				t.Errorf("database called %d times, want 1", db.execCalls)
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
		{name: "malformed token", authorization: "******"},
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
