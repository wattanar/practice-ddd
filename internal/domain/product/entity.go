package product

import (
	"time"
	"practice-ddd/internal/domain/shared"
)

type Product struct {
	id        ProductID
	name      string
	price     shared.Money
	stock     int
	createdAt time.Time
	updatedAt time.Time
}

func NewProduct(id ProductID, name string, price shared.Money, stock int) (*Product, error) {
	if name == "" {
		return nil, ErrProductNameRequired
	}
	if stock < 0 {
		return nil, ErrNegativeStock
	}
	now := time.Now()
	return &Product{
		id: id, name: name, price: price, stock: stock,
		createdAt: now, updatedAt: now,
	}, nil
}

func RestoreProduct(id ProductID, name string, price shared.Money, stock int, createdAt, updatedAt time.Time) *Product {
	return &Product{
		id: id, name: name, price: price, stock: stock,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

func (p *Product) ID() ProductID              { return p.id }
func (p *Product) Name() string               { return p.name }
func (p *Product) Price() shared.Money        { return p.price }
func (p *Product) Stock() int                 { return p.stock }
func (p *Product) CreatedAt() time.Time       { return p.createdAt }
func (p *Product) UpdatedAt() time.Time       { return p.updatedAt }

func (p *Product) AdjustStock(quantity int) error {
	newStock := p.stock + quantity
	if newStock < 0 {
		return ErrInsufficientStock
	}
	p.stock = newStock
	p.updatedAt = time.Now()
	return nil
}
