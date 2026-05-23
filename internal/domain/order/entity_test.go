package order_test

import (
	"testing"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

func mustMoney(amount int64, currency string) shared.Money {
	m, _ := shared.NewMoney(amount, currency)
	return m
}

func TestNewOrder(t *testing.T) {
	custID := customer.NewCustomerID("c1")
	addr, _ := order.NewAddress("123 Main St", "Portland", "OR", "97201", "US")
	ord, err := order.NewOrder(order.NewOrderID("o1"), custID, addr)
	if err != nil {
		t.Fatal(err)
	}
	if ord.Status() != order.OrderStatusPending {
		t.Fatalf("expected pending, got %s", ord.Status())
	}
	if ord.Total().Amount() != 0 {
		t.Fatalf("expected zero total, got %d", ord.Total().Amount())
	}
}

func TestAddItem(t *testing.T) {
	t.Run("adds item and recalculates total", func(t *testing.T) {
		ord := newTestOrder(t)
		price := mustMoney(1000, "USD")
		prodID := product.NewProductID("p1")

		err := ord.AddItem(order.NewOrderItemID("i1"), prodID, 2, price)
		if err != nil {
			t.Fatal(err)
		}

		if len(ord.Items()) != 1 {
			t.Fatalf("expected 1 item, got %d", len(ord.Items()))
		}
		if ord.Total().Amount() != 2000 {
			t.Fatalf("expected total 2000, got %d", ord.Total().Amount())
		}
	})

	t.Run("increments quantity for existing product", func(t *testing.T) {
		ord := newTestOrder(t)
		prodID := product.NewProductID("p1")
		ord.AddItem(order.NewOrderItemID("i1"), prodID, 2, mustMoney(1000, "USD"))

		err := ord.AddItem(order.NewOrderItemID("i2"), prodID, 3, mustMoney(1000, "USD"))
		if err != nil {
			t.Fatal(err)
		}

		if len(ord.Items()) != 1 {
			t.Fatalf("expected 1 item, got %d", len(ord.Items()))
		}
		if ord.Items()[0].Quantity() != 5 {
			t.Fatalf("expected quantity 5, got %d", ord.Items()[0].Quantity())
		}
	})

	t.Run("rejected after order is placed", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(500, "USD"))
		ord.Place()

		err := ord.AddItem(order.NewOrderItemID("i2"), product.NewProductID("p2"), 1, mustMoney(500, "USD"))
		if err != order.ErrCannotModifyPlacedOrder {
			t.Fatalf("expected ErrCannotModifyPlacedOrder, got %v", err)
		}
	})
}

func TestRemoveItem(t *testing.T) {
	t.Run("removes item", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(1000, "USD"))
		ord.AddItem(order.NewOrderItemID("i2"), product.NewProductID("p2"), 1, mustMoney(2000, "USD"))

		err := ord.RemoveItem(product.NewProductID("p1"))
		if err != nil {
			t.Fatal(err)
		}

		if len(ord.Items()) != 1 {
			t.Fatalf("expected 1 item, got %d", len(ord.Items()))
		}
		if ord.Total().Amount() != 2000 {
			t.Fatalf("expected total 2000, got %d", ord.Total().Amount())
		}
	})

	t.Run("rejected when only one item remains", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(1000, "USD"))

		err := ord.RemoveItem(product.NewProductID("p1"))
		if err != order.ErrOrderMustHaveItems {
			t.Fatalf("expected ErrOrderMustHaveItems, got %v", err)
		}
	})
}

func TestOrderLifecycle(t *testing.T) {
	ord := newTestOrder(t)
	ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 2, mustMoney(1000, "USD"))

	t.Run("place", func(t *testing.T) {
		events, err := ord.Place()
		if err != nil {
			t.Fatal(err)
		}
		if ord.Status() != order.OrderStatusPlaced {
			t.Fatalf("expected placed, got %s", ord.Status())
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}
		if events[0].EventName() != "order.placed" {
			t.Fatalf("expected order.placed, got %s", events[0].EventName())
		}
	})

	t.Run("ship", func(t *testing.T) {
		_, err := ord.Ship()
		if err != nil {
			t.Fatal(err)
		}
		if ord.Status() != order.OrderStatusShipped {
			t.Fatalf("expected shipped, got %s", ord.Status())
		}
	})

	t.Run("cannot cancel after shipped", func(t *testing.T) {
		_, err := ord.Cancel()
		if err == nil {
			t.Fatal("expected error cancelling shipped order")
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Run("pending order can be cancelled", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(500, "USD"))
		events, err := ord.Cancel()
		if err != nil {
			t.Fatal(err)
		}
		if ord.Status() != order.OrderStatusCancelled {
			t.Fatalf("expected cancelled, got %s", ord.Status())
		}
		if events[0].EventName() != "order.cancelled" {
			t.Fatalf("expected order.cancelled event")
		}
	})

	t.Run("placed order can be cancelled", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(500, "USD"))
		ord.Place()
		_, err := ord.Cancel()
		if err != nil {
			t.Fatal(err)
		}
		if ord.Status() != order.OrderStatusCancelled {
			t.Fatalf("expected cancelled, got %s", ord.Status())
		}
	})

	t.Run("cancelled order cannot be cancelled again", func(t *testing.T) {
		ord := newTestOrder(t)
		ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(500, "USD"))
		ord.Cancel()
		_, err := ord.Cancel()
		if err == nil {
			t.Fatal("expected error cancelling already cancelled order")
		}
	})
}

func TestPlaceWithoutItems(t *testing.T) {
	ord := newTestOrder(t)
	_, err := ord.Place()
	if err != order.ErrOrderMustHaveItems {
		t.Fatalf("expected ErrOrderMustHaveItems, got %v", err)
	}
}

func newTestOrder(t *testing.T) *order.Order {
	t.Helper()
	addr, err := order.NewAddress("123 Main St", "Portland", "OR", "97201", "US")
	if err != nil {
		t.Fatal(err)
	}
	ord, err := order.NewOrder(order.NewOrderID("o-"+t.Name()), customer.NewCustomerID("c1"), addr)
	if err != nil {
		t.Fatal(err)
	}
	return ord
}
