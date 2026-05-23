package order

import (
	"context"
	"practice-ddd/internal/domain/order"
)

type EventBus interface {
	Publish(ctx context.Context, event order.DomainEvent) error
}
