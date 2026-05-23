package order_test

import (
	"context"
	"testing"
	"practice-ddd/internal/domain/customer"
	domainorder "practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
	apporder "practice-ddd/internal/application/order"
)

type mockCustomerRepo struct {
	customers map[string]*customer.Customer
}

func (m *mockCustomerRepo) FindByID(_ context.Context, id customer.CustomerID) (*customer.Customer, error) {
	c, ok := m.customers[id.String()]
	if !ok {
		return nil, customer.ErrCustomerNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) Save(_ context.Context, c *customer.Customer) error {
	m.customers[c.ID().String()] = c
	return nil
}

type mockProductRepo struct {
	products map[string]*product.Product
}

func (m *mockProductRepo) FindByID(_ context.Context, id product.ProductID) (*product.Product, error) {
	p, ok := m.products[id.String()]
	if !ok {
		return nil, product.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductRepo) Save(_ context.Context, p *product.Product) error {
	m.products[p.ID().String()] = p
	return nil
}

func (m *mockProductRepo) FindAll(_ context.Context) ([]*product.Product, error) {
	var out []*product.Product
	for _, p := range m.products {
		out = append(out, p)
	}
	return out, nil
}

type mockOrderRepo struct {
	orders map[string]*domainorder.Order
}

func (m *mockOrderRepo) FindByID(_ context.Context, id domainorder.OrderID) (*domainorder.Order, error) {
	o, ok := m.orders[id.String()]
	if !ok {
		return nil, domainorder.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) Save(_ context.Context, o *domainorder.Order) error {
	m.orders[o.ID().String()] = o
	return nil
}

type mockEventBus struct {
	events []domainorder.DomainEvent
}

func (m *mockEventBus) Publish(_ context.Context, e domainorder.DomainEvent) error {
	m.events = append(m.events, e)
	return nil
}

func TestPlaceOrderHandler(t *testing.T) {
	cust, _ := customer.NewCustomer(
		customer.NewCustomerID("c1"),
		customer.NewCustomerName("Alice", "Wonderland"),
		customer.NewEmail("alice@example.com"),
	)
	price, _ := shared.NewMoney(9999, "USD")
	prod, _ := product.NewProduct(product.NewProductID("p1"), "Laptop", price, 10)

	customers := &mockCustomerRepo{customers: map[string]*customer.Customer{"c1": cust}}
	products := &mockProductRepo{products: map[string]*product.Product{"p1": prod}}
	orders := &mockOrderRepo{orders: map[string]*domainorder.Order{}}
	events := &mockEventBus{}

	handler := apporder.NewPlaceOrderHandler(orders, products, customers, events)

	req := apporder.PlaceOrderRequest{
		CustomerID: "c1",
		Items: []apporder.PlaceOrderItemInput{
			{ProductID: "p1", Quantity: 2},
		},
		ShippingAddress: apporder.AddressInput{
			Line1: "123 Main St", City: "Portland",
			State: "OR", ZipCode: "97201", Country: "US",
		},
	}

	ord, err := handler.Handle(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if ord.Status() != domainorder.OrderStatusPlaced {
		t.Fatalf("expected placed, got %s", ord.Status())
	}
	if ord.Total().Amount() != 19998 {
		t.Fatalf("expected total 19998, got %d", ord.Total().Amount())
	}
	if len(events.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events.events))
	}
}

func TestPlaceOrderHandler_CustomerNotFound(t *testing.T) {
	handler := apporder.NewPlaceOrderHandler(
		&mockOrderRepo{orders: map[string]*domainorder.Order{}},
		&mockProductRepo{products: map[string]*product.Product{}},
		&mockCustomerRepo{customers: map[string]*customer.Customer{}},
		&mockEventBus{},
	)

	_, err := handler.Handle(context.Background(), apporder.PlaceOrderRequest{
		CustomerID: "nonexistent",
	})
	if err != customer.ErrCustomerNotFound {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}
