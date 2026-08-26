package datacategory

import (
	"errors"
	"fmt"
)

var ErrEmptyCategories = errors.New("at least one data category is required")

type InvalidCategoryError struct {
	Category Category
}

func (e *InvalidCategoryError) Error() string {
	return fmt.Sprintf("invalid data category: %q", e.Category)
}
