package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// USERDEFN - user accounts
// pending_registrations - OTP pending
// sessions - session management
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
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

		CREATE TABLE IF NOT EXISTS sessions (
			session_id  TEXT PRIMARY KEY,
			user_agent  TEXT,
			ip_address  TEXT,
			user_email  TEXT,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)

	return db, err
}
