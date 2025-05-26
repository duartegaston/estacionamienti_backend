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
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

func main() {
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Println("No .env file found")
		}
	}
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

	// Cron scheduler
	//   "@hourly"                   : Ejecutar al inicio de cada hora
	//   "*/1 * * * *"               : Ejecutar cada minuto (para pruebas, puede ser muy frecuente para producción)
	c := cron.New(cron.WithLocation(time.Local))
	_, err = c.AddFunc("@hourly", func() {
		log.Println("Executing scheduled task: Update Finished Reservations")
		if err := reservationSvc.UpdateFinishedReservations(); err != nil {
			log.Printf("Error during scheduled task: UpdateFinishedReservations: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}
	c.Start()
	log.Println("Cron scheduler started.")

	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/availability", userReservationHandler.CheckAvailability).Methods("GET")
	r.HandleFunc("/api/reservations", userReservationHandler.CreateReservation).Methods("POST")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.GetReservation).Methods("GET")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.CancelReservation).Methods("DELETE")
	r.HandleFunc("/api/vehicle-types", userReservationHandler.GetVehicleTypes).Methods("GET")
	r.HandleFunc("/api/prices", userReservationHandler.GetPrices).Methods("GET")
	r.HandleFunc("/api/total-price", userReservationHandler.GetTotalPriceForReservation).Methods("GET")

	// Admin login
	r.HandleFunc("/api/login", adminAuthHandler.CreateUserAdmin).Methods("POST")
	r.HandleFunc("/admin/login", adminAuthHandler.Login).Methods("POST")

	// Admin endpoints (protected)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(auth.AdminAuthMiddleware)
	adminRouter.HandleFunc("/reservations", adminHandler.ListReservations).Methods("GET")
	adminRouter.HandleFunc("/reservations", adminHandler.CreateReservation).Methods("POST")
	adminRouter.HandleFunc("/reservations/{code}", adminHandler.AdminDeleteReservation).Methods("DELETE")
	adminRouter.HandleFunc("/vehicle-config", adminHandler.ListVehicleSpaces).Methods("GET")
	adminRouter.HandleFunc("/vehicle-config/{vehicle_type}", adminHandler.UpdateVehicleSpaces).Methods("PUT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
