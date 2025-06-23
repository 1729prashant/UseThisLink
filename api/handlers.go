package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"usethislink/internal/analytics"
	"usethislink/internal/mw"
	"usethislink/internal/shortner"

	"crypto/rand"
	"encoding/base64"
	"net/smtp"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

type shortenRequest struct {
	URL string `json:"original_url"`
}

type shortenResponse struct {
	ShortURL string `json:"short_url"`
}

// isReservedIP checks if the IP is in a reserved/special-use range and returns (true, scope string)
func isReservedIP(ipStr string) (bool, string) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, ""
	}
	// IPv4 ranges
	reserved := []struct {
		cidr  string
		scope string
	}{
		{"0.0.0.0/8", "Software"},
		{"10.0.0.0/8", "Private network"},
		{"100.64.0.0/10", "Private network"},
		{"127.0.0.0/8", "Host"},
		{"169.254.0.0/16", "Subnet"},
		{"172.16.0.0/12", "Private network"},
		{"192.0.0.0/24", "Private network"},
		{"192.0.2.0/24", "Documentation"},
		{"192.88.99.0/24", "Internet"},
		{"192.168.0.0/16", "Private network"},
		{"198.18.0.0/15", "Private network"},
		{"198.51.100.0/24", "Documentation"},
		{"203.0.113.0/24", "Documentation"},
		{"224.0.0.0/4", "Internet"},
		{"233.252.0.0/24", "Documentation"},
		{"240.0.0.0/4", "Internet"},
		{"255.255.255.255/32", "Subnet"},
	}
	for _, r := range reserved {
		_, ipnet, err := net.ParseCIDR(r.cidr)
		if err == nil && ipnet.Contains(ip) {
			return true, r.scope
		}
	}
	// IPv6 ranges
	reserved6 := []struct {
		cidr  string
		scope string
	}{
		{"::1/128", "Host"},
		{"::/128", "Software"},
		{"::ffff:0:0/96", "Software"},
		{"::ffff:0:0:0/96", "Software"},
		{"64:ff9b::/96", "The global Internet"},
		{"64:ff9b:1::/48", "Private internets"},
		{"100::/64", "Routing"},
		{"2001::/32", "The global Internet"},
		{"2001:20::/28", "Software"},
		{"2001:db8::/32", "Documentation"},
		{"2002::/16", "The global Internet"},
		{"3fff::/20", "Documentation"},
		{"5f00::/16", "Routing"},
		{"fc00::/7", "Private internets"},
		{"fe80::/10", "Link"},
		{"ff00::/8", "The global Internet"},
	}
	for _, r := range reserved6 {
		_, ipnet, err := net.ParseCIDR(r.cidr)
		if err == nil && ipnet.Contains(ip) {
			return true, r.scope
		}
	}
	return false, ""
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

		// Check if a scheme exists
		hasScheme := strings.Contains(rawURL, "://")

		// If no scheme, default to http://
		if !hasScheme {
			rawURL = "http://" + rawURL
		}

		// Parse the URL to validate domain presence
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || !strings.Contains(parsed.Host, ".") {
			logrus.Errorf("Invalid or incomplete domain: %v", err)
			http.Error(w, "Invalid or incomplete domain", http.StatusBadRequest)
			return
		}

		// Prevent shortening URLs that contain the service's own BASE_URL
		baseURL := os.Getenv("BASE_URL")
		if baseURL != "" && (strings.Contains(rawURL, baseURL) || strings.Contains(parsed.Host, strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://"))) {
			logrus.Warnf("Attempt to shorten a URL containing BASE_URL: %s", rawURL)
			http.Error(w, "You cannot shorten URLs that point to this service.", http.StatusBadRequest)
			return
		}

		sid := r.Context().Value(mw.SessionKey).(string)
		var userEmail string
		cookie, err := r.Cookie("UTL_SESSION")
		if err == nil && cookie.Value != "" {
			_ = db.QueryRow(`SELECT user_email FROM sessions WHERE session_id = ?`, cookie.Value).Scan(&userEmail)
		}
		// Extract IP address only (remove port)
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr // fallback, but should not happen
		}
		// 1-time insert session row (best-effort; ignore error if exists)
		db.Exec(`INSERT OR IGNORE INTO sessions (session_id, user_agent, ip_address) VALUES (?, ?, ?)`, sid, r.UserAgent(), host)

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

		// Get session ID from context
		sid, _ := r.Context().Value(mw.SessionKey).(string)

		// Parse user agent
		deviceInfo := analytics.ParseUserAgent(r.UserAgent())

		// Get IP address
		ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ipAddr = r.RemoteAddr
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddr = strings.Split(forwarded, ",")[0]
		}

		// Get geolocation (non-blocking)
		go func() {
			var geoInfo analytics.GeoInfo
			if reserved, scope := isReservedIP(ipAddr); reserved {
				geoInfo = analytics.GeoInfo{City: "Localhost", Country: scope}
			} else {
				var err error
				geoInfo, err = analytics.GetLocationFromIP(ipAddr)
				if err != nil {
					logrus.Errorf("Failed to get geolocation: %v", err)
					geoInfo = analytics.GeoInfo{City: "Unknown", Country: "Unknown"}
				}
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
		sid := r.Context().Value(mw.SessionKey).(string)
		var userEmail string
		cookie, err := r.Cookie("UTL_SESSION")
		if err == nil && cookie.Value != "" {
			_ = db.QueryRow(`SELECT user_email FROM sessions WHERE session_id = ?`, cookie.Value).Scan(&userEmail)
		}
		baseURL := os.Getenv("BASE_URL")
		var rows *sql.Rows
		if userEmail != "" {
			// Show all URLs for this user_email or for this session_id (pre-login)
			rows, err = db.Query(`
				SELECT original_url, short_url, expiry_date, is_logged_in, user_email
				FROM url_mappings WHERE user_email = ? OR session_id = ? ORDER BY created_at DESC
			`, userEmail, sid)
		} else {
			// Not logged in: show only session_id URLs
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

// QRCodeHandler serves a QR code PNG for a given URL (via ?data=...)
func QRCodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := r.URL.Query().Get("data")
		if data == "" {
			logrus.Errorf("Missing data parameter")
			http.Error(w, "Missing data parameter", http.StatusBadRequest)
			return
		}
		qr, err := qrcode.New(data, qrcode.Medium)
		if err != nil {
			logrus.Errorf("Failed to generate QR code: %v", err)
			http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		img := qr.Image(150)
		png.Encode(w, img)
	}
}

// generateOTP generates a 6-digit random OTP
func generateOTP() (string, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		logrus.Errorf("Failed to generate OTP: %v", err)
		return "", err
	}
	otp := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	otp = otp % 1000000
	if otp < 0 {
		otp = -otp
	}
	return fmt.Sprintf("%06d", otp), nil
}

// hashPassword hashes the password using bcrypt
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("Failed to hash password: %v", err)
		return "", err
	}
	return string(hash), nil
}

