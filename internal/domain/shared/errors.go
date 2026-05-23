package shared

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string { return e.Message }
