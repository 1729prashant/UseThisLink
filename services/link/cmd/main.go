package main

import (
	"log"
	"net/http"
	"os"

	"usethislink/services/link/internal/db"
	"usethislink/services/link/internal/handler"

	"github.com/gorilla/mux"
)

func main() {
	dbPath := os.Getenv("LINK_DB_PATH")
	if dbPath == "" {
		dbPath = "usethislink_link.db"
	}
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/shorten", handler.ShortenHandler(dbConn)).Methods("POST")
	r.HandleFunc("/r/{shortcode}", handler.RedirectHandler(dbConn)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Link Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
