package sqlite

import (
	"context"
	"database/sql"
	"time"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func scanProduct(row *sql.Row) (*product.Product, error) {
	var (
		id, name, currency, createdAt, updatedAt string
		amount                                   int64
		stock                                    int
	)
	if err := row.Scan(&id, &name, &amount, &currency, &stock, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, product.ErrProductNotFound
		}
		return nil, err
	}
	price, _ := shared.NewMoney(amount, currency)
	created, _ := time.Parse(time.RFC3339, createdAt)
	updated, _ := time.Parse(time.RFC3339, updatedAt)
	return product.RestoreProduct(
		product.NewProductID(id), name, price, stock, created, updated,
	), nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id product.ProductID) (*product.Product, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, price_amount, price_currency, stock, created_at, updated_at FROM products WHERE id = ?`,
		id.String(),
	)
	return scanProduct(row)
}

func (r *ProductRepository) FindAll(ctx context.Context) ([]*product.Product, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, price_amount, price_currency, stock, created_at, updated_at FROM products ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*product.Product
	for rows.Next() {
		var (
			id, name, currency, createdAt, updatedAt string
			amount                                   int64
			stock                                    int
		)
		if err := rows.Scan(&id, &name, &amount, &currency, &stock, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		price, _ := shared.NewMoney(amount, currency)
		created, _ := time.Parse(time.RFC3339, createdAt)
		updated, _ := time.Parse(time.RFC3339, updatedAt)
		products = append(products, product.RestoreProduct(
			product.NewProductID(id), name, price, stock, created, updated,
		))
	}
	return products, rows.Err()
}

func (r *ProductRepository) Save(ctx context.Context, p *product.Product) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO products (id, name, price_amount, price_currency, stock, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			price_amount = excluded.price_amount,
			price_currency = excluded.price_currency,
			stock = excluded.stock,
			updated_at = excluded.updated_at`,
		p.ID().String(), p.Name(), p.Price().Amount(), p.Price().Currency(),
		p.Stock(), p.CreatedAt().Format(time.RFC3339), p.UpdatedAt().Format(time.RFC3339),
	)
	return err
}
