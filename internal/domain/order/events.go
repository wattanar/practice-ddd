package order

import (
	"time"
	"practice-ddd/internal/domain/shared"
	"practice-ddd/internal/domain/customer"
)

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

type OrderPlaced struct {
	OrderID    OrderID
	CustomerID customer.CustomerID
	Total      shared.Money
	PlacedAt   time.Time
}

func (e OrderPlaced) EventName() string         { return "order.placed" }
func (e OrderPlaced) OccurredAt() time.Time     { return e.PlacedAt }

type OrderCancelled struct {
	OrderID     OrderID
	CustomerID  customer.CustomerID
	CancelledAt time.Time
}

func (e OrderCancelled) EventName() string      { return "order.cancelled" }
func (e OrderCancelled) OccurredAt() time.Time  { return e.CancelledAt }
