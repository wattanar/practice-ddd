package shared_test

import (
	"testing"
	"practice-ddd/internal/domain/shared"
)

func TestMoneyCreation(t *testing.T) {
	t.Run("valid money", func(t *testing.T) {
		m, err := shared.NewMoney(1000, "USD")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if m.Amount() != 1000 || m.Currency() != "USD" {
			t.Fatalf("got %d %s", m.Amount(), m.Currency())
		}
	})

	t.Run("negative amount rejected", func(t *testing.T) {
		_, err := shared.NewMoney(-1, "USD")
		if err != shared.ErrNegativeAmount {
			t.Fatal("expected ErrNegativeAmount, got", err)
		}
	})

	t.Run("empty currency rejected", func(t *testing.T) {
		_, err := shared.NewMoney(100, "")
		if err != shared.ErrInvalidCurrency {
			t.Fatal("expected ErrInvalidCurrency, got", err)
		}
	})
}

func TestMoneyOperations(t *testing.T) {
	a, _ := shared.NewMoney(1000, "USD")
	b, _ := shared.NewMoney(2500, "USD")
	eur, _ := shared.NewMoney(100, "EUR")

	t.Run("add", func(t *testing.T) {
		sum, err := a.Add(b)
		if err != nil {
			t.Fatal(err)
		}
		if sum.Amount() != 3500 {
			t.Fatalf("expected 3500, got %d", sum.Amount())
		}
	})

	t.Run("add different currencies rejected", func(t *testing.T) {
		_, err := a.Add(eur)
		if err != shared.ErrCurrencyMismatch {
			t.Fatal("expected ErrCurrencyMismatch")
		}
	})

	t.Run("multiply", func(t *testing.T) {
		product := a.Multiply(3)
		if product.Amount() != 3000 {
			t.Fatalf("expected 3000, got %d", product.Amount())
		}
	})

	t.Run("equality", func(t *testing.T) {
		x, _ := shared.NewMoney(500, "USD")
		y, _ := shared.NewMoney(500, "USD")
		if !x.Equals(y) {
			t.Fatal("expected equal")
		}
		if x.Equals(eur) {
			t.Fatal("expected not equal")
		}
	})

	t.Run("immutability", func(t *testing.T) {
		original, _ := shared.NewMoney(100, "USD")
		original.Add(a)
		if original.Amount() != 100 {
			t.Fatal("original should be unchanged")
		}
	})
}
