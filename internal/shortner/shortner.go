package shortner

import (
	"database/sql"
	"errors"
	"os"
	"strconv"

	"github.com/cespare/xxhash"
)

func generateShortURL(longURL string) string {
	// Generate 64-bit hash
	hash := xxhash.Sum64String(longURL)

	// Convert to base62 for human-readable URLs
	shortURL := toBase62(hash)

	// Truncate to desired length (between 4-10 chars)
	return shortURL[:min(len(shortURL), 8)]
}

// Base62 encoding (a-zA-Z0-9)
func toBase62(num uint64) string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	length := len(charset)
	var result []byte

	for num > 0 {
		result = append([]byte{charset[num%uint64(length)]}, result...)
		num /= uint64(length)
	}

	// Ensure minimum length
	if len(result) < 4 {
		padding := make([]byte, 4-len(result))
		for i := range padding {
			padding[i] = charset[0]
		}
		result = append(padding, result...)
	}

	return string(result)
}

// manage collisions
func StoreURL(db *sql.DB, sessionID, originalURL string) (string, error) {

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		return "", errors.New("BASE_URL not set")
	}

	//TODO: need a way to test and make robust
	for i := 0; i < 5; i++ {
		shortcode := generateShortURL(originalURL + strconv.Itoa(i))

		_, err := db.Exec(
			`INSERT OR IGNORE INTO url_mappings (short_url, original_url, session_id) VALUES (?, ?, ?)`, shortcode, originalURL, sessionID)

		if err == nil {
			// return the full short URL to the user
			return baseURL + "/" + shortcode, nil
		}
	}
	return "", errors.New("failed to generate unique short URL")
}
