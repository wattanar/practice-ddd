package product

import "practice-ddd/internal/domain/shared"

var (
	ErrProductNameRequired = &shared.DomainError{Code: "PRODUCT_NAME_REQUIRED", Message: "product name is required"}
	ErrNegativeStock       = &shared.DomainError{Code: "PRODUCT_NEGATIVE_STOCK", Message: "stock cannot be negative"}
	ErrInsufficientStock   = &shared.DomainError{Code: "PRODUCT_INSUFFICIENT_STOCK", Message: "insufficient stock"}
	ErrProductNotFound     = &shared.DomainError{Code: "PRODUCT_NOT_FOUND", Message: "product not found"}
)
