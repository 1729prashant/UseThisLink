package main

import (
	"log"
	"net/http"
	"os"

	"usethislink/services/user/internal/db"
	"usethislink/services/user/internal/handler"

	"github.com/gorilla/mux"
)

func main() {
	dbPath := os.Getenv("USER_DB_PATH")
	if dbPath == "" {
		dbPath = "usethislink_user.db"
	}
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/register", handler.RegisterHandler(dbConn)).Methods("POST")
	r.HandleFunc("/verify-otp", handler.VerifyOTPHandler(dbConn)).Methods("POST")
	r.HandleFunc("/login", handler.LoginHandler(dbConn)).Methods("POST")
	r.HandleFunc("/logout", handler.LogoutHandler(dbConn)).Methods("POST")
	r.HandleFunc("/session", handler.SessionStatusHandler(dbConn)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("User Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
