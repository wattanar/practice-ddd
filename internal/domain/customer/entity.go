package customer

import (
	"regexp"
	"time"
)

type Customer struct {
	id        CustomerID
	name      CustomerName
	email     Email
	createdAt time.Time
	updatedAt time.Time
}

func NewCustomer(id CustomerID, name CustomerName, email Email) (*Customer, error) {
	if name.First() == "" || name.Last() == "" {
		return nil, ErrNameRequired
	}
	if !isValidEmail(email.String()) {
		return nil, ErrInvalidEmail
	}
	now := time.Now()
	return &Customer{
		id: id, name: name, email: email,
		createdAt: now, updatedAt: now,
	}, nil
}

func RestoreCustomer(id CustomerID, name CustomerName, email Email, createdAt, updatedAt time.Time) *Customer {
	return &Customer{
		id: id, name: name, email: email,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func (c *Customer) ID() CustomerID          { return c.id }
func (c *Customer) Name() CustomerName       { return c.name }
func (c *Customer) Email() Email             { return c.email }
func (c *Customer) CreatedAt() time.Time     { return c.createdAt }
func (c *Customer) UpdatedAt() time.Time     { return c.updatedAt }

func (c *Customer) UpdateProfile(name CustomerName, email Email) error {
	if name.FullName() == "" {
		return ErrNameRequired
	}
	if !isValidEmail(email.String()) {
		return ErrInvalidEmail
	}
	c.name = name
	c.email = email
	c.updatedAt = time.Now()
	return nil
}
