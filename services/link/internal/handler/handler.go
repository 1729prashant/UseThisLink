package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"usethislink/services/link/internal/shortner"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
			logrus.Errorf("Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		rawURL := req.URL
		hasScheme := strings.Contains(rawURL, "://")
		if !hasScheme {
			rawURL = "http://" + rawURL
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || !strings.Contains(parsed.Host, ".") {
			logrus.Errorf("Invalid or incomplete domain: %v", err)
			http.Error(w, "Invalid or incomplete domain", http.StatusBadRequest)
			return
		}
		baseURL := os.Getenv("BASE_URL")
		if baseURL != "" && (strings.Contains(rawURL, baseURL) || strings.Contains(parsed.Host, strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://"))) {
			logrus.Warnf("Attempt to shorten a URL containing BASE_URL: %s", rawURL)
			http.Error(w, "You cannot shorten URLs that point to this service.", http.StatusBadRequest)
			return
		}
		// TODO: session/user extraction for distributed context
		sid := r.Header.Get("X-Session-ID")
		userEmail := r.Header.Get("X-User-Email")
		shortURL, err := shortner.StoreURL(db, sid, userEmail, rawURL)
		if err != nil {
			logrus.Errorf("Failed to generate short URL: %v", err)
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
			logrus.Errorf("Failed to fetch original URL: %v", err)
			http.NotFound(w, r)
			return
		}
		// Send analytics event to Analytics service (async)
		go func() {
			analyticsURL := os.Getenv("ANALYTICS_SERVICE_URL")
			if analyticsURL == "" {
				analyticsURL = "http://analytics:8082"
			}
			payload := map[string]interface{}{
				"short_url":  shortcode,
				"session_id": r.Header.Get("X-Session-ID"),
				"user_email": r.Header.Get("X-User-Email"),
				"ip_address": r.RemoteAddr,
				"user_agent": r.UserAgent(),
				"referrer":   r.Referer(),
				"event":      "redirect",
			}
			b, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", analyticsURL+"/log", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			// Pass session/user headers
			req.Header.Set("X-Session-ID", r.Header.Get("X-Session-ID"))
			req.Header.Set("X-User-Email", r.Header.Get("X-User-Email"))
			_, _ = http.DefaultClient.Do(req)
		}()
		http.Redirect(w, r, longURL, http.StatusFound)
	}
}
