// Package datacategory defines the fixed set of patient-data categories
// that a hospital can request access to, and that a patient approves,
// rejects or revokes independently. This is the vocabulary shared between
// the `patient` package (which owns the actual data) and the `access`
// package (which owns consent state) - keeping it in its own package
// avoids a circular dependency between the two.
package datacategory

type Category string

const (
	Identity         Category = "IDENTITY"          // full name, NIK, date of birth, gender
	Contact          Category = "CONTACT"           // phone number, address
	MedicalBasic     Category = "MEDICAL_BASIC"     // blood type, known allergies
	Insurance        Category = "INSURANCE"         // insurance number
	EmergencyContact Category = "EMERGENCY_CONTACT" // emergency contact name & phone
)

// All lists every known category, e.g. for validation or for building UI.
var All = []Category{Identity, Contact, MedicalBasic, Insurance, EmergencyContact}

func IsValid(c Category) bool {
	for _, known := range All {
		if c == known {
			return true
		}
	}
	return false
}

// ValidateAll returns an error-free, de-duplicated slice, or an error
// naming the first invalid entry. Requests must always name at least one
// category - "give me everything" is not an allowed shape, by design,
// since the whole point of the system is category-scoped consent.
func ValidateAll(categories []Category) ([]Category, error) {
	if len(categories) == 0 {
		return nil, ErrEmptyCategories
	}

	seen := make(map[Category]struct{}, len(categories))
	result := make([]Category, 0, len(categories))

	for _, c := range categories {
		if !IsValid(c) {
			return nil, &InvalidCategoryError{Category: c}
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		result = append(result, c)
	}

	return result, nil
}
