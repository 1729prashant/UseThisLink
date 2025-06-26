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
	dbPath := os.Getenv("ANALYTICS_DB_PATH")
	if dbPath == "" {
		dbPath = "usethislink_analytics.db"
	}
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/stats/{shortcode}", handler.StatsHandler(dbConn)).Methods("GET")
	r.HandleFunc("/history", handler.HistoryHandler(dbConn)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Analytics Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
