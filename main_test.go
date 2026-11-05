package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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