// sendOTPEmail sends an OTP to the user's email using SMTP
func sendOTPEmail(to, otp string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	from := smtpUser
	msg := []byte("Subject: Your OTP for UseThisLink Registration\r\n" +
		"To: " + to + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		fmt.Sprintf("Your OTP for UseThisLink registration is: %s\nThis OTP is valid for 10 minutes.", otp))
	var auth smtp.Auth
	if smtpUser != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	} else {
		auth = nil
	}
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		logrus.Errorf("Failed to send OTP email via smtp.SendMail: %v", err)
		return err
	}
	return nil
}

// RegisterHandler handles user registration and sends OTP
func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type reqBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		var req reqBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
			logrus.Errorf("Invalid request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		// Check if user already exists
		var exists int
		err := db.QueryRow("SELECT COUNT(1) FROM USERDEFN WHERE EMAILID = ?", req.Email).Scan(&exists)
		if err != nil {
			logrus.Errorf("DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if exists > 0 {
			logrus.Errorf("User already exists")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "An account with this email already exists. Please log in or use a different email."})
			return
		}
		// Generate OTP
		otp, err := generateOTP()
		if err != nil {
			logrus.Errorf("Failed to generate OTP: %v", err)
			http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
			return
		}
		// Hash password
		phash, err := hashPassword(req.Password)
		if err != nil {
			logrus.Errorf("Failed to hash password: %v", err)
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		// Generate UUID
		uuid := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%d", req.Email, time.Now().UnixNano())))
		// Store in pending_registrations
		expiry := time.Now().Add(10 * time.Minute)
		_, err = db.Exec(`INSERT OR REPLACE INTO pending_registrations (EMAILID, OTP, OTP_EXPIRES_AT, USERPSWD, UNIQUEID, CREATED_AT) VALUES (?, ?, ?, ?, ?, ?)`, req.Email, otp, expiry, phash, uuid, time.Now())
		if err != nil {
			logrus.Errorf("Failed to store registration: %v", err)
			http.Error(w, "Failed to store registration", http.StatusInternalServerError)
			return
		}
		// Send OTP email
		if err := sendOTPEmail(req.Email, otp); err != nil {
			logrus.Errorf("Failed to send OTP email: %v", err)
			http.Error(w, "Failed to send OTP email", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"otp_sent"}`))
	}
}

// VerifyOTPHandler handles OTP verification and user creation
func VerifyOTPHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type reqBody struct {
			Email string `json:"email"`
			OTP   string `json:"otp"`
		}
		var req reqBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.OTP == "" {
			logrus.Errorf("Invalid request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		// Get pending registration
		var dbOTP, phash, uuid string
		var otpExpiry, createdAt time.Time
		err := db.QueryRow(`SELECT OTP, OTP_EXPIRES_AT, USERPSWD, UNIQUEID, CREATED_AT FROM pending_registrations WHERE EMAILID = ?`, req.Email).Scan(&dbOTP, &otpExpiry, &phash, &uuid, &createdAt)
		if err == sql.ErrNoRows {
			logrus.Errorf("No pending registration")
			http.Error(w, "No pending registration", http.StatusNotFound)
			return
		} else if err != nil {
			logrus.Errorf("DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if time.Now().After(otpExpiry) {
			logrus.Errorf("OTP expired")
			http.Error(w, "OTP expired", http.StatusUnauthorized)
			return
		}
		if req.OTP != dbOTP {
			logrus.Errorf("Invalid OTP")
			http.Error(w, "Invalid OTP", http.StatusUnauthorized)
			return
		}
		// Insert into USERDEFN
		_, err = db.Exec(`INSERT INTO USERDEFN (EMAILID, UNIQUEID, USERPSWD, CREATEDETTM, LASTUPDDTTM, LASTPSWDCHANGE, ACCTLOCK, ISSIGNEDIN, DEFAULTHOME, FAILEDLOGINS, LANGUAGE_CODE, CURRENCY_CODE) VALUES (?, ?, ?, ?, ?, ?, 0, 0, '', 0, 'ENG', 'INR')`, req.Email, uuid, phash, createdAt, time.Now(), time.Now())
		if err != nil {
			logrus.Errorf("Failed to create user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		// Delete from pending_registrations
		_, _ = db.Exec(`DELETE FROM pending_registrations WHERE EMAILID = ?`, req.Email)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"registered"}`))
	}
}

