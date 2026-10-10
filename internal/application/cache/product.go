package cache

import (
	"context"
	"time"

	domainproduct "practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

const productKeyPrefix = "product:"

type CachedProductRepository struct {
	inner domainproduct.Repository
	cache Cache
	ttl   time.Duration
}

func NewCachedProductRepository(inner domainproduct.Repository, cache Cache, ttl time.Duration) domainproduct.Repository {
	return &CachedProductRepository{inner: inner, cache: cache, ttl: ttl}
}

type productWire struct {
	ID            string
	Name          string
	PriceAmount   int64
	PriceCurrency string
	Stock         int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r *CachedProductRepository) FindByID(ctx context.Context, id domainproduct.ProductID) (*domainproduct.Product, error) {
	key := productKeyPrefix + id.String()
	value, err := r.cache.Get(ctx, key)
	if err == nil {
		if p, derr := productFromWire(value); derr == nil {
			return p, nil
		}
	}
	return r.inner.FindByID(ctx, id)
}

func (r *CachedProductRepository) FindAll(ctx context.Context) ([]*domainproduct.Product, error) {
	return r.inner.FindAll(ctx)
}

func (r *CachedProductRepository) Save(ctx context.Context, p *domainproduct.Product) error {
	if err := r.inner.Save(ctx, p); err != nil {
		return err
	}
	value, err := encode(productToWire(p))
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, productKeyPrefix+p.ID().String(), value, r.ttl)
}

func productToWire(p *domainproduct.Product) productWire {
	price := p.Price()
	return productWire{
		ID:            p.ID().String(),
		Name:          p.Name(),
		PriceAmount:   price.Amount(),
		PriceCurrency: price.Currency(),
		Stock:         p.Stock(),
		CreatedAt:     p.CreatedAt(),
		UpdatedAt:     p.UpdatedAt(),
	}
}

func productFromWire(value string) (*domainproduct.Product, error) {
	var w productWire
	if err := decode(value, &w); err != nil {
		return nil, err
	}
	price, err := shared.NewMoney(w.PriceAmount, w.PriceCurrency)
	if err != nil {
		return nil, err
	}
	return domainproduct.RestoreProduct(
		domainproduct.NewProductID(w.ID), w.Name, price, w.Stock,
		w.CreatedAt, w.UpdatedAt,
	), nil
}
