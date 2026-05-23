package sqlite

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS customers (
		id TEXT PRIMARY KEY,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		price_amount INTEGER NOT NULL,
		price_currency TEXT NOT NULL DEFAULT 'USD',
		stock INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS orders (
		id TEXT PRIMARY KEY,
		customer_id TEXT NOT NULL REFERENCES customers(id),
		status TEXT NOT NULL DEFAULT 'pending',
		shipping_address_line1 TEXT NOT NULL,
		shipping_address_city TEXT NOT NULL,
		shipping_address_state TEXT NOT NULL,
		shipping_address_zip TEXT NOT NULL,
		shipping_address_country TEXT NOT NULL,
		total_amount INTEGER NOT NULL,
		total_currency TEXT NOT NULL DEFAULT 'USD',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS order_items (
		id TEXT PRIMARY KEY,
		order_id TEXT NOT NULL REFERENCES orders(id),
		product_id TEXT NOT NULL REFERENCES products(id),
		quantity INTEGER NOT NULL,
		unit_price_amount INTEGER NOT NULL,
		unit_price_currency TEXT NOT NULL DEFAULT 'USD',
		created_at TEXT NOT NULL
	);`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
