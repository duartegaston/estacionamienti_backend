package main

import (
	"database/sql"
	"estacionamienti/internal/repository"
	"estacionamienti/internal/service"
	"log"
	"net/http"
	"os"

	"estacionamienti/internal/api"
	"estacionamienti/internal/auth"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
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

	repo := repository.NewReservationRepository(db)
	svc := service.NewReservationService(repo)

	userReservationHandler := api.NewUserReservationHandler(svc)
	adminHandler := api.NewAdminHandler(svc)

	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/availability", userReservationHandler.CheckAvailability).Methods("POST")
	r.HandleFunc("/api/reservations", userReservationHandler.CreateReservation).Methods("POST")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.GetReservation).Methods("GET")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.UpdateReservation).Methods("PUT")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.CancelReservation).Methods("DELETE")

	// Admin endpoints (protected)
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(auth.AdminAuthMiddleware)
	admin.HandleFunc("/reservations", adminHandler.ListReservations).Methods("GET")
	admin.HandleFunc("/reservations/{id}", adminHandler.AdminUpdateReservation).Methods("PUT")
	admin.HandleFunc("/reservations/{id}", adminHandler.AdminDeleteReservation).Methods("DELETE")
	admin.HandleFunc("/spaces/{vehicle_type}", adminHandler.UpdateVehicleSpaces).Methods("PUT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
