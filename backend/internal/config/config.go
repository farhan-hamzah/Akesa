package config

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	ClerkSecretKey string

	// FieldEncryptionKey is the AES-256 key used to encrypt sensitive
	// fields (NIK, insurance number) at rest. Must be exactly 32 bytes
	// once base64-decoded. Generate one with: openssl rand -base64 32
	FieldEncryptionKey []byte

	// NIKHashKey is the HMAC key used for the deterministic, one-way NIK
	// lookup hash stored alongside the encrypted NIK - it's what lets us
	// enforce "one NIK, one profile" and search by NIK without ever
	// decrypting every row. Deliberately a different key from
	// FieldEncryptionKey and AuditHashKey (key separation).
	NIKHashKey []byte

	// AuditHashKey is the HMAC key used to compute the profile content
	// hash that gets written into the audit hash chain. Also
	// deliberately separate from the other two keys.
	AuditHashKey []byte

	// Blockchain layer configuration (optional; falls back to local cryptographic ledger if empty)
	BlockchainRPCURL          string
	BlockchainContractAddress string
	BlockchainFromAddress     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		AppEnv:                    os.Getenv("APP_ENV"),
		Port:                      os.Getenv("PORT"),
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		ClerkSecretKey:            os.Getenv("CLERK_SECRET_KEY"),
		BlockchainRPCURL:          os.Getenv("BLOCKCHAIN_RPC_URL"),
		BlockchainContractAddress: os.Getenv("BLOCKCHAIN_CONTRACT_ADDRESS"),
		BlockchainFromAddress:     os.Getenv("BLOCKCHAIN_FROM_ADDRESS"),
	}

	if config.AppEnv == "" {
		config.AppEnv = "development"
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if config.ClerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	fieldKey, err := decodeBase64Key("FIELD_ENCRYPTION_KEY", 32)
	if err != nil {
		return nil, err
	}
	config.FieldEncryptionKey = fieldKey

	nikHashKey, err := requireSecret("NIK_HASH_KEY")
	if err != nil {
		return nil, err
	}
	config.NIKHashKey = nikHashKey

	auditHashKey, err := requireSecret("AUDIT_HASH_KEY")
	if err != nil {
		return nil, err
	}
	config.AuditHashKey = auditHashKey

	return config, nil
}

// decodeBase64Key reads a base64-encoded env var and requires it to
// decode to exactly wantBytes bytes - used for the AES-256 encryption key,
// which (unlike an HMAC key) has a hard length requirement.
func decodeBase64Key(envVar string, wantBytes int) ([]byte, error) {
	raw := os.Getenv(envVar)
	if raw == "" {
		return nil, fmt.Errorf("%s is required (generate one with: openssl rand -base64 32)", envVar)
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be valid base64: %w", envVar, err)
	}

	if len(decoded) != wantBytes {
		return nil, fmt.Errorf("%s must decode to %d bytes, got %d", envVar, wantBytes, len(decoded))
	}

	return decoded, nil
}

// requireSecret reads a plain-string secret env var (used as an HMAC key,
// which has no fixed length requirement) and rejects it if it's missing or
// suspiciously short - a short key defeats the point of keyed hashing.
func requireSecret(envVar string) ([]byte, error) {
	raw := os.Getenv(envVar)
	if len(raw) < 16 {
		return nil, fmt.Errorf("%s is required and should be at least 16 random characters", envVar)
	}
	return []byte(raw), nil
}
