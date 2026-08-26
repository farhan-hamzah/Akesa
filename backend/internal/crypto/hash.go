package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// KeyedHasher produces deterministic HMAC-SHA256 hashes of sensitive
// values, using a secret key that never leaves the server.
//
// Why HMAC and not a plain sha256(value)? A NIK is only 16 digits with a
// known, structured format (region code + birth date + sequence number),
// so its real entropy is much smaller than "16 random digits" - a plain
// hash can be brute-forced offline by anyone who gets hold of it (say, a
// leaked nik_hash column, or an entry in the audit chain) simply by
// hashing every plausible NIK and comparing. HMAC with a secret key makes
// that infeasible unless the key itself also leaks - which is why the key
// must be treated like any other production secret (see .env.example).
//
// Two different KeyedHasher instances are used in this codebase, with two
// different keys - one for the NIK lookup/uniqueness hash stored in
// Postgres, one for the profile hash written into the audit chain. Key
// separation means a leak of one doesn't automatically compromise the
// other.
type KeyedHasher struct {
	key []byte
}

func NewKeyedHasher(key []byte) *KeyedHasher {
	return &KeyedHasher{key: key}
}

// Hash returns a hex-encoded HMAC-SHA256 of value. Same input + same key
// always produces the same output (needed for equality lookups like "does
// this NIK already exist"), but it cannot be reversed back into value.
func (h *KeyedHasher) Hash(value string) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
