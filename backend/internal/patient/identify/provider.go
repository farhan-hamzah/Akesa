package identity

import "context"

type IdentityProvider interface {
	ExtractDocument(
		ctx context.Context,
		document []byte,
	) (*DocumentExtractionResult, error)

	VerifyDocument(
		ctx context.Context,
		document []byte,
	) (*DocumentVerificationResult, error)

	CheckLiveness(
		ctx context.Context,
		selfie []byte,
	) (*LivenessResult, error)

	MatchFace(
		ctx context.Context,
		document []byte,
		selfie []byte,
	) (*FaceMatchResult, error)
}

type DocumentExtractionResult struct {
	NIK         string
	FullName    string
	DateOfBirth string
	Gender      string
}

type DocumentVerificationResult struct {
	Verified bool
	Reason   string
}

type LivenessResult struct {
	Passed bool
	Reason string
}

type FaceMatchResult struct {
	Matched bool
	Score   float64
}
