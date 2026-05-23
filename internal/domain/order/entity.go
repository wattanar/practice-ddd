package order

import (
	"fmt"
	"time"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

type OrderItem struct {
	id        OrderItemID
	productID product.ProductID
	quantity  int
	unitPrice shared.Money
}

func NewOrderItem(id OrderItemID, productID product.ProductID, quantity int, unitPrice shared.Money) (*OrderItem, error) {
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	return &OrderItem{id: id, productID: productID, quantity: quantity, unitPrice: unitPrice}, nil
}

func RestoreOrderItem(id OrderItemID, productID product.ProductID, quantity int, unitPrice shared.Money) *OrderItem {
	return &OrderItem{id: id, productID: productID, quantity: quantity, unitPrice: unitPrice}
}

func (i *OrderItem) ID() OrderItemID                { return i.id }
func (i *OrderItem) ProductID() product.ProductID    { return i.productID }
func (i *OrderItem) Quantity() int                   { return i.quantity }
func (i *OrderItem) UnitPrice() shared.Money         { return i.unitPrice }
func (i *OrderItem) Total() shared.Money             { return i.unitPrice.Multiply(i.quantity) }

type Order struct {
	id           OrderID
	customerID   customer.CustomerID
	items        []*OrderItem
	status       OrderStatus
	shippingAddr Address
	total        shared.Money
	createdAt    time.Time
	updatedAt    time.Time
}

func NewOrder(id OrderID, customerID customer.CustomerID, shippingAddr Address) (*Order, error) {
	zero, _ := shared.NewMoney(0, "USD")
	now := time.Now()
	return &Order{
		id: id, customerID: customerID, status: OrderStatusPending,
		shippingAddr: shippingAddr, total: zero,
		createdAt: now, updatedAt: now,
	}, nil
}

func RestoreOrder(
	id OrderID, customerID customer.CustomerID, items []*OrderItem,
	status OrderStatus, shippingAddr Address, total shared.Money,
	createdAt, updatedAt time.Time,
) *Order {
	return &Order{
		id: id, customerID: customerID, items: items, status: status,
		shippingAddr: shippingAddr, total: total,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

func (o *Order) ID() OrderID                       { return o.id }
func (o *Order) CustomerID() customer.CustomerID   { return o.customerID }
func (o *Order) Items() []*OrderItem               { return o.items }
func (o *Order) Status() OrderStatus               { return o.status }
func (o *Order) ShippingAddress() Address           { return o.shippingAddr }
func (o *Order) Total() shared.Money               { return o.total }
func (o *Order) CreatedAt() time.Time              { return o.createdAt }
func (o *Order) UpdatedAt() time.Time              { return o.updatedAt }

func (o *Order) AddItem(id OrderItemID, productID product.ProductID, quantity int, unitPrice shared.Money) error {
	if o.status != OrderStatusPending {
		return ErrCannotModifyPlacedOrder
	}
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	for _, item := range o.items {
		if item.productID.String() == productID.String() {
			item.quantity += quantity
			o.recalculateTotal()
			o.updatedAt = time.Now()
			return nil
		}
	}
	item, err := NewOrderItem(id, productID, quantity, unitPrice)
	if err != nil {
		return err
	}
	o.items = append(o.items, item)
	o.recalculateTotal()
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) RemoveItem(productID product.ProductID) error {
	if o.status != OrderStatusPending {
		return ErrCannotModifyPlacedOrder
	}
	if len(o.items) <= 1 {
		return ErrOrderMustHaveItems
	}
	for i, item := range o.items {
		if item.productID.String() == productID.String() {
			o.items = append(o.items[:i], o.items[i+1:]...)
			o.recalculateTotal()
			o.updatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotFound
}

func (o *Order) Place() ([]DomainEvent, error) {
	if !o.status.CanTransitionTo(OrderStatusPlaced) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatusTransition, o.status)
	}
	if len(o.items) == 0 {
		return nil, ErrOrderMustHaveItems
	}
	o.status = OrderStatusPlaced
	o.updatedAt = time.Now()
	return []DomainEvent{OrderPlaced{
		OrderID: o.id, CustomerID: o.customerID,
		Total: o.total, PlacedAt: o.updatedAt,
	}}, nil
}

func (o *Order) Cancel() ([]DomainEvent, error) {
	if !o.status.CanTransitionTo(OrderStatusCancelled) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatusTransition, o.status)
	}
	o.status = OrderStatusCancelled
	o.updatedAt = time.Now()
	return []DomainEvent{OrderCancelled{
		OrderID: o.id, CustomerID: o.customerID, CancelledAt: o.updatedAt,
	}}, nil
}

func (o *Order) Ship() ([]DomainEvent, error) {
	if !o.status.CanTransitionTo(OrderStatusShipped) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatusTransition, o.status)
	}
	o.status = OrderStatusShipped
	o.updatedAt = time.Now()
	return nil, nil
}

func (o *Order) recalculateTotal() {
	zero, _ := shared.NewMoney(0, "USD")
	total := zero
	for _, item := range o.items {
		total, _ = total.Add(item.Total())
	}
	o.total = total
}
