package main

import (
	"database/sql"
	"estacionamienti/internal/api"
	"estacionamienti/internal/auth"
	"estacionamienti/internal/repository"
	"estacionamienti/internal/service"
	"github.com/stripe/stripe-go/v82"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

func initStripe() {
	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		log.Fatal("STRIPE_SECRET_KEY no está configurada.")
	}
	stripe.Key = stripeSecretKey
}

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

	initStripe()

	// Repositories
	reservationRepo := repository.NewReservationRepository(db)
	jobRepo := repository.NewJobRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	adminAuthRepo := repository.NewAdminAuthRepository(db)

	// Services
	stripeSvc := service.NewStripeService(reservationRepo)
	reservationSvc := service.NewReservationService(reservationRepo, stripeSvc)
	jobSvc := service.NewJobService(jobRepo)
	adminSvc := service.NewAdminService(adminRepo, reservationRepo, stripeSvc)
	adminAuthSvc := service.NewAdminAuthService(adminAuthRepo)

	// Handlers
	userReservationHandler := api.NewUserReservationHandler(reservationSvc)
	adminHandler := api.NewAdminHandler(adminSvc)
	adminAuthHandler := api.NewAdminAuthHandler(adminAuthSvc)
	stripeHandler := api.NewStripeWebhookHandler(os.Getenv("STRIPE_WEBHOOK_SECRET"), reservationSvc)

	// Cron scheduler
	//   "@hourly": Ejecutar al inicio de cada hora
	//   "*/1 * * * *"               : Ejecutar cada minuto (para pruebas, puede ser muy frecuente para producción)
	c := cron.New(cron.WithLocation(time.UTC))
	_, err = c.AddFunc("@hourly", func() {
		log.Println("Executing scheduled task: Update Finished Reservations")
		if err := jobSvc.UpdateFinishedReservations(); err != nil {
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
	r.HandleFunc("/api/prices", userReservationHandler.GetPrices).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/vehicle-types", userReservationHandler.GetVehicleTypes).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/availability", userReservationHandler.CheckAvailability).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/total-price", userReservationHandler.GetTotalPriceForReservation).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/reservations", userReservationHandler.CreateReservation).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.GetReservation).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/reservations/{code}", userReservationHandler.CancelReservation).Methods("DELETE", "OPTIONS")

	// Admin login
	r.HandleFunc("/api/login", adminAuthHandler.CreateUserAdmin).Methods("POST", "OPTIONS")
	r.HandleFunc("/admin/login", adminAuthHandler.Login).Methods("POST", "OPTIONS")

	// Admin endpoints (protected)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(auth.AdminAuthMiddleware)
	adminRouter.HandleFunc("/reservations", adminHandler.ListReservations).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/reservations", adminHandler.CreateReservation).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/reservations/{code}", adminHandler.AdminDeleteReservation).Methods("DELETE", "OPTIONS")
	adminRouter.HandleFunc("/vehicle-config", adminHandler.ListVehicleSpaces).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/vehicle-config/{vehicle_type}", adminHandler.UpdateVehicleSpaces).Methods("PUT", "OPTIONS")

	// Stripe
	r.HandleFunc("/webhook/stripe", stripeHandler.HandleWebhook).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/reservation/by-session", stripeHandler.GetReservationBySessionIDHandler).Methods("GET", "OPTIONS")

	allowedOrigins := handlers.AllowedOrigins([]string{
		"https://front-estacionamiento-one.vercel.app",
	})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Requested-With"})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r)))
}
