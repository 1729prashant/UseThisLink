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
	dbConn, err := db.InitDBFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer dbConn.Close()

	r := mux.NewRouter()
	r.HandleFunc("/api/register", handler.RegisterHandler(dbConn)).Methods("POST")
	r.HandleFunc("/api/verify-otp", handler.VerifyOTPHandler(dbConn)).Methods("POST")
	r.HandleFunc("/api/login", handler.LoginHandler(dbConn)).Methods("POST")
	r.HandleFunc("/api/logout", handler.LogoutHandler(dbConn)).Methods("POST")
	r.HandleFunc("/api/session", handler.SessionStatusHandler(dbConn)).Methods("GET")
	r.HandleFunc("/api/userinfo", handler.UserInfoHandler(dbConn)).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("User Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
