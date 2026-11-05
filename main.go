package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/iitung-dev/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries *database.Queries
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to start connection to database: %v", err)
	}
	dbQueries := database.New(db)

	const port = "8080"
	const filepathRoot = "."

	apiCfg := &apiConfig{
		dbQueries: dbQueries,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerFileServerHitCount)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	http_server := &http.Server {
		Addr: ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatalf("Server failed to start: %v\n", http_server.ListenAndServe())
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerFileServerHitCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	count := fmt.Sprintf(`
	<html>
  		<body>
    		<h1>Welcome, Chirpy Admin</h1>
    		<p>Chirpy has been visited %d times!</p>
  		</body>
	</html>`, cfg.fileServerHits.Load())
	w.WriteHeader(200)
	w.Write([]byte(count))
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type CleanedChirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

	params := parameters{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode params.")
		return 
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := getCleanedBody(params.Body)

	response := CleanedChirpResponse{
		CleanedBody: cleanedBody,
	}
	
	respondWithJSON(w, http.StatusOK, response)

}

func getCleanedBody(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		lower := strings.ToLower(string(word))
		for _, profane := range []string{"kerfuffle", "sharbert", "fornax"}{
			if profane == lower {
				words[i] = "****"
			}
		}
	}

	return strings.Join(words, " ")
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	resp, err := json.Marshal(&payload)
	if err != nil {
		log.Printf("Error marshalling chirp validation: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(resp)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code >= 500 {
		log.Printf("Server error: %v", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{Error: msg})
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	cfg.fileServerHits.Store(0)
	fmt.Println("File server hit count reset!")
}