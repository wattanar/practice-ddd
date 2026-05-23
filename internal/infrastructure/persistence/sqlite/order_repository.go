package sqlite

import (
	"context"
	"database/sql"
	"time"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id order.OrderID) (*order.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, customer_id, status,
		       shipping_address_line1, shipping_address_city, shipping_address_state,
		       shipping_address_zip, shipping_address_country,
		       total_amount, total_currency, created_at, updated_at
		FROM orders WHERE id = ?`, id.String(),
	)

	var (
		oid, custID, status, line1, city, state, zip, country, currency, createdAt, updatedAt string
		totalAmount                                                                             int64
	)
	if err := row.Scan(&oid, &custID, &status, &line1, &city, &state, &zip, &country,
		&totalAmount, &currency, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, order.ErrOrderNotFound
		}
		return nil, err
	}

	addr, _ := order.NewAddress(line1, city, state, zip, country)
	total, _ := shared.NewMoney(totalAmount, currency)
	created, _ := time.Parse(time.RFC3339, createdAt)
	updated, _ := time.Parse(time.RFC3339, updatedAt)

	items, err := r.findItems(ctx, id)
	if err != nil {
		return nil, err
	}

	return order.RestoreOrder(
		order.NewOrderID(oid), customer.NewCustomerID(custID), items,
		order.OrderStatus(status), addr, total, created, updated,
	), nil
}

func (r *OrderRepository) findItems(ctx context.Context, oid order.OrderID) ([]*order.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, quantity, unit_price_amount, unit_price_currency
		FROM order_items WHERE order_id = ?`, oid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*order.OrderItem
	for rows.Next() {
		var (
			iid, pid, currency string
			qty                int
			priceAmount        int64
		)
		if err := rows.Scan(&iid, &pid, &qty, &priceAmount, &currency); err != nil {
			return nil, err
		}
		price, _ := shared.NewMoney(priceAmount, currency)
		items = append(items, order.RestoreOrderItem(
			order.NewOrderItemID(iid), product.NewProductID(pid), qty, price,
		))
	}
	return items, rows.Err()
}

func (r *OrderRepository) Save(ctx context.Context, ord *order.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, customer_id, status,
		                   shipping_address_line1, shipping_address_city, shipping_address_state,
		                   shipping_address_zip, shipping_address_country,
		                   total_amount, total_currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			total_amount = excluded.total_amount,
			total_currency = excluded.total_currency,
			updated_at = excluded.updated_at`,
		ord.ID().String(), ord.CustomerID().String(), string(ord.Status()),
		ord.ShippingAddress().Line1(), ord.ShippingAddress().City(),
		ord.ShippingAddress().State(), ord.ShippingAddress().ZipCode(),
		ord.ShippingAddress().Country(),
		ord.Total().Amount(), ord.Total().Currency(),
		ord.CreatedAt().Format(time.RFC3339), ord.UpdatedAt().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = ?`, ord.ID().String()); err != nil {
		return err
	}

	for _, item := range ord.Items() {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price_amount, unit_price_currency)
			VALUES (?, ?, ?, ?, ?, ?)`,
			item.ID().String(), ord.ID().String(), item.ProductID().String(),
			item.Quantity(), item.UnitPrice().Amount(), item.UnitPrice().Currency(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
