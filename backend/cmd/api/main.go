package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"

	"github.com/farhan-hamzah/Akesa/backend/internal/access"
	"github.com/farhan-hamzah/Akesa/backend/internal/audit"
	"github.com/farhan-hamzah/Akesa/backend/internal/config"
	"github.com/farhan-hamzah/Akesa/backend/internal/crypto"
	"github.com/farhan-hamzah/Akesa/backend/internal/database"
	"github.com/farhan-hamzah/Akesa/backend/internal/hospital"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/server"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	clerk.SetKey(cfg.ClerkSecretKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("PostgreSQL connection successful")

	// --- Wiring, one domain at a time. Each domain follows the same
	// shape: repository (owns SQL) -> service (owns business rules,
	// depends only on small interfaces of other domains) -> handler
	// (owns HTTP). Adding a new domain means adding one block here and
	// registering its routes in internal/server/server.go. ---

	fieldCipher, err := crypto.NewFieldCipher(cfg.FieldEncryptionKey)
	if err != nil {
		log.Fatalf("failed to initialize field encryption: %v", err)
	}
	// Two separate keyed hashers, two separate keys - a leak of one
	// doesn't automatically compromise the other. See internal/crypto/hash.go.
	nikHasher := crypto.NewKeyedHasher(cfg.NIKHashKey)
	auditHasher := crypto.NewKeyedHasher(cfg.AuditHashKey)

	auditRepository := audit.NewRepository(db)
	auditService := audit.NewService(auditRepository)
	auditHandler := audit.NewHandler(auditService)

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)
	userHandler := user.NewHandler(userService)

	patientRepository := patient.NewRepository(db, fieldCipher, nikHasher)
	patientService := patient.NewService(patientRepository, auditService, auditHasher)
	patientHandler := patient.NewHandler(patientService)

	hospitalRepository := hospital.NewRepository(db)
	hospitalService := hospital.NewService(hospitalRepository, userService)
	hospitalHandler := hospital.NewHandler(hospitalService)

	accessRepository := access.NewRepository(db)
	accessService := access.NewService(accessRepository, patientService, hospitalService, auditService)
	accessHandler := access.NewHandler(accessService)

	appServer := server.New(server.Dependencies{
		UserLoader:      userService,
		UserHandler:     userHandler,
		PatientHandler:  patientHandler,
		HospitalHandler: hospitalHandler,
		AccessHandler:   accessHandler,
		AuditHandler:    auditHandler,
	})

	addr := ":" + cfg.Port

	fmt.Printf("Akesa API running on http://localhost%s\n", addr)

	if err := http.ListenAndServe(addr, appServer.Handler()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
