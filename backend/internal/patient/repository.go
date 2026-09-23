package patient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/farhan-hamzah/Akesa/backend/internal/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolationCode = "23505"

type Repository struct {
	db        *pgxpool.Pool
	cipher    *crypto.FieldCipher
	nikHasher *crypto.KeyedHasher
}

func NewRepository(db *pgxpool.Pool, cipher *crypto.FieldCipher, nikHasher *crypto.KeyedHasher) *Repository {
	return &Repository{db: db, cipher: cipher, nikHasher: nikHasher}
}

func (r *Repository) FindByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return r.scanOne(ctx,
		`SELECT id, user_id, patient_code, full_name, nik, date_of_birth, gender,
		        phone_number, address, blood_type, insurance_number,
		        emergency_contact_name, emergency_contact_phone, created_at, updated_at
		 FROM patient_profiles WHERE user_id = $1`,
		userID,
	)
}

func (r *Repository) FindByPatientCode(ctx context.Context, patientCode string) (*Profile, error) {
	return r.scanOne(ctx,
		`SELECT id, user_id, patient_code, full_name, nik, date_of_birth, gender,
		        phone_number, address, blood_type, insurance_number,
		        emergency_contact_name, emergency_contact_phone, created_at, updated_at
		 FROM patient_profiles WHERE patient_code = $1`,
		patientCode,
	)
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Profile, error) {
	return r.scanOne(ctx,
		`SELECT id, user_id, patient_code, full_name, nik, date_of_birth, gender,
		        phone_number, address, blood_type, insurance_number,
		        emergency_contact_name, emergency_contact_phone, created_at, updated_at
		 FROM patient_profiles WHERE id = $1`,
		id,
	)
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, in ProfileInput) (*Profile, error) {
	encryptedNIK, err := r.cipher.Encrypt(in.NIK)
	if err != nil {
		return nil, fmt.Errorf("encrypt nik: %w", err)
	}
	nikHash := r.nikHasher.Hash(in.NIK)

	encryptedInsurance, err := r.cipher.Encrypt(derefOrEmpty(in.InsuranceNumber))
	if err != nil {
		return nil, fmt.Errorf("encrypt insurance number: %w", err)
	}

	for attempt := 0; attempt < 5; attempt++ {
		code, err := generatePatientCode()
		if err != nil {
			return nil, fmt.Errorf("generate patient code: %w", err)
		}

		row := r.db.QueryRow(ctx,
			`INSERT INTO patient_profiles (
				id, user_id, patient_code, full_name, nik, nik_hash, date_of_birth, gender,
				phone_number, address, blood_type, insurance_number,
				emergency_contact_name, emergency_contact_phone
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			RETURNING id, user_id, patient_code, full_name, nik, date_of_birth, gender,
			          phone_number, address, blood_type, insurance_number,
			          emergency_contact_name, emergency_contact_phone, created_at, updated_at`,
			uuid.New(), userID, code, in.FullName, encryptedNIK, nikHash, in.DateOfBirth, in.Gender,
			in.PhoneNumber, in.Address, in.BloodType, nilIfEmpty(encryptedInsurance),
			in.EmergencyContactName, in.EmergencyContactPhone,
		)

		profile, err := r.scanRow(row)
		if err == nil {
			return profile, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			switch pgErr.ConstraintName {
			case "patient_profiles_patient_code_key":
				continue // code collision, retry with a new code
			case "patient_profiles_user_id_key":
				return nil, ErrProfileExists
			case "patient_profiles_nik_hash_key":
				return nil, ErrNIKAlreadyRegistered
			}
		}

		return nil, fmt.Errorf("insert patient profile: %w", err)
	}

	return nil, fmt.Errorf("could not generate a unique patient code after several attempts")
}

func (r *Repository) Update(ctx context.Context, userID uuid.UUID, in ProfileInput) (*Profile, error) {
	encryptedNIK, err := r.cipher.Encrypt(in.NIK)
	if err != nil {
		return nil, fmt.Errorf("encrypt nik: %w", err)
	}
	nikHash := r.nikHasher.Hash(in.NIK)

	encryptedInsurance, err := r.cipher.Encrypt(derefOrEmpty(in.InsuranceNumber))
	if err != nil {
		return nil, fmt.Errorf("encrypt insurance number: %w", err)
	}

	row := r.db.QueryRow(ctx,
		`UPDATE patient_profiles SET
			full_name = $2, nik = $3, nik_hash = $4, date_of_birth = $5, gender = $6,
			phone_number = $7, address = $8, blood_type = $9, insurance_number = $10,
			emergency_contact_name = $11, emergency_contact_phone = $12, updated_at = NOW()
		 WHERE user_id = $1
		 RETURNING id, user_id, patient_code, full_name, nik, date_of_birth, gender,
		           phone_number, address, blood_type, insurance_number,
		           emergency_contact_name, emergency_contact_phone, created_at, updated_at`,
		userID, in.FullName, encryptedNIK, nikHash, in.DateOfBirth, in.Gender,
		in.PhoneNumber, in.Address, in.BloodType, nilIfEmpty(encryptedInsurance),
		in.EmergencyContactName, in.EmergencyContactPhone,
	)

	profile, err := r.scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == "patient_profiles_nik_hash_key" {
			return nil, ErrNIKAlreadyRegistered
		}
		return nil, fmt.Errorf("update patient profile: %w", err)
	}

	return profile, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanRow(row rowScanner) (*Profile, error) {
	profile := &Profile{}
	var encryptedNIK string
	var encryptedInsurance *string

	err := row.Scan(
		&profile.ID, &profile.UserID, &profile.PatientCode, &profile.FullName, &encryptedNIK,
		&profile.DateOfBirth, &profile.Gender, &profile.PhoneNumber, &profile.Address,
		&profile.BloodType, &encryptedInsurance, &profile.EmergencyContactName,
		&profile.EmergencyContactPhone, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	nik, err := r.cipher.Decrypt(encryptedNIK)
	if err != nil {
		return nil, fmt.Errorf("decrypt nik: %w", err)
	}
	profile.NIK = nik

	if encryptedInsurance != nil {
		insurance, err := r.cipher.Decrypt(*encryptedInsurance)
		if err != nil {
			return nil, fmt.Errorf("decrypt insurance number: %w", err)
		}
		if insurance != "" {
			profile.InsuranceNumber = &insurance
		}
	}

	return profile, nil
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (*Profile, error) {
	row := r.db.QueryRow(ctx, query, args...)

	profile, err := r.scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query patient profile: %w", err)
	}

	return profile, nil
}

func generatePatientCode() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "AKS-" + hex.EncodeToString(buf), nil
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
