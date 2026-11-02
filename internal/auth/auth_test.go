package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "  Bearer   test-token  ")

	got, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("GetBearerToken() error = %v", err)
	}
	if got != "test-token" {
		t.Errorf("GetBearerToken() = %q, want %q", got, "test-token")
	}
}

func TestGetBearerTokenRejectsMissingOrInvalidHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "missing"},
		{name: "missing token", header: "Bearer"},
		{name: "wrong scheme", header: "Basic credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			if tt.header != "" {
				headers.Set("Authorization", tt.header)
			}

			if _, err := GetBearerToken(headers); err == nil {
				t.Fatal("GetBearerToken() error = nil, want an error")
			}
		})
	}
}

func TestMakeJWTAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}
	if token == "" {
		t.Fatal("MakeJWT() returned an empty token")
	}

	gotUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}
	if gotUserID != userID {
		t.Errorf("ValidateJWT() user ID = %v, want %v", gotUserID, userID)
	}
}

func TestValidateJWTRejectsExpiredToken(t *testing.T) {
	token, err := MakeJWT(uuid.New(), "test-secret", -time.Second)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	if _, err := ValidateJWT(token, "test-secret"); err == nil {
		t.Fatal("ValidateJWT() accepted an expired token")
	}
}

func TestValidateJWTRejectsWrongSecret(t *testing.T) {
	token, err := MakeJWT(uuid.New(), "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	if _, err := ValidateJWT(token, "wrong-secret"); err == nil {
		t.Fatal("ValidateJWT() accepted a token signed with a different secret")
	}
}
