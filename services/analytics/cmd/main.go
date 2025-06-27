package main

import (
	"log"
	"net/http"
	"os"

	"usethislink/services/analytics/internal/db"
	"usethislink/services/analytics/internal/handler"

	"github.com/gorilla/mux"
)

func main() {
	dbConn, err := db.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/stats/{shortcode}", handler.StatsHandler(dbConn)).Methods("GET")
	r.HandleFunc("/history", handler.HistoryHandler(dbConn)).Methods("GET")
	r.HandleFunc("/log", handler.LogHandler(dbConn)).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Analytics Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
