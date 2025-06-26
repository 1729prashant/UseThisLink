package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// url_access_logs - every redirect or preview hit (internal analytics)
// link_analytics - cached aggregate stats per short_url
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS url_access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			short_url TEXT,
			session_id TEXT,
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
			short_url TEXT PRIMARY KEY,
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
