package sqlite

import (
	"context"
	"database/sql"
	"time"
	"practice-ddd/internal/domain/customer"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) FindByID(ctx context.Context, id customer.CustomerID) (*customer.Customer, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, first_name, last_name, email, created_at, updated_at FROM customers WHERE id = ?`,
		id.String(),
	)
	var (
		cid, firstName, lastName, email, createdAt, updatedAt string
	)
	if err := row.Scan(&cid, &firstName, &lastName, &email, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, customer.ErrCustomerNotFound
		}
		return nil, err
	}
	created, _ := time.Parse(time.RFC3339, createdAt)
	updated, _ := time.Parse(time.RFC3339, updatedAt)
	return customer.RestoreCustomer(
		customer.NewCustomerID(cid),
		customer.NewCustomerName(firstName, lastName),
		customer.NewEmail(email),
		created, updated,
	), nil
}

func (r *CustomerRepository) Save(ctx context.Context, c *customer.Customer) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO customers (id, first_name, last_name, email, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			email = excluded.email,
			updated_at = excluded.updated_at`,
		c.ID().String(),
		c.Name().First(),
		c.Name().Last(),
		c.Email().String(),
		c.CreatedAt().Format(time.RFC3339),
		c.UpdatedAt().Format(time.RFC3339),
	)
	return err
}