// Helper to set secure session cookie
func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "UTL_SESSION",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60 * 60 * 24 * 7, // 1 week
	})
}

// LoginHandler handles secure login
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type reqBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		var req reqBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
			logrus.Errorf("Invalid request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		// Get user info
		var hash string
		var acctLock, failedLogins int
		var uniqueID string
		var lastSignOn, lastSignOff, lastPwdChange, createDetTm, lastUpdTm sql.NullTime
		err := db.QueryRow(`SELECT USERPSWD, ACCTLOCK, FAILEDLOGINS, UNIQUEID, LASTSIGNONDTTM, LASTSIGNOFFDTTM, LASTPSWDCHANGE, CREATEDETTM, LASTUPDDTTM FROM USERDEFN WHERE EMAILID = ?`, req.Email).Scan(&hash, &acctLock, &failedLogins, &uniqueID, &lastSignOn, &lastSignOff, &lastPwdChange, &createDetTm, &lastUpdTm)
		if err == sql.ErrNoRows {
			logrus.Errorf("Invalid email or password")
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		} else if err != nil {
			logrus.Errorf("DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if acctLock != 0 {
			logrus.Errorf("Account is locked")
			http.Error(w, "Account is locked", http.StatusForbidden)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			// Increment FAILEDLOGINS
			db.Exec(`UPDATE USERDEFN SET FAILEDLOGINS = FAILEDLOGINS + 1 WHERE EMAILID = ?`, req.Email)
			logrus.Errorf("Invalid email or password")
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		// Reset FAILEDLOGINS, set ISSIGNEDIN, update LASTSIGNONDTTM
		db.Exec(`UPDATE USERDEFN SET FAILEDLOGINS = 0, ISSIGNEDIN = 1, LASTSIGNONDTTM = ? WHERE EMAILID = ?`, time.Now(), req.Email)
		// Get or create session
		sid, err := r.Cookie("UTL_SESSION")
		var sessionID string
		if err == nil && sid.Value != "" {
			sessionID = sid.Value
		} else {
			sessionID = base64.URLEncoding.EncodeToString([]byte(req.Email + time.Now().String()))
			_, _ = db.Exec(`INSERT OR IGNORE INTO sessions (session_id, user_agent, ip_address, user_email) VALUES (?, ?, ?, ?)`, sessionID, r.UserAgent(), r.RemoteAddr, req.Email)
		}
		// Update session to associate with user
		_, _ = db.Exec(`UPDATE sessions SET user_email = ?, created_at = ? WHERE session_id = ?`, req.Email, time.Now(), sessionID)

		// Migrate all url_mappings for this session_id with empty user_email to this user_email
		_, _ = db.Exec(`UPDATE url_mappings SET user_email = ? WHERE session_id = ? AND (user_email IS NULL OR user_email = '')`, req.Email, sessionID)

		setSessionCookie(w, sessionID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"logged_in"}`))
	}
}

// LogoutHandler logs out the user and clears session
func LogoutHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, err := r.Cookie("UTL_SESSION")
		if err != nil || sid.Value == "" {
			logrus.Errorf("No session")
			http.Error(w, "No session", http.StatusUnauthorized)
			return
		}
		// Get user_email from session
		var email string
		db.QueryRow(`SELECT user_email FROM sessions WHERE session_id = ?`, sid.Value).Scan(&email)
		if email != "" {
			// Set ISSIGNEDIN=0, update LASTSIGNOFFDTTM
			_, _ = db.Exec(`UPDATE USERDEFN SET ISSIGNEDIN = 0, LASTSIGNOFFDTTM = ? WHERE EMAILID = ?`, time.Now(), email)
		}
		// Remove user_email from session
		_, _ = db.Exec(`UPDATE sessions SET user_email = NULL WHERE session_id = ?`, sid.Value)
		// Expire cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "UTL_SESSION",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"logged_out"}`))
	}
}

// AuthMiddleware checks for authenticated session (stub for now)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid, err := r.Cookie("UTL_SESSION")
		if err != nil || sid.Value == "" {
			logrus.Errorf("Unauthorized")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Optionally, check session in DB and user_email is set
		// ...
		next.ServeHTTP(w, r)
	})
}

// SessionStatusHandler returns login status and email
func SessionStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, err := r.Cookie("UTL_SESSION")
		if err != nil || sid.Value == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"logged_in": false}`))
			return
		}
		var email string
		_ = db.QueryRow(`SELECT user_email FROM sessions WHERE session_id = ?`, sid.Value).Scan(&email)
		if email != "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"logged_in": true, "email": "` + email + `"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"logged_in": false}`))
	}
}
