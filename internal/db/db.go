package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// url_mappings - shortened urls
// sessions - who is visiting (one row per browser cookie)
// url_access_logs - every redirect or preview hit (internal analytics)
// link_analytics - cached aggregate stats per short_url
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS url_mappings (
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL,
			session_id TEXT,
			user_email TEXT,
			visits INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expiry_date DATETIME,
			is_logged_in BOOLEAN DEFAULT 0,
			PRIMARY KEY (short_url, session_id, user_email)
		);

		CREATE TABLE IF NOT EXISTS sessions (
			session_id  TEXT PRIMARY KEY,
			user_agent  TEXT,
			ip_address  TEXT,
			user_email  TEXT,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS USERDEFN (
			EMAILID TEXT PRIMARY KEY NOT NULL,
			UNIQUEID TEXT NOT NULL,
			FULLNAMEDESC TEXT DEFAULT '',
			USERPSWD TEXT NOT NULL,
			LANGUAGE_CODE TEXT DEFAULT 'ENG',
			CURRENCY_CODE TEXT DEFAULT 'INR',
			LASTPSWDCHANGE DATETIME,
			ACCTLOCK INTEGER DEFAULT 0,
			ISSIGNEDIN INTEGER DEFAULT 0,
			DEFAULTHOME TEXT DEFAULT '',
			FAILEDLOGINS INTEGER DEFAULT 0,
			CREATEDETTM DATETIME,
			LASTSIGNONDTTM DATETIME,
			LASTSIGNOFFDTTM DATETIME,
			LASTUPDDTTM DATETIME
		);

		CREATE TABLE IF NOT EXISTS pending_registrations (
			EMAILID TEXT PRIMARY KEY,
			OTP TEXT NOT NULL,
			OTP_EXPIRES_AT DATETIME NOT NULL,
			USERPSWD TEXT NOT NULL,
			UNIQUEID TEXT NOT NULL,
			CREATED_AT DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS url_access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			short_url TEXT REFERENCES url_mappings(short_url),
			session_id TEXT REFERENCES sessions(session_id),
			ip_address TEXT,
			user_agent TEXT,
			referrer TEXT,
			accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			visit_type TEXT DEFAULT 'redirect' CHECK (visit_type IN ('redirect', 'preview')),
			city TEXT,
			country TEXT,
			browser TEXT,
			device TEXT,
			operating_system TEXT,
			deleted_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS link_analytics (
			short_url TEXT PRIMARY KEY REFERENCES url_mappings(short_url),
			total_visits INTEGER DEFAULT 0,
			unique_visitors INTEGER DEFAULT 0,
			redirect_count INTEGER DEFAULT 0,
			preview_count INTEGER DEFAULT 0,
			country_counts TEXT,  -- JSON blob (e.g. {"US":10,"IN":5})
			browser_counts TEXT,  -- JSON blob
			device_counts TEXT,   -- JSON blob
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)

	return db, err
}
