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
	"github.com/farhan-hamzah/Akesa/backend/internal/blockchain"
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

	bcClient := blockchain.NewClient(blockchain.Config{
		RPCURL:          cfg.BlockchainRPCURL,
		ContractAddress: cfg.BlockchainContractAddress,
		FromAddress:     cfg.BlockchainFromAddress,
	})
	if bcClient.IsEnabled() {
		fmt.Printf("Blockchain layer connected: contract=%s rpc=%s\n", cfg.BlockchainContractAddress, cfg.BlockchainRPCURL)
	} else {
		fmt.Println("Blockchain layer: local cryptographic hash-chain active (on-chain anchoring disabled)")
	}

	fieldCipher, err := crypto.NewFieldCipher(cfg.FieldEncryptionKey)
	if err != nil {
		log.Fatalf("failed to initialize field encryption: %v", err)
	}
	nikHasher := crypto.NewKeyedHasher(cfg.NIKHashKey)
	auditHasher := crypto.NewKeyedHasher(cfg.AuditHashKey)

	auditRepository := audit.NewRepository(db)
	auditService := audit.NewService(auditRepository, bcClient)
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
