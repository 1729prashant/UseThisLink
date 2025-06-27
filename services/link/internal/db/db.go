package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// url_mappings - shortened urls
// sessions - who is visiting (one row per browser cookie)
// url_access_logs - every redirect or preview hit (internal analytics)
// link_analytics - cached aggregate stats per short_url
func InitDBFromEnv() (*sql.DB, error) {
	host := os.Getenv("PGHOST")
	port := os.Getenv("PGPORT")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	dbname := os.Getenv("PGDATABASE")
	schema := os.Getenv("PGSCHEMA")
	if schema == "" {
		schema = "link"
	}
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema))
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expiry_date TIMESTAMP,
		is_logged_in BOOLEAN DEFAULT FALSE,
		PRIMARY KEY (short_url, session_id, user_email)
	);
	`)
	if err != nil {
		return nil, err
	}
	return db, nil
}
