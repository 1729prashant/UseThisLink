package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

type sessionStatus struct {
	LoggedIn bool   `json:"logged_in"`
	Email    string `json:"email,omitempty"`
}

func proxyTo(target string, passUser bool) http.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("Invalid proxy target: " + target)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	return func(w http.ResponseWriter, r *http.Request) {
		if passUser {
			if email := r.Context().Value("userEmail"); email != nil {
				r.Header.Set("X-User-Email", email.(string))
			}
			if sid := r.Context().Value("sessionID"); sid != nil {
				r.Header.Set("X-Session-ID", sid.(string))
			}
		}
		proxy.ServeHTTP(w, r)
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{w, http.StatusOK}
		next.ServeHTTP(lrw, r)
		duration := time.Since(start)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, lrw.statusCode, duration)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Auth middleware: checks session with user service
func authMiddleware(userService string, required bool) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("UTL_SESSION")
			if err != nil || cookie.Value == "" {
				if required {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			// Call user service /session endpoint
			req, _ := http.NewRequest("GET", userService+"/session", nil)
			req.AddCookie(cookie)
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if required {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			defer resp.Body.Close()
			var status sessionStatus
			body, _ := io.ReadAll(resp.Body)
			_ = json.Unmarshal(body, &status)
			if !status.LoggedIn && required {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, "userEmail", status.Email)
			ctx = context.WithValue(ctx, "sessionID", cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func main() {
	linkService := os.Getenv("LINK_SERVICE_URL")
	if linkService == "" {
		linkService = "http://localhost:8081"
	}
	analyticsService := os.Getenv("ANALYTICS_SERVICE_URL")
	if analyticsService == "" {
		analyticsService = "http://localhost:8082"
	}
	userService := os.Getenv("USER_SERVICE_URL")
	if userService == "" {
		userService = "http://localhost:8083"
	}

	r := mux.NewRouter()

	// Health and readiness endpoints
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check all backend services
		services := []string{linkService, analyticsService, userService}
		for _, svc := range services {
			resp, err := http.Get(svc + "/health")
			if err != nil || resp.StatusCode != 200 {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"ready":false,"service":"` + svc + `"}`))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ready":true}`))
	}).Methods("GET")

	// Static file serving
	staticDir := "/app/static"
	if _, err := os.Stat(staticDir); err == nil {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	}
	assetsDir := "/app/templates/assets"
	if _, err := os.Stat(assetsDir); err == nil {
		r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
	}

	// HTML page serving
	templatesDir := "/app/templates"
	serveHTML := func(filename string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(templatesDir, filename)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, path)
		}
	}
	r.HandleFunc("/", serveHTML("index.html")).Methods("GET")
	r.HandleFunc("/index.html", serveHTML("index.html")).Methods("GET")
	r.HandleFunc("/login", serveHTML("login.html")).Methods("GET")
	r.HandleFunc("/tos.html", serveHTML("tos.html")).Methods("GET")
	r.HandleFunc("/privacy.html", serveHTML("privacy.html")).Methods("GET")

	// Link Service (protected)
	r.Handle("/shorten", authMiddleware(userService, true)(proxyTo(linkService, true))).Methods("POST")
	r.Handle("/r/{shortcode}", proxyTo(linkService, false)).Methods("GET")

	// Analytics Service (protected)
	r.Handle("/stats/{shortcode}", authMiddleware(userService, true)(proxyTo(analyticsService, true))).Methods("GET")
	r.Handle("/history", authMiddleware(userService, true)(proxyTo(analyticsService, true))).Methods("GET")

	// User Service (public)
	r.HandleFunc("/register", proxyTo(userService, false)).Methods("POST")
	r.HandleFunc("/verify-otp", proxyTo(userService, false)).Methods("POST")
	r.HandleFunc("/login", proxyTo(userService, false)).Methods("POST")
	r.HandleFunc("/logout", proxyTo(userService, false)).Methods("POST")
	r.HandleFunc("/session", proxyTo(userService, false)).Methods("GET")

	// Fallback: serve index.html for unknown routes (optional, SPA support)
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(templatesDir, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, path)
	})

	// CORS and logging middleware
	handler := corsMiddleware(loggingMiddleware(r))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("API Gateway running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
