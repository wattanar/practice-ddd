package product_test

import (
	"testing"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

func TestNewProduct(t *testing.T) {
	t.Run("valid product", func(t *testing.T) {
		price, _ := shared.NewMoney(9999, "USD")
		p, err := product.NewProduct(product.NewProductID("p1"), "Laptop", price, 10)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name() != "Laptop" || p.Stock() != 10 || !p.Price().Equals(price) {
			t.Fatal("product fields don't match")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		price, _ := shared.NewMoney(100, "USD")
		_, err := product.NewProduct(product.NewProductID("p1"), "", price, 0)
		if err != product.ErrProductNameRequired {
			t.Fatal("expected ErrProductNameRequired")
		}
	})

	t.Run("rejects negative stock", func(t *testing.T) {
		price, _ := shared.NewMoney(100, "USD")
		_, err := product.NewProduct(product.NewProductID("p1"), "Test", price, -1)
		if err != product.ErrNegativeStock {
			t.Fatal("expected ErrNegativeStock")
		}
	})
}

func TestAdjustStock(t *testing.T) {
	p := newTestProduct(t, 10)

	err := p.AdjustStock(-3)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stock() != 7 {
		t.Fatalf("expected stock 7, got %d", p.Stock())
	}

	err = p.AdjustStock(-10)
	if err != product.ErrInsufficientStock {
		t.Fatal("expected ErrInsufficientStock, got", err)
	}

	err = p.AdjustStock(5)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stock() != 12 {
		t.Fatalf("expected stock 12, got %d", p.Stock())
	}
}

func newTestProduct(t *testing.T, stock int) *product.Product {
	t.Helper()
	price, _ := shared.NewMoney(1000, "USD")
	p, err := product.NewProduct(product.NewProductID("test"), "Test Product", price, stock)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
