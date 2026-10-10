package cache

import (
	"context"
	"time"

	domainorder "practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

const orderKeyPrefix = "order:"

type CachedOrderRepository struct {
	inner domainorder.Repository
	cache Cache
	ttl   time.Duration
}

func NewCachedOrderRepository(inner domainorder.Repository, cache Cache, ttl time.Duration) domainorder.Repository {
	return &CachedOrderRepository{inner: inner, cache: cache, ttl: ttl}
}

type orderWire struct {
	ID            string
	CustomerID    string
	Status        string
	Line1         string
	City          string
	State         string
	Zip           string
	Country       string
	TotalAmount   int64
	TotalCurrency string
	Items         []orderItemWire
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type orderItemWire struct {
	ID                string
	ProductID         string
	Quantity          int
	UnitPriceAmount   int64
	UnitPriceCurrency string
}

func (r *CachedOrderRepository) FindByID(ctx context.Context, id domainorder.OrderID) (*domainorder.Order, error) {
	key := orderKeyPrefix + id.String()
	value, err := r.cache.Get(ctx, key)
	if err == nil {
		if order, derr := orderFromWire(value); derr == nil {
			return order, nil
		}
	}
	return r.inner.FindByID(ctx, id)
}

func (r *CachedOrderRepository) Save(ctx context.Context, o *domainorder.Order) error {
	if err := r.inner.Save(ctx, o); err != nil {
		return err
	}
	value, err := encode(orderToWire(o))
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, orderKeyPrefix+o.ID().String(), value, r.ttl)
}

func orderToWire(o *domainorder.Order) orderWire {
	addr := o.ShippingAddress()
	total := o.Total()
	items := make([]orderItemWire, 0, len(o.Items()))
	for _, it := range o.Items() {
		unit := it.UnitPrice()
		items = append(items, orderItemWire{
			ID:                it.ID().String(),
			ProductID:         it.ProductID().String(),
			Quantity:          it.Quantity(),
			UnitPriceAmount:   unit.Amount(),
			UnitPriceCurrency: unit.Currency(),
		})
	}
	return orderWire{
		ID:            o.ID().String(),
		CustomerID:    o.CustomerID().String(),
		Status:        string(o.Status()),
		Line1:         addr.Line1(),
		City:          addr.City(),
		State:         addr.State(),
		Zip:           addr.ZipCode(),
		Country:       addr.Country(),
		TotalAmount:   total.Amount(),
		TotalCurrency: total.Currency(),
		Items:         items,
		CreatedAt:     o.CreatedAt(),
		UpdatedAt:     o.UpdatedAt(),
	}
}

func orderFromWire(value string) (*domainorder.Order, error) {
	var w orderWire
	if err := decode(value, &w); err != nil {
		return nil, err
	}
	items := make([]*domainorder.OrderItem, 0, len(w.Items))
	for _, it := range w.Items {
		unit, err := shared.NewMoney(it.UnitPriceAmount, it.UnitPriceCurrency)
		if err != nil {
			return nil, err
		}
		items = append(items, domainorder.RestoreOrderItem(
			domainorder.NewOrderItemID(it.ID),
			product.NewProductID(it.ProductID),
			it.Quantity,
			unit,
		))
	}
	total, err := shared.NewMoney(w.TotalAmount, w.TotalCurrency)
	if err != nil {
		return nil, err
	}
	addr, err := domainorder.NewAddress(w.Line1, w.City, w.State, w.Zip, w.Country)
	if err != nil {
		return nil, err
	}
	return domainorder.RestoreOrder(
		domainorder.NewOrderID(w.ID),
		customer.NewCustomerID(w.CustomerID),
		items,
		domainorder.OrderStatus(w.Status),
		addr,
		total,
		w.CreatedAt,
		w.UpdatedAt,
	), nil
}
