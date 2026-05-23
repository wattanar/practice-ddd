package order

import (
	"context"
	"practice-ddd/internal/domain/order"
)

type CancelOrderRequest struct {
	OrderID string
}

type CancelOrderHandler struct {
	orders order.Repository
	events EventBus
}

func NewCancelOrderHandler(orders order.Repository, events EventBus) *CancelOrderHandler {
	return &CancelOrderHandler{orders: orders, events: events}
}

func (h *CancelOrderHandler) Handle(ctx context.Context, req CancelOrderRequest) error {
	ord, err := h.orders.FindByID(ctx, order.NewOrderID(req.OrderID))
	if err != nil {
		return err
	}

	events, err := ord.Cancel()
	if err != nil {
		return err
	}

	if err := h.orders.Save(ctx, ord); err != nil {
		return err
	}

	for _, event := range events {
		if err := h.events.Publish(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
