package customer

import "context"

type Repository interface {
	FindByID(ctx context.Context, id CustomerID) (*Customer, error)
	Save(ctx context.Context, customer *Customer) error
}
