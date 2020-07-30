package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/julienschmidt/httprouter"

	"github.com/rs/cors"

	"github.com/gorilla/schema"
)

const (
	// queryInsertMessage is the query used to insert a message.
	queryInsertMessage = "INSERT INTO messages(name, message) VALUES ($1, $2)"
	// queryGetRecentMessages is the query used to read recent messages.
	queryGetRecentMessages = "SELECT submitted_at, name, message FROM messages ORDER BY submitted_at DESC LIMIT 5"
)

// api is the API being served.
type api struct {
	// database is the underlying database.
	database *sql.DB
}

// messageForm represents a submitted message in a POST request.
type messageForm struct {
	// Name is the name of the message submitter.
	Name string `schema:"name"`
	// Message is the message.
	Message string `schema:"message"`
}

// insertMessage inserts a new message.
func (a *api) insertMessage(w http.ResponseWriter, r *http.Request) {
	// Parse the request's form data.
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Error: unable to parse form data: %v", err), http.StatusBadRequest)
		return
	}

	// Decode form data.
	var form messageForm
	decoder := schema.NewDecoder()
	if err := decoder.Decode(&form, r.PostForm); err != nil {
		http.Error(w, fmt.Sprintf("Error: unable to decode form data: %v", err), http.StatusBadRequest)
		return
	}

	// Validate form data.
	if form.Name == "" {
		http.Error(w, "Error: empty submitter name", http.StatusBadRequest)
		return
	} else if form.Message == "" {
		http.Error(w, "Error: empty message", http.StatusBadRequest)
		return
	}

	// Insert the message to the database.
	if _, err := a.database.ExecContext(r.Context(), queryInsertMessage, form.Name, form.Message); err != nil {
		http.Error(w, fmt.Sprintf("Error: unable to record message: %v", err), http.StatusInternalServerError)
		return
	}

	// Success.
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "Message successfully recorded!")
}

// message represents a message returned in a JSON array response.
type message struct {
	// Time is the time that the message was submitted.
	Time time.Time `json:"time"`
	// Name is the name of the message submitter.
	Name string `json:"name"`
	// Message is the message.
	Message string `json:"message"`
}

// getRecentMessages returns a list of the most recent messages.
func (a *api) getRecentMessages(w http.ResponseWriter, r *http.Request) {
	// Perform the query and defer closure of the iterator.
	rows, err := a.database.QueryContext(r.Context(), queryGetRecentMessages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: unable to perform database query: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Create the results. We always start with an empty but non-nil slice
	// to ensure the resulting JSON object is an empty array and not null if
	// there are no results.
	results := make([]message, 0)
	for rows.Next() {
		var row message
		if err := rows.Scan(&row.Time, &row.Name, &row.Message); err != nil {
			http.Error(w, fmt.Sprintf("Error: unable to load message: %v", err), http.StatusInternalServerError)
			return
		}
		results = append(results, row)
	}

	// Check that iteration didn't fail.
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error: Processing messages failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the content type to JSON. We have to do this manually since Go's
	// built-in content sniffing doesn't know about JSON (golang/go#10630).
	w.Header().Set("Content-Type", "application/json")

	// Encode the results as JSON.
	json.NewEncoder(w).Encode(results)
}

// serve is the primary entry point.
func serve() error {
	// Grab and validate configuration parameters from the environment.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("invalid or unspecified database URL")
	}
	bind := os.Getenv("SERVER_BIND")
	if bind == "" {
		return errors.New("invalid or unspecified server bind")
	}
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		return errors.New("invalid or unspecified CORS origin")
	}

	// Connect to the database and defer its closure.
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer database.Close()

	// Create an API instance.
	api := &api{database: database}

	// Create the request router.
	router := httprouter.New()

	// Set up handlers.
	router.HandlerFunc(http.MethodPost, "/api/messages", api.insertMessage)
	router.HandlerFunc(http.MethodGet, "/api/messages", api.getRecentMessages)

	// Take the router as our root handler.
	handler := http.Handler(router)

	// Set up CORS middleware to allow cross-origin requests.
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{corsOrigin},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
	})
	handler = corsMiddleware.Handler(handler)

	// Serve files.
	return http.ListenAndServe(bind, handler)
}

func main() {
	// Run the server and log any error.
	log.Fatal(serve())
}
