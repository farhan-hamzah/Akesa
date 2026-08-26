package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"

	"github.com/farhan-hamzah/Akesa/backend/internal/config"
	"github.com/farhan-hamzah/Akesa/backend/internal/database"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/server"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(
			"failed to load config: %v",
			err,
		)
	}

	clerk.SetKey(
		cfg.ClerkSecretKey,
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)

	if err != nil {
		log.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}

	defer db.Close()

	fmt.Println(
		"PostgreSQL connection successful",
	)

	// User
	userRepository := user.NewRepository(db)

	userService := user.NewService(
		userRepository,
	)

	userHandler := user.NewHandler(
		userService,
	)

	// Patient
	patientRepository := patient.NewRepository(db)

	patientService := patient.NewService(
		patientRepository,
	)

	patientHandler := patient.NewHandler(
		patientService,
		userService,
	)

	// Server
	appServer := server.New(
		userHandler,
		patientHandler,
	)

	addr := ":" + cfg.Port

	fmt.Printf(
		"Akesa API running on http://localhost%s\n",
		addr,
	)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           appServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf(
			"server failed: %v",
			err,
		)
	}
}
