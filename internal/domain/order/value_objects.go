package order

import "strings"

type OrderID struct {
	value string
}

func NewOrderID(value string) OrderID {
	return OrderID{value: value}
}

func (id OrderID) String() string { return id.value }

type OrderItemID struct {
	value string
}

func NewOrderItemID(value string) OrderItemID {
	return OrderItemID{value: value}
}

func (id OrderItemID) String() string { return id.value }

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPlaced    OrderStatus = "placed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

var validTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending:   {OrderStatusPlaced, OrderStatusCancelled},
	OrderStatusPlaced:    {OrderStatusShipped, OrderStatusCancelled},
	OrderStatusShipped:   {OrderStatusDelivered},
	OrderStatusDelivered: {},
	OrderStatusCancelled: {},
}

func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	for _, allowed := range validTransitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}

type Address struct {
	line1   string
	city    string
	state   string
	zipCode string
	country string
}

func NewAddress(line1, city, state, zipCode, country string) (Address, error) {
	if strings.TrimSpace(line1) == "" {
		return Address{}, ErrAddressLine1Required
	}
	if strings.TrimSpace(city) == "" {
		return Address{}, ErrAddressCityRequired
	}
	if strings.TrimSpace(country) == "" {
		return Address{}, ErrAddressCountryRequired
	}
	return Address{line1: line1, city: city, state: state, zipCode: zipCode, country: country}, nil
}

func (a Address) Line1() string   { return a.line1 }
func (a Address) City() string    { return a.city }
func (a Address) State() string   { return a.state }
func (a Address) ZipCode() string { return a.zipCode }
func (a Address) Country() string { return a.country }
