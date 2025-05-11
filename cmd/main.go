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
	// env vars
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := db.InitDB(os.Getenv("DB_PATH"))
	if err != nil {
		logrus.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	// parse template
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	// Setup router
	r := mux.NewRouter()

	// Serve static files if you have CSS/JS later
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve index.html on "/"
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	// Keep /health for JSON health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	}).Methods("GET")

	// api endpoints
	r.HandleFunc("/shorten", api.ShortenHandler(db)).Methods("POST")
	r.HandleFunc("/{shortcode}", api.RedirectHandler(db)).Methods("GET")
	r.HandleFunc("/stats/{shortcode}", api.StatsHandler(db)).Methods("GET")

	logrus.Infof("Starting UseThisLink on port:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
