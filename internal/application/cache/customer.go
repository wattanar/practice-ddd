package cache

import (
	"context"
	"time"

	domaincustomer "practice-ddd/internal/domain/customer"
)

const customerKeyPrefix = "customer:"

type CachedCustomerRepository struct {
	inner domaincustomer.Repository
	cache Cache
	ttl   time.Duration
}

func NewCachedCustomerRepository(inner domaincustomer.Repository, cache Cache, ttl time.Duration) domaincustomer.Repository {
	return &CachedCustomerRepository{inner: inner, cache: cache, ttl: ttl}
}

type customerWire struct {
	First     string
	Last      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *CachedCustomerRepository) FindByID(ctx context.Context, id domaincustomer.CustomerID) (*domaincustomer.Customer, error) {
	key := customerKeyPrefix + id.String()
	value, err := r.cache.Get(ctx, key)
	if err == nil {
		if c, derr := customerFromWire(value, id); derr == nil {
			return c, nil
		}
	}
	return r.inner.FindByID(ctx, id)
}

func (r *CachedCustomerRepository) Save(ctx context.Context, c *domaincustomer.Customer) error {
	if err := r.inner.Save(ctx, c); err != nil {
		return err
	}
	value, err := encode(customerToWire(c))
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, customerKeyPrefix+c.ID().String(), value, r.ttl)
}

func customerToWire(c *domaincustomer.Customer) customerWire {
	name := c.Name()
	return customerWire{
		First:     name.First(),
		Last:      name.Last(),
		Email:     c.Email().String(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

func customerFromWire(value string, id domaincustomer.CustomerID) (*domaincustomer.Customer, error) {
	var w customerWire
	if err := decode(value, &w); err != nil {
		return nil, err
	}
	return domaincustomer.RestoreCustomer(
		id,
		domaincustomer.NewCustomerName(w.First, w.Last),
		domaincustomer.NewEmail(w.Email),
		w.CreatedAt,
		w.UpdatedAt,
	), nil
}
