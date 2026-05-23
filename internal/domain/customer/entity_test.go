package customer_test

import (
	"testing"
	"practice-ddd/internal/domain/customer"
)

func TestNewCustomer(t *testing.T) {
	t.Run("valid customer", func(t *testing.T) {
		c, err := customer.NewCustomer(
			customer.NewCustomerID("c1"),
			customer.NewCustomerName("Alice", "Wonderland"),
			customer.NewEmail("alice@example.com"),
		)
		if err != nil {
			t.Fatal(err)
		}
		if c.Name().FullName() != "Alice Wonderland" {
			t.Fatalf("expected Alice Wonderland, got %s", c.Name().FullName())
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		_, err := customer.NewCustomer(
			customer.NewCustomerID("c2"),
			customer.NewCustomerName("Bob", "Builder"),
			customer.NewEmail("not-an-email"),
		)
		if err != customer.ErrInvalidEmail {
			t.Fatal("expected ErrInvalidEmail, got", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := customer.NewCustomer(
			customer.NewCustomerID("c3"),
			customer.NewCustomerName("", ""),
			customer.NewEmail("test@example.com"),
		)
		if err != customer.ErrNameRequired {
			t.Fatal("expected ErrNameRequired, got", err)
		}
	})
}

func TestUpdateProfile(t *testing.T) {
	c, _ := customer.NewCustomer(
		customer.NewCustomerID("c1"),
		customer.NewCustomerName("Old", "Name"),
		customer.NewEmail("old@example.com"),
	)

	err := c.UpdateProfile(
		customer.NewCustomerName("New", "Name"),
		customer.NewEmail("new@example.com"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name().FullName() != "New Name" || c.Email().String() != "new@example.com" {
		t.Fatal("profile not updated")
	}
}
