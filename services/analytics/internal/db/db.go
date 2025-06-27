package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

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
		schema = "analytics"
	}
	if port == "" {
		port = "5432"
	}
	// Build DSN
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// Create schema if not exists
	_, err = db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema))
	if err != nil {
		return nil, err
	}
	// Set search_path
	_, err = db.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema))
	if err != nil {
		return nil, err
	}
	// Create tables if not exists
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS url_access_logs (
		id SERIAL PRIMARY KEY,
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
		deleted_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS link_analytics (
		short_url TEXT PRIMARY KEY,
		total_visits INTEGER DEFAULT 0,
		unique_visitors INTEGER DEFAULT 0,
		redirect_count INTEGER DEFAULT 0,
		preview_count INTEGER DEFAULT 0,
		country_counts TEXT,
		browser_counts TEXT,
		device_counts TEXT,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		return nil, err
	}
	return db, nil
}
