package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"estacionamienti/internal/api"
	"estacionamienti/internal/auth"
	"github.com/gorilla/mux"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Assign DB to handlers (assuming api.DB is a global *sql.DB)
	api.DB = db

	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/availability", api.CheckAvailability).Methods("POST")
	r.HandleFunc("/api/reservations", api.CreateReservation).Methods("POST")
	r.HandleFunc("/api/reservations/{code}", api.GetReservation).Methods("GET")
	r.HandleFunc("/api/reservations/{code}", api.UpdateReservation).Methods("PUT")
	r.HandleFunc("/api/reservations/{code}", api.CancelReservation).Methods("DELETE")

	// Admin endpoints (protected)
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(auth.AdminAuthMiddleware)
	admin.HandleFunc("/reservations", api.ListReservations).Methods("GET")
	admin.HandleFunc("/reservations/{id}", api.AdminUpdateReservation).Methods("PUT")
	admin.HandleFunc("/reservations/{id}", api.AdminDeleteReservation).Methods("DELETE")
	admin.HandleFunc("/spaces/{vehicle_type}", api.UpdateVehicleSpaces).Methods("PUT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
