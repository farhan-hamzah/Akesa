package patientqr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func generateToken() (string, error) {
	raw := make([]byte, 32)

	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func generateDisplayCode() string {
	b := make([]byte, 4)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%02X%02X%02X%02X", b[0], b[1], b[2], b[3])
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) GetOrCreate(
	ctx context.Context,
	userID uuid.UUID,
) (*Credential, error) {

	existing, err := s.repo.FindByUserID(ctx, userID)

	if err == nil {
		if existing.IsActive {
			return existing, nil
		}

		token, err := generateToken()
		if err != nil {
			return nil, fmt.Errorf("generate QR token: %w", err)
		}

		existing.TokenHash = hashToken(token)
		existing.DisplayCode = generateDisplayCode()
		existing.IsActive = true
		existing.Token = token

		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update QR credential: %w", err)
		}

		return existing, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("find QR credential: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate QR token: %w", err)
	}

	credential := &Credential{
		ID:          uuid.New(),
		UserID:      userID,
		DisplayCode: generateDisplayCode(),
		TokenHash:   hashToken(token),
		IsActive:    true,
		Token:       token,
	}

	if err := s.repo.Create(ctx, credential); err != nil {
		return nil, fmt.Errorf("create QR credential: %w", err)
	}

	return credential, nil
}

func (s *Service) Rotate(
	ctx context.Context,
	userID uuid.UUID,
) (*Credential, error) {

	existing, err := s.repo.FindByUserID(ctx, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.GetOrCreate(ctx, userID)
		}

		return nil, fmt.Errorf("find QR credential: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate QR token: %w", err)
	}

	existing.DisplayCode = generateDisplayCode()
	existing.TokenHash = hashToken(token)
	existing.IsActive = true
	existing.Token = token

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("rotate QR credential: %w", err)
	}

	return existing, nil
}
