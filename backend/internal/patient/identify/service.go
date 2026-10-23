package identity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient/storage"
	"github.com/google/uuid"
)

const verificationSessionDuration = 15 * time.Minute

type PatientProfileFinder interface {
	GetByUserID(
		ctx context.Context,
		userID uuid.UUID,
	) (*patient.Profile, error)
}

type Service struct {
	repository    *Repository
	patientFinder PatientProfileFinder
	fileStorage   storage.FileStorage
}

func NewService(
	repository *Repository,
	patientFinder PatientProfileFinder,
	fileStorage storage.FileStorage,
) *Service {
	return &Service{
		repository:    repository,
		patientFinder: patientFinder,
		fileStorage:   fileStorage,
	}
}

func (s *Service) CreateVerification(
	ctx context.Context,
	userID uuid.UUID,
) (*Verification, error) {
	profile, err := s.patientFinder.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, patient.ErrProfileNotFound) {
			return nil, patient.ErrProfileNotFound
		}

		return nil, fmt.Errorf("get patient profile: %w", err)
	}

	latest, err := s.repository.FindLatestByPatientID(
		ctx,
		profile.ID,
	)

	if err == nil {
		if latest.Status == StatusPending ||
			latest.Status == StatusProcessing {

			if latest.ExpiresAt != nil &&
				time.Now().Before(*latest.ExpiresAt) {
				return latest, nil
			}

			_, _ = s.repository.UpdateStatus(
				ctx,
				latest.ID,
				StatusExpired,
				nil,
				nil,
			)
		}

	} else if !errors.Is(err, ErrVerificationNotFound) {
		return nil, fmt.Errorf(
			"get latest verification: %w",
			err,
		)
	}

	now := time.Now()
	expiresAt := now.Add(verificationSessionDuration)

	verification := &Verification{
		ID:           uuid.New(),
		PatientID:    profile.ID,
		Status:       StatusPending,
		DocumentType: DocumentKTP,
		ExpiresAt:    &expiresAt,
	}

	created, err := s.repository.Create(
		ctx,
		verification,
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetVerification(
	ctx context.Context,
	userID uuid.UUID,
	verificationID uuid.UUID,
) (*Verification, error) {

	profile, err := s.patientFinder.GetByUserID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	verification, err := s.repository.FindByID(
		ctx,
		verificationID,
	)
	if err != nil {
		return nil, err
	}

	if verification.PatientID != profile.ID {
		return nil, ErrVerificationNotFound
	}

	if verification.ExpiresAt != nil &&
		time.Now().After(*verification.ExpiresAt) {

		if verification.Status == StatusPending ||
			verification.Status == StatusProcessing {

			verification, err =
				s.repository.UpdateStatus(
					ctx,
					verification.ID,
					StatusExpired,
					nil,
					nil,
				)

			if err != nil {
				return nil, err
			}
		}
	}

	return verification, nil
}

func (s *Service) UploadDocument(
	ctx context.Context,
	userID uuid.UUID,
	verificationID uuid.UUID,
	reader io.Reader,
	extension string,
) (*Verification, error) {

	verification, err := s.GetVerification(
		ctx,
		userID,
		verificationID,
	)
	if err != nil {
		return nil, err
	}

	if verification.Status == StatusVerified {
		return nil, ErrVerificationCompleted
	}

	if verification.Status == StatusExpired {
		return nil, ErrVerificationExpired
	}

	if verification.Status != StatusPending &&
		verification.Status != StatusProcessing {
		return nil, ErrVerificationCompleted
	}

	storageKey := fmt.Sprintf(
		"identity/%s/%s/%s%s",
		verification.PatientID,
		verification.ID,
		uuid.New().String(),
		extension,
	)

	if err := s.fileStorage.Upload(
		ctx,
		storageKey,
		reader,
	); err != nil {
		fmt.Printf(
			"[IDENTITY] Upload KTP ke storage gagal. verification_id=%s, error=%v\n",
			verification.ID,
			err,
		)

		return nil, fmt.Errorf(
			"upload identity document: %w",
			err,
		)
	}

	updated, err := s.repository.UpdateDocument(
		ctx,
		verification.ID,
		storageKey,
		StatusPending,
	)
	if err != nil {
		fmt.Printf(
			"[IDENTITY] Update document ke database gagal. verification_id=%s, error=%v\n",
			verification.ID,
			err,
		)

		_ = s.fileStorage.Delete(ctx, storageKey)

		return nil, fmt.Errorf(
			"save identity document: %w",
			err,
		)
	}

	return updated, nil
}

func (s *Service) UploadSelfie(
	ctx context.Context,
	userID uuid.UUID,
	verificationID uuid.UUID,
	reader io.Reader,
	extension string,
) (*Verification, error) {

	verification, err := s.GetVerification(
		ctx,
		userID,
		verificationID,
	)
	if err != nil {
		return nil, err
	}

	if verification.Status == StatusVerified {
		return nil, ErrVerificationCompleted
	}

	if verification.Status == StatusExpired {
		return nil, ErrVerificationExpired
	}

	if verification.Status != StatusPending &&
		verification.Status != StatusProcessing {
		return nil, ErrVerificationCompleted
	}

	// Selfie hanya bisa dikirim jika KTP sudah ada.
	if verification.DocumentStorageKey == nil ||
		*verification.DocumentStorageKey == "" {
		return nil, ErrDocumentNotUploaded
	}

	storageKey := fmt.Sprintf(
		"identity/%s/%s/selfie/%s%s",
		verification.PatientID,
		verification.ID,
		uuid.New().String(),
		extension,
	)

	if err := s.fileStorage.Upload(ctx, storageKey, reader); err != nil {
		fmt.Printf(
			"[IDENTITY] Upload selfie ke storage gagal. verification_id=%s, error=%v\n",
			verification.ID,
			err,
		)

		return nil, fmt.Errorf("upload identity selfie: %w", err)
	}

	storageKeyPtr := &storageKey

	updated, err := s.repository.UpdateSelfie(
		ctx,
		verification.ID,
		storageKeyPtr,
		StatusManualReview,
	)
	if err != nil {
		fmt.Printf(
			"[IDENTITY] Update selfie ke database gagal. verification_id=%s, error=%v\n",
			verification.ID,
			err,
		)

		if deleteErr := s.fileStorage.Delete(ctx, storageKey); deleteErr != nil {
			fmt.Printf(
				"[IDENTITY] Cleanup selfie gagal. verification_id=%s, error=%v\n",
				verification.ID,
				deleteErr,
			)
		}

		return nil, fmt.Errorf("save identity selfie: %w", err)
	}

	return updated, nil

}

func (s *Service) ApproveVerification(
	ctx context.Context,
	adminID uuid.UUID,
	verificationID uuid.UUID,
) (*Verification, error) {

	verification, err := s.repository.FindByID(ctx, verificationID)
	if err != nil {
		return nil, err
	}

	if verification.Status != StatusManualReview {
		return nil, ErrVerificationNotInReview
	}

	now := time.Now()

	verification.Status = StatusVerified
	verification.VerifiedAt = &now
	verification.UpdatedAt = now

	if err := s.repository.MarkVerified(
		ctx,
		verificationID,
		now,
	); err != nil {
		return nil, err
	}

	return verification, nil
}

func (s *Service) RejectVerification(
	ctx context.Context,
	adminID uuid.UUID,
	verificationID uuid.UUID,
	reason string,
) (*Verification, error) {
	verification, err := s.repository.FindByID(ctx, verificationID)
	if err != nil {
		return nil, err
	}

	if verification.Status != StatusManualReview {
		return nil, ErrVerificationNotInReview
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("rejection reason is required")
	}

	verification.Status = StatusDocumentRejected
	verification.FailureReason = &reason
	verification.UpdatedAt = time.Now()

	if err := s.repository.MarkRejected(
		ctx,
		verificationID,
		reason,
	); err != nil {
		return nil, err
	}

	return verification, nil
}

func (s *Service) GetLatestVerification(
	ctx context.Context,
	userID uuid.UUID,
) (*Verification, error) {
	profile, err := s.patientFinder.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	verification, err := s.repository.FindLatestByPatientID(
		ctx,
		profile.ID,
	)
	if err != nil {
		return nil, err
	}

	if verification.ExpiresAt != nil &&
		time.Now().After(*verification.ExpiresAt) &&
		(verification.Status == StatusPending ||
			verification.Status == StatusProcessing) {

		verification, err = s.repository.UpdateStatus(
			ctx,
			verification.ID,
			StatusExpired,
			nil,
			nil,
		)

		if err != nil {
			return nil, err
		}
	}

	return verification, nil
}

func (s *Service) GetReviewDocument(
	ctx context.Context,
	verificationID uuid.UUID,
	documentType string,
) ([]byte, string, error) {
	verification, err := s.repository.FindByID(
		ctx,
		verificationID,
	)
	if err != nil {
		return nil, "", err
	}

	// Dokumen hanya dapat dibaca selama menunggu pemeriksaan admin.
	if verification.Status != StatusManualReview {
		return nil, "", ErrVerificationNotInReview
	}

	var storageKey *string

	switch strings.ToLower(documentType) {
	case "ktp":
		storageKey = verification.DocumentStorageKey
	case "selfie":
		storageKey = verification.SelfieStorageKey
	default:
		return nil, "", fmt.Errorf("unsupported document type")
	}

	if storageKey == nil || strings.TrimSpace(*storageKey) == "" {
		return nil, "", ErrDocumentNotUploaded
	}

	data, err := s.fileStorage.Read(ctx, *storageKey)
	if err != nil {
		return nil, "", fmt.Errorf("read identity document: %w", err)
	}

	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" &&
		contentType != "image/png" {
		return nil, "", fmt.Errorf("unsupported identity document format")
	}

	return data, contentType, nil
}

func (s *Service) ListPendingReview(
	ctx context.Context,
) ([]AdminIdentityQueueItem, error) {
	return s.repository.ListPendingReview(ctx)
}
