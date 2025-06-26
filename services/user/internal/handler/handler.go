package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

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

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("Failed to hash password: %v", err)
		return "", err
	}
	return string(hash), nil
}

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
		otp, err := generateOTP()
		if err != nil {
			logrus.Errorf("Failed to generate OTP: %v", err)
			http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
			return
		}
		phash, err := hashPassword(req.Password)
		if err != nil {
			logrus.Errorf("Failed to hash password: %v", err)
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		uuid := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%d", req.Email, time.Now().UnixNano())))
		expiry := time.Now().Add(10 * time.Minute)
		_, err = db.Exec(`INSERT OR REPLACE INTO pending_registrations (EMAILID, OTP, OTP_EXPIRES_AT, USERPSWD, UNIQUEID, CREATED_AT) VALUES (?, ?, ?, ?, ?, ?)`, req.Email, otp, expiry, phash, uuid, time.Now())
		if err != nil {
			logrus.Errorf("Failed to store registration: %v", err)
			http.Error(w, "Failed to store registration", http.StatusInternalServerError)
			return
		}
		if err := sendOTPEmail(req.Email, otp); err != nil {
			logrus.Errorf("Failed to send OTP email: %v", err)
			http.Error(w, "Failed to send OTP email", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"otp_sent"}`))
	}
}

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
		_, err = db.Exec(`INSERT INTO USERDEFN (EMAILID, UNIQUEID, USERPSWD, CREATEDETTM, LASTUPDDTTM, LASTPSWDCHANGE, ACCTLOCK, ISSIGNEDIN, DEFAULTHOME, FAILEDLOGINS, LANGUAGE_CODE, CURRENCY_CODE) VALUES (?, ?, ?, ?, ?, ?, 0, 0, '', 0, 'ENG', 'INR')`, req.Email, uuid, phash, createdAt, time.Now(), time.Now())
		if err != nil {
			logrus.Errorf("Failed to create user: %v", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		_, _ = db.Exec(`DELETE FROM pending_registrations WHERE EMAILID = ?`, req.Email)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"registered"}`))
	}
}

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
			db.Exec(`UPDATE USERDEFN SET FAILEDLOGINS = FAILEDLOGINS + 1 WHERE EMAILID = ?`, req.Email)
			logrus.Errorf("Invalid email or password")
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		db.Exec(`UPDATE USERDEFN SET FAILEDLOGINS = 0, ISSIGNEDIN = 1, LASTSIGNONDTTM = ? WHERE EMAILID = ?`, time.Now(), req.Email)
		sid, err := r.Cookie("UTL_SESSION")
		var sessionID string
		if err == nil && sid.Value != "" {
			sessionID = sid.Value
		} else {
			sessionID = base64.URLEncoding.EncodeToString([]byte(req.Email + time.Now().String()))
			_, _ = db.Exec(`INSERT OR IGNORE INTO sessions (session_id, user_agent, ip_address, user_email) VALUES (?, ?, ?, ?)`, sessionID, r.UserAgent(), r.RemoteAddr, req.Email)
		}
		_, _ = db.Exec(`UPDATE sessions SET user_email = ?, created_at = ? WHERE session_id = ?`, req.Email, time.Now(), sessionID)
		_, _ = db.Exec(`UPDATE url_mappings SET user_email = ? WHERE session_id = ? AND (user_email IS NULL OR user_email = '')`, req.Email, sessionID)
		setSessionCookie(w, sessionID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"logged_in"}`))
	}
}

func LogoutHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, err := r.Cookie("UTL_SESSION")
		if err != nil || sid.Value == "" {
			logrus.Errorf("No session")
			http.Error(w, "No session", http.StatusUnauthorized)
			return
		}
		var email string
		db.QueryRow(`SELECT user_email FROM sessions WHERE session_id = ?`, sid.Value).Scan(&email)
		if email != "" {
			_, _ = db.Exec(`UPDATE USERDEFN SET ISSIGNEDIN = 0, LASTSIGNOFFDTTM = ? WHERE EMAILID = ?`, time.Now(), email)
		}
		_, _ = db.Exec(`UPDATE sessions SET user_email = NULL WHERE session_id = ?`, sid.Value)
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
