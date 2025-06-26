package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type statsResponse struct {
	ShortURL       string `json:"short_url"`
	OriginalURL    string `json:"original_url"`
	TotalVisits    int    `json:"total_visits"`
	UniqueVisitors int    `json:"unique_visitors"`
	RedirectCount  int    `json:"redirect_count"`
	PreviewCount   int    `json:"preview_count"`
	CreatedAt      string `json:"created_at"`
	CountryStats   string `json:"country_stats,omitempty"`
	BrowserStats   string `json:"browser_stats,omitempty"`
	DeviceStats    string `json:"device_stats,omitempty"`
}

func StatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortcode := mux.Vars(r)["shortcode"]
		var resp statsResponse
		err := db.QueryRowContext(context.Background(),
			`SELECT 
				u.short_url, u.original_url, u.visits, u.created_at,
				COALESCE(a.total_visits, 0) as total_visits,
				COALESCE(a.unique_visitors, 0) as unique_visitors,
				COALESCE(a.redirect_count, 0) as redirect_count,
				COALESCE(a.preview_count, 0) as preview_count,
				COALESCE(a.country_counts, '{}') as country_counts,
				COALESCE(a.browser_counts, '{}') as browser_counts,
				COALESCE(a.device_counts, '{}') as device_counts
			FROM url_mappings u 
			LEFT JOIN link_analytics a ON u.short_url = a.short_url
			WHERE u.short_url = ?`,
			shortcode).Scan(
			&resp.ShortURL,
			&resp.OriginalURL,
			&resp.TotalVisits,
			&resp.CreatedAt,
			&resp.TotalVisits,
			&resp.UniqueVisitors,
			&resp.RedirectCount,
			&resp.PreviewCount,
			&resp.CountryStats,
			&resp.BrowserStats,
			&resp.DeviceStats,
		)
		if err != nil {
			logrus.Errorf("Failed to fetch stats: %v", err)
			http.NotFound(w, r)
			return
		}
		resp.ShortURL = os.Getenv("BASE_URL") + "/" + resp.ShortURL
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

type urlHistoryResponse struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	ExpiryDate  string `json:"expiry_date"`
	IsLoggedIn  bool   `json:"is_logged_in"`
	UserEmail   string `json:"user_email"`
}

func HistoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := r.Header.Get("X-Session-ID")
		userEmail := r.Header.Get("X-User-Email")
		baseURL := os.Getenv("BASE_URL")
		var rows *sql.Rows
		var err error
		if userEmail != "" {
			rows, err = db.Query(`
				SELECT original_url, short_url, expiry_date, is_logged_in, user_email
				FROM url_mappings WHERE user_email = ? OR session_id = ? ORDER BY created_at DESC
			`, userEmail, sid)
		} else {
			rows, err = db.Query(`
				SELECT original_url, short_url, expiry_date, is_logged_in, user_email
				FROM url_mappings WHERE session_id = ? ORDER BY created_at DESC
			`, sid)
		}
		if err != nil {
			logrus.Errorf("Failed to fetch history: %v", err)
			http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var history []urlHistoryResponse
		for rows.Next() {
			var h urlHistoryResponse
			err := rows.Scan(&h.OriginalURL, &h.ShortURL, &h.ExpiryDate, &h.IsLoggedIn, &h.UserEmail)
			if err == nil {
				if baseURL != "" && h.ShortURL != "" {
					h.ShortURL = baseURL + "/" + h.ShortURL
				}
				history = append(history, h)
			}
		}
		if history == nil {
			history = []urlHistoryResponse{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}
