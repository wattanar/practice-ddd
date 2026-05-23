package order

import (
	"context"
	"github.com/google/uuid"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
)

type PlaceOrderRequest struct {
	CustomerID      string
	Items           []PlaceOrderItemInput
	ShippingAddress AddressInput
}

type PlaceOrderItemInput struct {
	ProductID string
	Quantity  int
}

type AddressInput struct {
	Line1   string
	City    string
	State   string
	ZipCode string
	Country string
}

type PlaceOrderHandler struct {
	orders    order.Repository
	products  product.Repository
	customers customer.Repository
	events    EventBus
}

func NewPlaceOrderHandler(
	orders order.Repository,
	products product.Repository,
	customers customer.Repository,
	events EventBus,
) *PlaceOrderHandler {
	return &PlaceOrderHandler{
		orders:    orders,
		products:  products,
		customers: customers,
		events:    events,
	}
}

func (h *PlaceOrderHandler) Handle(ctx context.Context, req PlaceOrderRequest) (*order.Order, error) {
	custID := customer.NewCustomerID(req.CustomerID)
	if _, err := h.customers.FindByID(ctx, custID); err != nil {
		return nil, err
	}

	addr, err := order.NewAddress(req.ShippingAddress.Line1, req.ShippingAddress.City,
		req.ShippingAddress.State, req.ShippingAddress.ZipCode, req.ShippingAddress.Country)
	if err != nil {
		return nil, err
	}

	ord, err := order.NewOrder(order.NewOrderID(uuid.New().String()), custID, addr)
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		prodID := product.NewProductID(item.ProductID)
		prod, err := h.products.FindByID(ctx, prodID)
		if err != nil {
			return nil, err
		}
		if err := ord.AddItem(order.NewOrderItemID(uuid.New().String()), prodID, item.Quantity, prod.Price()); err != nil {
			return nil, err
		}
	}

	events, err := ord.Place()
	if err != nil {
		return nil, err
	}

	if err := h.orders.Save(ctx, ord); err != nil {
		return nil, err
	}

	for _, event := range events {
		if err := h.events.Publish(ctx, event); err != nil {
			return nil, err
		}
	}

	return ord, nil
}
