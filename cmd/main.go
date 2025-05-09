package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"usethislink/api"
	"usethislink/internal/db"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		log.Fatal("BASE_URL not set in environment")
	}
	logrus.SetOutput(os.Stdout) // Switch to file later

	db, err := db.InitDB(os.Getenv("DB_PATH"))
	if err != nil {
		logrus.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
		tmpl.Execute(w, nil)
	}).Methods("GET")
	r.HandleFunc("/shorten", api.ShortenHandler(db)).Methods("POST")
	r.HandleFunc("/{shortcode}", api.RedirectHandler(db)).Methods("GET")
	r.HandleFunc("/stats/{shortcode}", api.StatsHandler(db)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logrus.Infof("Starting UseThisLink on :%s", port)
	http.ListenAndServe(":"+port, r)
}
