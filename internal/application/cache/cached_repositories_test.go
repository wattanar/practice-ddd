package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincustomer "practice-ddd/internal/domain/customer"
	domainorder "practice-ddd/internal/domain/order"
	domainproduct "practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

type fakeCache struct {
	values map[string]string
	down   bool
}

func (f *fakeCache) Get(ctx context.Context, key string) (string, error) {
	if f.down {
		return "", errors.New("redis down")
	}
	v, ok := f.values[key]
	if !ok {
		return "", ErrMiss
	}
	return v, nil
}

func (f *fakeCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	f.values[key] = value
	return nil
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

type memProductRepo struct {
	products map[string]*domainproduct.Product
	all      []*domainproduct.Product
}

func (r *memProductRepo) FindByID(ctx context.Context, id domainproduct.ProductID) (*domainproduct.Product, error) {
	p, ok := r.products[id.String()]
	if !ok {
		return nil, domainproduct.ErrProductNotFound
	}
	return p, nil
}

func (r *memProductRepo) FindAll(ctx context.Context) ([]*domainproduct.Product, error) {
	return r.all, nil
}

func (r *memProductRepo) Save(ctx context.Context, p *domainproduct.Product) error {
	r.products[p.ID().String()] = p
	return nil
}

type memCustomerRepo struct {
	customers map[string]*domaincustomer.Customer
}

func (r *memCustomerRepo) FindByID(ctx context.Context, id domaincustomer.CustomerID) (*domaincustomer.Customer, error) {
	c, ok := r.customers[id.String()]
	if !ok {
		return nil, domaincustomer.ErrCustomerNotFound
	}
	return c, nil
}

func (r *memCustomerRepo) Save(ctx context.Context, c *domaincustomer.Customer) error {
	r.customers[c.ID().String()] = c
	return nil
}

func testOrder() *domainorder.Order {
	addr, _ := domainorder.NewAddress("1 Main St", "Springfield", "IL", "62704", "US")
	o, _ := domainorder.NewOrder(domainorder.NewOrderID("o1"), domaincustomer.NewCustomerID("c1"), addr)
	price, _ := shared.NewMoney(1000, "USD")
	_ = o.AddItem(domainorder.NewOrderItemID("i1"), domainproduct.NewProductID("p1"), 2, price)
	return o
}

func TestCachedOrderRepository_ReadsFromCacheAfterSave(t *testing.T) {
	ctx := context.Background()
	db := &memOrderRepo{orders: map[string]*domainorder.Order{}}
	fc := &fakeCache{values: map[string]string{}}
	repo := NewCachedOrderRepository(db, fc, time.Minute)

	order := testOrder()
	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("save: %v", err)
	}

	stale, _ := domainorder.NewOrder(domainorder.NewOrderID("o1"), domaincustomer.NewCustomerID("c1"), mustAddress(t))
	_, _ = stale.Cancel()
	db.orders["o1"] = stale

	got, err := repo.FindByID(ctx, domainorder.NewOrderID("o1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Status() != domainorder.OrderStatusPending {
		t.Errorf("status = %q, want pending", got.Status())
	}
	if got.Total().Amount() != 2000 {
		t.Errorf("total = %d, want 2000", got.Total().Amount())
	}
	if len(got.Items()) != 1 || got.Items()[0].Quantity() != 2 {
		t.Errorf("items = %+v, want 1 item qty 2", got.Items())
	}
}

func TestCachedOrderRepository_FallsBackToDBWhenCacheDown(t *testing.T) {
	ctx := context.Background()
	order := testOrder()
	db := &memOrderRepo{orders: map[string]*domainorder.Order{"o1": order}}
	fc := &fakeCache{values: map[string]string{}, down: true}
	repo := NewCachedOrderRepository(db, fc, time.Minute)

	got, err := repo.FindByID(ctx, domainorder.NewOrderID("o1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Total().Amount() != 2000 {
		t.Errorf("total = %d, want 2000", got.Total().Amount())
	}
}

func TestCachedOrderRepository_FallsBackToDBOnCorruptEntry(t *testing.T) {
	ctx := context.Background()
	order := testOrder()
	db := &memOrderRepo{orders: map[string]*domainorder.Order{"o1": order}}
	fc := &fakeCache{values: map[string]string{"order:o1": "{not json"}}
	repo := NewCachedOrderRepository(db, fc, time.Minute)

	got, err := repo.FindByID(ctx, domainorder.NewOrderID("o1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Total().Amount() != 2000 {
		t.Errorf("total = %d, want 2000", got.Total().Amount())
	}
}

func TestCachedOrderRepository_SaveUpdatesCacheValue(t *testing.T) {
	ctx := context.Background()
	order := testOrder()
	db := &memOrderRepo{orders: map[string]*domainorder.Order{}}
	fc := &fakeCache{values: map[string]string{}}
	repo := NewCachedOrderRepository(db, fc, time.Minute)

	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("save: %v", err)
	}
	_, _ = order.Place()
	if err := repo.Save(ctx, order); err != nil {
		t.Fatalf("save: %v", err)
	}

	var decoded orderWire
	if err := decode(fc.values["order:o1"], &decoded); err != nil {
		t.Fatalf("decode cache value: %v", err)
	}
	if decoded.Status != "placed" {
		t.Errorf("cached status = %q, want placed", decoded.Status)
	}
}

func TestCachedProductRepository_ReadsFromCacheAfterSave(t *testing.T) {
	ctx := context.Background()
	price, _ := shared.NewMoney(2500, "USD")
	p, _ := domainproduct.NewProduct(domainproduct.NewProductID("p1"), "Widget", price, 10)
	db := &memProductRepo{products: map[string]*domainproduct.Product{}}
	fc := &fakeCache{values: map[string]string{}}
	repo := NewCachedProductRepository(db, fc, time.Minute)

	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("save: %v", err)
	}
	delete(db.products, "p1")

	got, err := repo.FindByID(ctx, domainproduct.NewProductID("p1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name() != "Widget" || got.Stock() != 10 || got.Price().Amount() != 2500 {
		t.Errorf("got %s stock=%d price=%d, want Widget 10 2500", got.Name(), got.Stock(), got.Price().Amount())
	}
}

func TestCachedProductRepository_FindAllBypassesCache(t *testing.T) {
	ctx := context.Background()
	p1, _ := domainproduct.NewProduct(domainproduct.NewProductID("p1"), "A", money(1, "USD"), 1)
	p2, _ := domainproduct.NewProduct(domainproduct.NewProductID("p2"), "B", money(2, "USD"), 2)
	db := &memProductRepo{
		products: map[string]*domainproduct.Product{"p1": p1, "p2": p2},
		all:      []*domainproduct.Product{p1, p2},
	}
	fc := &fakeCache{values: map[string]string{}}
	repo := NewCachedProductRepository(db, fc, time.Minute)

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) != 2 || all[1].Name() != "B" {
		t.Errorf("got %d products, want 2", len(all))
	}
}

func TestCachedCustomerRepository_ReadsFromCacheAfterSave(t *testing.T) {
	ctx := context.Background()
	c, _ := domaincustomer.NewCustomer(
		domaincustomer.NewCustomerID("c1"),
		domaincustomer.NewCustomerName("Ada", "Lovelace"),
		domaincustomer.NewEmail("ada@example.com"),
	)
	db := &memCustomerRepo{customers: map[string]*domaincustomer.Customer{}}
	fc := &fakeCache{values: map[string]string{}}
	repo := NewCachedCustomerRepository(db, fc, time.Minute)

	if err := repo.Save(ctx, c); err != nil {
		t.Fatalf("save: %v", err)
	}
	delete(db.customers, "c1")

	got, err := repo.FindByID(ctx, domaincustomer.NewCustomerID("c1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name().FullName() != "Ada Lovelace" || got.Email().String() != "ada@example.com" {
		t.Errorf("got %s %s, want Ada Lovelace ada@example.com", got.Name().FullName(), got.Email().String())
	}
}

func TestCachedCustomerRepository_FallsBackToDBWhenCacheDown(t *testing.T) {
	ctx := context.Background()
	c, _ := domaincustomer.NewCustomer(
		domaincustomer.NewCustomerID("c1"),
		domaincustomer.NewCustomerName("Ada", "Lovelace"),
		domaincustomer.NewEmail("ada@example.com"),
	)
	db := &memCustomerRepo{customers: map[string]*domaincustomer.Customer{"c1": c}}
	fc := &fakeCache{values: map[string]string{}, down: true}
	repo := NewCachedCustomerRepository(db, fc, time.Minute)

	got, err := repo.FindByID(ctx, domaincustomer.NewCustomerID("c1"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name().FullName() != "Ada Lovelace" {
		t.Errorf("name = %q, want Ada Lovelace", got.Name().FullName())
	}
}

func mustAddress(t *testing.T) domainorder.Address {
	t.Helper()
	addr, err := domainorder.NewAddress("1 Main St", "Springfield", "IL", "62704", "US")
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	return addr
}

func money(amount int64, currency string) shared.Money {
	m, err := shared.NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}
