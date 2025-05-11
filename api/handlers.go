package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"usethislink/internal/shortner"

	"github.com/gorilla/mux"
)

type shortenRequest struct {
	URL string `json:"original_url"`
}

type shortenResponse struct {
	ShortURL string `json:"short_url"`
}

func ShortenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req shortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		rawURL := req.URL

		// Check if a scheme exists
		hasScheme := strings.Contains(rawURL, "://")

		// If no scheme, default to http://
		if !hasScheme {
			rawURL = "http://" + rawURL
		}

		// Parse the URL to validate domain presence
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || !strings.Contains(parsed.Host, ".") {
			http.Error(w, "Invalid or incomplete domain", http.StatusBadRequest)
			return
		}

		shortURL, err := shortner.StoreURL(db, rawURL)
		if err != nil {
			http.Error(w, "Could not generate short URL", http.StatusInternalServerError)
			return
		}

		resp := shortenResponse{
			ShortURL: shortURL,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func RedirectHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortcode := mux.Vars(r)["shortcode"]

		var longURL string
		err := db.QueryRowContext(context.Background(),
			`SELECT original_url FROM url_mappings WHERE short_url = ?`,
			shortcode).Scan(&longURL)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Increment visits (in a separate query)
		_, _ = db.ExecContext(context.Background(),
			`UPDATE url_mappings SET visits = visits + 1 WHERE short_url = ?`,
			shortcode)

		http.Redirect(w, r, longURL, http.StatusFound)
	}
}

type statsResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	Visits      int    `json:"visits"`
	CreatedAt   string `json:"created_at"`
}

func StatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortcode := mux.Vars(r)["shortcode"]

		var resp statsResponse
		err := db.QueryRowContext(context.Background(),
			`SELECT short_url, original_url, visits, created_at
			 FROM url_mappings WHERE short_url = ?`,
			shortcode).Scan(&resp.ShortURL, &resp.OriginalURL, &resp.Visits, &resp.CreatedAt)

		if err != nil {
			http.NotFound(w, r)
			return
		}
		resp.ShortURL = os.Getenv("BASE_URL") + "/" + resp.ShortURL

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
