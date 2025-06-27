package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// USERDEFN, sessions, pending_registrations
func InitDBFromEnv() (*sql.DB, error) {
	host := os.Getenv("PGHOST")
	port := os.Getenv("PGPORT")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	dbname := os.Getenv("PGDATABASE")
	schema := os.Getenv("PGSCHEMA")
	if schema == "" {
		schema = "user"
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
	CREATE TABLE IF NOT EXISTS USERDEFN (
		EMAILID TEXT PRIMARY KEY NOT NULL,
		UNIQUEID TEXT NOT NULL,
		FULLNAMEDESC TEXT DEFAULT '',
		USERPSWD TEXT NOT NULL,
		LANGUAGE_CODE TEXT DEFAULT 'ENG',
		CURRENCY_CODE TEXT DEFAULT 'INR',
		LASTPSWDCHANGE TIMESTAMP,
		ACCTLOCK INTEGER DEFAULT 0,
		ISSIGNEDIN INTEGER DEFAULT 0,
		DEFAULTHOME TEXT DEFAULT '',
		FAILEDLOGINS INTEGER DEFAULT 0,
		CREATEDETTM TIMESTAMP,
		LASTSIGNONDTTM TIMESTAMP,
		LASTSIGNOFFDTTM TIMESTAMP,
		LASTUPDDTTM TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS pending_registrations (
		EMAILID TEXT PRIMARY KEY,
		OTP TEXT NOT NULL,
		OTP_EXPIRES_AT TIMESTAMP NOT NULL,
		USERPSWD TEXT NOT NULL,
		UNIQUEID TEXT NOT NULL,
		CREATED_AT TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		session_id  TEXT PRIMARY KEY,
		user_agent  TEXT,
		ip_address  TEXT,
		user_email  TEXT,
		created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		return nil, err
	}
	return db, nil
}
