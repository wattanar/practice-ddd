package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	appcache "practice-ddd/internal/application/cache"
	domaincustomer "practice-ddd/internal/domain/customer"
	domainorder "practice-ddd/internal/domain/order"
	domainproduct "practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

func TestRedisCache_MissReturnsErrMiss(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := NewRedisCache(client)

	_, err := c.Get(context.Background(), "order:o1")
	if !errors.Is(err, appcache.ErrMiss) {
		t.Fatalf("err = %v, want appcache.ErrMiss", err)
	}
}

func TestRedisCache_SetThenGetReturnsValueWithTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := NewRedisCache(client)
	ctx := context.Background()

	if err := c.Set(ctx, "order:o1", `{"id":"o1"}`, 60*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}
	value, err := c.Get(ctx, "order:o1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != `{"id":"o1"}` {
		t.Errorf("value = %q, want {\"id\":\"o1\"}", value)
	}
	ttl := mr.TTL("order:o1")
	if ttl <= 0 || ttl > 60*time.Second {
		t.Errorf("ttl = %v, want within (0, 60s]", ttl)
	}
}

type memOrderRepo struct {
	orders map[string]*domainorder.Order
}

func (r *memOrderRepo) FindByID(ctx context.Context, id domainorder.OrderID) (*domainorder.Order, error) {
	o, ok := r.orders[id.String()]
	if !ok {
		return nil, domainorder.ErrOrderNotFound
	}
	return o, nil
}

func (r *memOrderRepo) Save(ctx context.Context, o *domainorder.Order) error {
	r.orders[o.ID().String()] = o
	return nil
}

func TestRedisCache_WithCachedOrderRepository_RoundTripsThroughRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db := &memOrderRepo{orders: map[string]*domainorder.Order{}}
	repo := appcache.NewCachedOrderRepository(db, NewRedisCache(client), time.Minute)

	addr, _ := domainorder.NewAddress("1 Main St", "Springfield", "IL", "62704", "US")
	order, _ := domainorder.NewOrder(domainorder.NewOrderID("o1"), domaincustomer.NewCustomerID("c1"), addr)
	price, _ := shared.NewMoney(1000, "USD")
	if err := order.AddItem(domainorder.NewOrderItemID("i1"), domainproduct.NewProductID("p1"), 2, price); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("save: %v", err)
	}
	delete(db.orders, "o1")

	got, err := repo.FindByID(ctx, domainorder.NewOrderID("o1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Status() != domainorder.OrderStatusPending || got.Total().Amount() != 2000 {
		t.Errorf("status=%q total=%d, want pending 2000", got.Status(), got.Total().Amount())
	}
}

func TestRedisCache_ValueExpiresAfterTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := NewRedisCache(client)
	ctx := context.Background()

	if err := c.Set(ctx, "order:o1", "v", 1*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}
	mr.FastForward(2 * time.Second)

	_, err := c.Get(ctx, "order:o1")
	if !errors.Is(err, appcache.ErrMiss) {
		t.Fatalf("err = %v, want appcache.ErrMiss after expiry", err)
	}
}
