package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"usethislink/api"
	"usethislink/internal/db"
	"usethislink/internal/mw"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func main() {
	// env vars
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
	r.Use(mw.SessionMiddleware) // manage sessions

	// Serve static files if CSS/JS is used later
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve assets (CSS, JS, etc.)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("templates/assets"))))

	// Serve login.html
	r.Path("/login.html").Handler(http.FileServer(http.Dir("templates")))

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
	r.HandleFunc("/admin/analytics", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query(`
        SELECT date(accessed_at) as day, COUNT(*) 
        FROM url_access_logs
        GROUP BY day ORDER BY day`)
		type row struct {
			Day string
			C   int
		}
		var data []row
		for rows.Next() {
			var d row
			rows.Scan(&d.Day, &d.C)
			data = append(data, d)
		}
		json.NewEncoder(w).Encode(data)
		w.Header().Set("Content-Type", "application/json")
	}).Methods("GET")

	r.HandleFunc("/api/history", api.HistoryHandler(db)).Methods("GET")
	r.HandleFunc("/api/qrcode", api.QRCodeHandler()).Methods("GET")
	r.HandleFunc("/api/register", api.RegisterHandler(db)).Methods("POST")
	r.HandleFunc("/api/verify-otp", api.VerifyOTPHandler(db)).Methods("POST")
	r.HandleFunc("/api/login", api.LoginHandler(db)).Methods("POST")
	r.HandleFunc("/api/logout", api.LogoutHandler(db)).Methods("POST")

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		logrus.Infof("Starting UseThisLink on port:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Setup channel to listen for signals to gracefully shutdown
	quit := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM will not be caught
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal in the quit channel
	sig := <-quit
	logrus.Infof("Received shutdown signal: %v", sig)

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Doesn't block if no connections, otherwise waits until the timeout
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server stopped")
}
