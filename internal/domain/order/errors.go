package order

import "practice-ddd/internal/domain/shared"

var (
	ErrAddressLine1Required   = &shared.DomainError{Code: "ADDRESS_LINE1_REQUIRED", Message: "address line1 is required"}
	ErrAddressCityRequired    = &shared.DomainError{Code: "ADDRESS_CITY_REQUIRED", Message: "address city is required"}
	ErrAddressCountryRequired = &shared.DomainError{Code: "ADDRESS_COUNTRY_REQUIRED", Message: "address country is required"}
	ErrOrderMustHaveItems     = &shared.DomainError{Code: "ORDER_MUST_HAVE_ITEMS", Message: "order must have at least one item"}
	ErrCannotModifyPlacedOrder = &shared.DomainError{Code: "ORDER_CANNOT_MODIFY", Message: "cannot modify placed order"}
	ErrOrderAlreadyCancelled  = &shared.DomainError{Code: "ORDER_ALREADY_CANCELLED", Message: "order already cancelled"}
	ErrItemNotFound           = &shared.DomainError{Code: "ORDER_ITEM_NOT_FOUND", Message: "item not found in order"}
	ErrInvalidQuantity        = &shared.DomainError{Code: "ORDER_INVALID_QUANTITY", Message: "quantity must be positive"}
	ErrOrderNotFound          = &shared.DomainError{Code: "ORDER_NOT_FOUND", Message: "order not found"}
	ErrInvalidStatusTransition = &shared.DomainError{Code: "ORDER_INVALID_STATUS", Message: "cannot transition to target status"}
)
