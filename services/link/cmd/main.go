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
	dbConn, err := db.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/shorten", handler.ShortenHandler(dbConn)).Methods("POST")
	r.HandleFunc("/s/{shortcode}", handler.RedirectHandler(dbConn)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Link Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
