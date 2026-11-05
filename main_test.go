package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/iitung-dev/chirpy/internal/database"
	"github.com/joho/godotenv"
)

func TestValidateChirpHandler(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		expectedStatus int
		expectedOutput string
	}{
		{
			name:           "Replaces lowercase profane word",
			inputBody:      `{"body": "This is a kerfuffle opinion"}`,
			expectedStatus: http.StatusOK,
			expectedOutput: `{"cleaned_body":"This is a **** opinion"}`,
		},
		{
			name:           "Replaces uppercase profane word",
			inputBody:      `{"body": "DANGER FORNAX ahead"}`,
			expectedStatus: http.StatusOK,
			expectedOutput: `{"cleaned_body":"DANGER **** ahead"}`,
		},
		{
			name:           "Ignores words with punctuation attached",
			inputBody:      `{"body": "Sharbert! That shouldn't change."}`,
			expectedStatus: http.StatusOK,
			expectedOutput: `{"cleaned_body":"Sharbert! That shouldn't change."}`,
		},
		{
			name:           "Leaves clean text unchanged",
			inputBody:      `{"body": "Hello world, everything is fine."}`,
			expectedStatus: http.StatusOK,
			expectedOutput: `{"cleaned_body":"Hello world, everything is fine."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/validate_chirp", bytes.NewBuffer([]byte(tt.inputBody)))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			
			// Replace `handlerValidateChirp` with your actual handler function name
			handlerValidateChirp(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("expected status %d; got %d", tt.expectedStatus, recorder.Code)
			}

			// Clean up whitespace/newlines for reliable string comparison
			var gotBuffer, expectedBuffer bytes.Buffer
			_ = json.Compact(&gotBuffer, recorder.Body.Bytes())
			_ = json.Compact(&expectedBuffer, []byte(tt.expectedOutput))

			if gotBuffer.String() != expectedBuffer.String() {
				t.Errorf("expected body %s; got %s", expectedBuffer.String(), gotBuffer.String())
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	// 1. Load environment variables from .env
	_ = godotenv.Load(".env")

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL is not set in the environment or .env file, skipping integration test.")
	}

	// 2. Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 3. Verify connection with Ping
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 4. Initialize SQLC queries
	dbQueries := database.New(db)
	ctx := context.Background()

	// 5. Test creating a user
	uniqueEmail := "test_user_" + uuid.New().String()[:8] + "@example.com"
	
	user, err := dbQueries.CreateUser(ctx, uniqueEmail)
	if err != nil {
		t.Fatalf("Failed to create user via SQLC: %v", err)
	}

	// 6. Assertions
	if user.Email != uniqueEmail {
		t.Errorf("Expected email %s, got %s", uniqueEmail, user.Email)
	}

	if user.ID == uuid.Nil {
		t.Errorf("Expected a valid UUID for user ID, got nil")
	}

	t.Logf("Success! Created user ID: %s with email: %s", user.ID, user.Email)
}