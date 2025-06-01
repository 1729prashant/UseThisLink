package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"usethislink/internal/analytics"
	"usethislink/internal/mw"
	"usethislink/internal/shortner"

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

		sid := r.Context().Value(mw.SessionKey).(string)
		// 1-time insert session row (best-effort; ignore error if exists)
		db.Exec(`INSERT OR IGNORE INTO sessions (session_id, user_agent, ip_address) VALUES (?, ?, ?)`, sid, r.UserAgent(), r.RemoteAddr)

		shortURL, err := shortner.StoreURL(db, sid, rawURL)
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

		// Get session ID from context
		sid, _ := r.Context().Value(mw.SessionKey).(string)

		// Parse user agent
		deviceInfo := analytics.ParseUserAgent(r.UserAgent())

		// Get IP address
		ipAddr := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddr = strings.Split(forwarded, ",")[0]
		}

		// Get geolocation (non-blocking)
		go func() {
			geoInfo, err := analytics.GetLocationFromIP(ipAddr)
			if err != nil {
				logrus.Errorf("Failed to get geolocation: %v", err)
				geoInfo = analytics.GeoInfo{City: "Unknown", Country: "Unknown"}
			}

			// Skip analytics for bots
			if deviceInfo.Bot {
				return
			}

			// Insert analytics record
			_, err = db.ExecContext(context.Background(),
				`INSERT INTO url_access_logs 
				(short_url, session_id, ip_address, user_agent, referrer, visit_type,
				city, country, browser, device, operating_system)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				shortcode, sid, ipAddr, r.UserAgent(), r.Referer(), "redirect",
				geoInfo.City, geoInfo.Country,
				fmt.Sprintf("%s %s", deviceInfo.Browser, deviceInfo.Version), // Include version
				deviceInfo.Device,
				deviceInfo.OperatingSystem)

			if err != nil {
				logrus.Errorf("Failed to insert access log: %v", err)
			}

			// Update link analytics
			_, err = db.ExecContext(context.Background(),
				`INSERT INTO link_analytics 
				(short_url, total_visits, unique_visitors, redirect_count,
				country_counts, browser_counts, device_counts)
				VALUES (?, 1, 1, 1, ?, ?, ?)
				ON CONFLICT(short_url) DO UPDATE SET
				total_visits = total_visits + 1,
				redirect_count = redirect_count + 1,
				country_counts = json_patch(country_counts, json_object(?, '1')),
				browser_counts = json_patch(browser_counts, json_object(?, '1')),
				device_counts = json_patch(device_counts, json_object(?, '1'))`,
				shortcode,
				fmt.Sprintf(`{"%s": 1}`, geoInfo.Country),
				fmt.Sprintf(`{"%s %s": 1}`, deviceInfo.Browser, deviceInfo.Version), // Include version
				fmt.Sprintf(`{"%s": 1}`, deviceInfo.Device),
				geoInfo.Country,
				fmt.Sprintf("%s %s", deviceInfo.Browser, deviceInfo.Version), // Include version
				deviceInfo.Device)

			if err != nil {
				logrus.Errorf("Failed to update analytics: %v", err)
			}
		}()

		// Only increment visits for non-bots
		if !deviceInfo.Bot {
			_, _ = db.ExecContext(context.Background(),
				`UPDATE url_mappings SET visits = visits + 1 WHERE short_url = ?`,
				shortcode)
		}

		http.Redirect(w, r, longURL, http.StatusFound)
	}
}

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
		sid := r.Context().Value(mw.SessionKey).(string)
		baseURL := os.Getenv("BASE_URL")
		rows, err := db.Query(`
			SELECT original_url, short_url, expiry_date, is_logged_in, user_email
			FROM url_mappings WHERE session_id = ? ORDER BY created_at DESC
		`, sid)
		if err != nil {
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
