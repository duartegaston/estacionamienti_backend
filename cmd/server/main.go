package main

import (
	"database/sql"
	"estacionamienti/internal/api"
	"estacionamienti/internal/auth"
	"estacionamienti/internal/repository"
	"estacionamienti/internal/service"
	"log"
	"net/http"
	"os"

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

	// Repositories
	reservationRepo := repository.NewReservationRepository(db)
	adminRepo := repository.NewAdminAuthRepository(db)

	// Services
	reservationSvc := service.NewReservationService(reservationRepo)
	adminSvc := service.NewAdminAuthService(adminRepo)

	// Handlers
	userReservationHandler := api.NewUserReservationHandler(reservationSvc)
	adminHandler := api.NewAdminHandler(reservationSvc)
	adminAuthHandler := api.NewAdminAuthHandler(adminSvc)

	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/availability", userReservationHandler.CheckAvailability).Methods("GET")
	r.HandleFunc("/api/reservations", userReservationHandler.CreateReservation).Methods("POST")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.GetReservation).Methods("GET")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.UpdateReservation).Methods("PUT")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.CancelReservation).Methods("DELETE")

	// Admin login
	r.HandleFunc("/api/login", adminAuthHandler.CreateUserAdmin).Methods("POST")
	r.HandleFunc("/admin/login", adminAuthHandler.Login).Methods("POST")

	// Admin endpoints (protected)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(auth.AdminAuthMiddleware)
	adminRouter.HandleFunc("/reservations", adminHandler.ListReservations).Methods("GET")
	adminRouter.HandleFunc("/reservations/{id}", adminHandler.AdminUpdateReservation).Methods("PUT")
	adminRouter.HandleFunc("/reservations/{id}", adminHandler.AdminDeleteReservation).Methods("DELETE")
	adminRouter.HandleFunc("/spaces", adminHandler.ListVehicleSpaces).Methods("GET")
	adminRouter.HandleFunc("/spaces/{vehicle_type}", adminHandler.UpdateVehicleSpaces).Methods("PUT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
