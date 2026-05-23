package order

import (
	"context"
	"practice-ddd/internal/domain/order"
)

type ShipOrderRequest struct {
	OrderID string
}

type ShipOrderHandler struct {
	orders order.Repository
	events EventBus
}

func NewShipOrderHandler(orders order.Repository, events EventBus) *ShipOrderHandler {
	return &ShipOrderHandler{orders: orders, events: events}
}

func (h *ShipOrderHandler) Handle(ctx context.Context, req ShipOrderRequest) error {
	ord, err := h.orders.FindByID(ctx, order.NewOrderID(req.OrderID))
	if err != nil {
		return err
	}

	events, err := ord.Ship()
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
