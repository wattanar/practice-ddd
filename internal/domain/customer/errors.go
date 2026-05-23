package customer

import "practice-ddd/internal/domain/shared"

var (
	ErrInvalidEmail     = &shared.DomainError{Code: "CUSTOMER_INVALID_EMAIL", Message: "invalid email address"}
	ErrNameRequired     = &shared.DomainError{Code: "CUSTOMER_NAME_REQUIRED", Message: "customer name is required"}
	ErrCustomerNotFound = &shared.DomainError{Code: "CUSTOMER_NOT_FOUND", Message: "customer not found"}
)
