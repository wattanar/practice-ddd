# Tutorial: Adding Promotions to the Order System

This tutorial walks through adding a promotion/discount feature to the DDD sample application. You'll see how each DDD concept guides implementation decisions.

## Business Requirement

Marketing wants promotions:

- Create promotions with a discount code (e.g., `WELCOME10` for 10% off, `FLAT500` for $5 off)
- Customers enter a code when ordering; the discount applies to the order total
- Promotions have a minimum order value to qualify
- Promotions can be activated or deactivated

```
POST /orders/{id}/apply-promotion  {"code": "WELCOME10"}
DELETE /orders/{id}/apply-promotion
POST /promotions                    {"code": ..., "type": "percentage", "value": 10}
```

## DDD Discovery — Before Writing Code

Ask DDD questions:

| Question | Answer |
|----------|--------|
| **Entity or Value Object?** | `Promotion` is an entity (has identity, lifecycle). Discount applied to an order is a value object (`AppliedDiscount`). |
| **New aggregate or part of existing?** | New `Promotion` aggregate — different lifecycle from Order, can exist independently. |
| **How does Order reference Promotion?** | By ID only (`PromotionID`). Order stores a snapshot of the discount as an `AppliedDiscount` value object. |
| **Where does discount calculation live?** | On the `Order` aggregate — it's order behavior that depends on order state (items, status). |
| **What invariants?** | Discount ≤ items total. Percentage must be 0–100. Promotion can only be applied to pending orders. |

## File Plan

| Step | Files | Layer |
|------|-------|-------|
| 1 | `internal/domain/promotion/` (6 files) | Domain — new aggregate |
| 2 | `internal/domain/order/value_objects.go` (amend) | Domain — add `AppliedDiscount` VO |
| 3 | `internal/domain/order/entity.go` (amend) | Domain — add `ApplyPromotion()`, `RemovePromotion()` |
| 4 | `internal/domain/order/events.go` (amend) | Domain — add `PromotionApplied` event |
| 5 | `internal/application/order/apply_promotion.go` | Application — new use case |
| 6 | `internal/application/order/ports.go` (amend) | Application — add `PromotionRepository` port |
| 7 | `internal/infrastructure/persistence/sqlite/promotion_repository.go` | Infrastructure — SQLite adapter |
| 8 | `internal/infrastructure/persistence/sqlite/db.go` (amend) | Infrastructure — migration |
| 9 | `internal/infrastructure/rest/handler.go` (amend) | Infrastructure — HTTP endpoints |
| 10 | `cmd/server/main.go` (amend) | Composition root — wire deps |
| 11 | Tests | All layers |

---

## Step 1: Create the Promotion Aggregate

### 1a. Value Objects

**`internal/domain/promotion/value_objects.go`**

Start with identity and value types. Value objects are immutable and self-validating — no identity, compared by value.

```go
package promotion

import "strings"

type PromotionID struct {
	value string
}

func NewPromotionID(value string) PromotionID {
	return PromotionID{value: value}
}

func (id PromotionID) String() string { return id.value }

type DiscountType string

const (
	DiscountTypePercentage DiscountType = "percentage"
	DiscountTypeFixed      DiscountType = "fixed"
)

type PromotionCode struct {
	value string
}

func NewPromotionCode(value string) PromotionCode {
	return PromotionCode{value: strings.ToUpper(strings.TrimSpace(value))}
}

func (c PromotionCode) String() string { return c.value }
```

**Why value objects?** `PromotionID` is comparable by value — two `PromotionID("abc")` are interchangeable. The constructor normalizes the code (`strings.ToUpper`), ensuring `WELCOME10` and `welcome10` match. Business rules live at creation time, not downstream.

### 1b. Entity

**`internal/domain/promotion/entity.go`**

The aggregate root. Enforces its own invariants: exact percentage range, non-negative fixed discount, date validity.

```go
package promotion

import (
	"errors"
	"time"
	"practice-ddd/internal/domain/shared"
)

var (
	ErrInvalidDiscountValue   = errors.New("percentage must be 1-100, fixed must be positive")
	ErrPromotionNotActive     = errors.New("promotion is not active")
	ErrMinOrderNotMet         = errors.New("order total does not meet minimum")
	ErrPromotionNotFound      = errors.New("promotion not found")
	ErrInvalidDiscountType    = errors.New("discount type must be 'percentage' or 'fixed'")
)

type Promotion struct {
	id             PromotionID
	code           PromotionCode
	discountType   DiscountType
	discountValue  int64
	minOrderAmount shared.Money
	active         bool
	createdAt      time.Time
	updatedAt      time.Time
}

func NewPromotion(
	id PromotionID, code PromotionCode,
	discountType DiscountType, discountValue int64,
	minOrderAmount shared.Money, active bool,
) (*Promotion, error) {
	if err := validateDiscount(discountType, discountValue); err != nil {
		return nil, err
	}
	now := time.Now()
	return &Promotion{
		id: id, code: code,
		discountType: discountType, discountValue: discountValue,
		minOrderAmount: minOrderAmount, active: active,
		createdAt: now, updatedAt: now,
	}, nil
}

func RestorePromotion(
	id PromotionID, code PromotionCode,
	discountType DiscountType, discountValue int64,
	minOrderAmount shared.Money, active bool,
	createdAt, updatedAt time.Time,
) *Promotion {
	return &Promotion{
		id: id, code: code,
		discountType: discountType, discountValue: discountValue,
		minOrderAmount: minOrderAmount, active: active,
		createdAt: createdAt, updatedAt: updatedAt,
	}
}

func validateDiscount(t DiscountType, v int64) error {
	switch t {
	case DiscountTypePercentage:
		if v < 1 || v > 100 {
			return ErrInvalidDiscountValue
		}
	case DiscountTypeFixed:
		if v <= 0 {
			return ErrInvalidDiscountValue
		}
	default:
		return ErrInvalidDiscountType
	}
	return nil
}

func (p *Promotion) ID() PromotionID           { return p.id }
func (p *Promotion) Code() PromotionCode        { return p.code }
func (p *Promotion) DiscountType() DiscountType  { return p.discountType }
func (p *Promotion) DiscountValue() int64        { return p.discountValue }
func (p *Promotion) MinOrderAmount() shared.Money { return p.minOrderAmount }
func (p *Promotion) IsActive() bool              { return p.active }
func (p *Promotion) CreatedAt() time.Time        { return p.createdAt }
func (p *Promotion) UpdatedAt() time.Time        { return p.updatedAt }

func (p *Promotion) Deactivate() {
	p.active = false
	p.updatedAt = time.Now()
}

func (p *Promotion) Activate() {
	p.active = true
	p.updatedAt = time.Now()
}
```

**Why entity?** `Promotion` has a unique `PromotionID` that persists across state changes (active → inactive). Two promotions with the same discount value are not interchangeable — they have different identities.

**Why validate at creation?** This prevents invalid promotions from existing in the system at all. The `RestorePromotion` factory bypasses validation for repository reconstitution (the data was already valid when created).

### 1c. Repository port

**`internal/domain/promotion/repository.go`**

The port lives in the domain, the adapter lives in infrastructure. This is the **driven port** pattern from Hexagonal Architecture.

```go
package promotion

import "context"

type Repository interface {
	FindByID(ctx context.Context, id PromotionID) (*Promotion, error)
	FindByCode(ctx context.Context, code PromotionCode) (*Promotion, error)
	Save(ctx context.Context, p *Promotion) error
	FindAll(ctx context.Context) ([]*Promotion, error)
}
```

**Why interface in domain?** The domain defines what it needs (a way to find promotions). The infrastructure decides how (SQLite, Postgres, in-memory). The `go test ./internal/domain/...` command runs with zero database dependencies.

### 1d. Errors

**`internal/domain/promotion/errors.go`**

```go
package promotion

import "errors"

var (
	ErrInvalidDiscountValue   = errors.New("percentage must be 1-100, fixed must be positive")
	ErrPromotionNotActive     = errors.New("promotion is not active")
	ErrMinOrderNotMet         = errors.New("order total does not meet minimum")
	ErrPromotionNotFound      = errors.New("promotion not found")
	ErrInvalidDiscountType    = errors.New("discount type must be 'percentage' or 'fixed'")
)
```

Sentinel errors let callers check specific failure reasons without parsing strings. The application layer maps these to HTTP status codes.

---

## Step 2: Add Discount to the Order Aggregate

The Order aggregate needs to store an optional discount and calculate the final total.

### 2a. Discount Value Object

**`internal/domain/order/value_objects.go`** — add after `Address`:

```go
// AppliedDiscount is a snapshot of a promotion applied to an order.
// It is a Value Object — identity comes from the Order, not the discount itself.
type AppliedDiscount struct {
	promotionID      string
	code             string
	discountType     DiscountType
	discountValue    int64           // percentage (e.g. 10) or cents (e.g. 5000)
	calculatedAmount shared.Money    // the actual cents off
}

func NewAppliedDiscount(
	promotionID, code string,
	discountType DiscountType, discountValue int64,
	calculatedAmount shared.Money,
) AppliedDiscount {
	return AppliedDiscount{
		promotionID: promotionID, code: code,
		discountType: discountType, discountValue: discountValue,
		calculatedAmount: calculatedAmount,
	}
}

func (d AppliedDiscount) PromotionID() string           { return d.promotionID }
func (d AppliedDiscount) Code() string                  { return d.code }
func (d AppliedDiscount) DiscountType() DiscountType    { return d.discountType }
func (d AppliedDiscount) DiscountValue() int64          { return d.discountValue }
func (d AppliedDiscount) CalculatedAmount() shared.Money { return d.calculatedAmount }
```

**Why value object?** `AppliedDiscount` has no independent identity — it's defined entirely by its fields. Two discounts with the same promotionID, type, value, and calculated amount are the same discount. And it's stored directly on Order, not in its own table.

### 2b. Order Entity Changes

**`internal/domain/order/entity.go`** — changes:

1. Add `appliedDiscount` field and `itemsSubtotal` field to `Order` struct
2. Add `ApplyPromotion()` and `RemovePromotion()` methods
3. Update `recalculateTotal()` to account for discount
4. Update `Place()` to include discount info in event

```go
// New field on the Order struct (add after the items field):
type Order struct {
	id             OrderID
	customerID     customer.CustomerID
	items          []*OrderItem
	appliedDiscount *AppliedDiscount   // nil if no discount applied
	status         OrderStatus
	shippingAddr   Address
	total          shared.Money
	createdAt      time.Time
	updatedAt      time.Time
}
```

Adding methods:

```go
func (o *Order) ApplyPromotion(promo *promotion.Promotion) ([]DomainEvent, error) {
	if o.status != OrderStatusPending {
		return nil, ErrCannotModifyPlacedOrder
	}
	if !promo.IsActive() {
		return nil, promotion.ErrPromotionNotActive
	}
	if o.total.LessThan(promo.MinOrderAmount()) {
		return nil, promotion.ErrMinOrderNotMet
	}

	var calcAmount int64
	switch promo.DiscountType() {
	case promotion.DiscountTypePercentage:
		calcAmount = o.total.Amount() * promo.DiscountValue() / 100
		if calcAmount > o.total.Amount() {
			calcAmount = o.total.Amount()
		}
	case promotion.DiscountTypeFixed:
		calcAmount = promo.DiscountValue()
		if calcAmount > o.total.Amount() {
			calcAmount = o.total.Amount()
		}
	}

	calculated, _ := shared.NewMoney(calcAmount, o.total.Currency())

	o.appliedDiscount = &AppliedDiscount{
		promotionID:      promo.ID().String(),
		code:             promo.Code().String(),
		discountType:     DiscountType(promo.DiscountType()),
		discountValue:    promo.DiscountValue(),
		calculatedAmount: calculated,
	}
	o.recalculateTotal()
	o.updatedAt = time.Now()

	return []DomainEvent{PromotionApplied{
		OrderID:         o.id,
		PromotionID:     promo.ID(),
		DiscountAmount:  calculated,
		AppliedAt:       o.updatedAt,
	}}, nil
}

func (o *Order) RemovePromotion() {
	if o.appliedDiscount == nil {
		return
	}
	o.appliedDiscount = nil
	o.recalculateTotal()
	o.updatedAt = time.Now()
}
```

Updated `recalculateTotal()`:

```go
func (o *Order) recalculateTotal() {
	zero, _ := shared.NewMoney(0, "USD")
	total := zero
	for _, item := range o.items {
		total, _ = total.Add(item.Total())
	}
	if o.appliedDiscount != nil {
		total, _ = total.Subtract(o.appliedDiscount.calculatedAmount)
	}
	o.total = total
}
```

**Why behavior on Order?** `ApplyPromotion` is a behavior of the Order aggregate — it reads order state (status, total) and enforces invariants (only pending orders, minimum amount). Putting this logic on the Order entity keeps it with the data it operates on, not scattered across services.

**Why return events?** The method returns `[]DomainEvent` instead of publishing directly. This keeps the domain side-effect-free and testable. The application layer collects and publishes events after persisting.

**What about the `PromotionApplied` event?** It's added to `events.go` in the order package so the order can reference it:

```go
type PromotionApplied struct {
	OrderID        OrderID
	PromotionID    promotion.PromotionID
	DiscountAmount shared.Money
	AppliedAt      time.Time
}

func (e PromotionApplied) EventName() string       { return "order.promotion_applied" }
func (e PromotionApplied) OccurredAt() time.Time   { return e.AppliedAt }
```

**Key DDD insight:** The event lives in the ORDER package, not the promotion package, because it's emitted BY the Order aggregate. The promotion package doesn't know about orders — that's the right dependency direction.

---

## Step 3: Create the Application Use Case

### 3a. Add PromotionRepository to Application Ports

**`internal/application/order/ports.go`** — add:

```go
type PromotionRepository interface {
	FindByCode(ctx context.Context, code string) (*promotion.Promotion, error)
}
```

This is a driven port — defined where it's needed (application), implemented in infrastructure. The application layer sees only the methods it needs (`FindByCode`), not the full Promotion repository interface.

### 3b. ApplyPromotion Handler

**`internal/application/order/apply_promotion.go`**

The use case orchestrates: load order → load promotion → call domain → save → publish events.

```go
package order

import (
	"context"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/promotion"
)

type ApplyPromotionRequest struct {
	OrderID string
	Code    string
}

type ApplyPromotionHandler struct {
	orders     order.Repository
	promotions PromotionRepository
	events     EventBus
}

func NewApplyPromotionHandler(
	orders order.Repository,
	promotions PromotionRepository,
	events EventBus,
) *ApplyPromotionHandler {
	return &ApplyPromotionHandler{
		orders: orders, promotions: promotions, events: events,
	}
}

func (h *ApplyPromotionHandler) Handle(ctx context.Context, req ApplyPromotionRequest) error {
	ord, err := h.orders.FindByID(ctx, order.NewOrderID(req.OrderID))
	if err != nil {
		return err
	}

	promo, err := h.promotions.FindByCode(ctx, req.Code)
	if err != nil {
		return err
	}

	domainEvents, err := ord.ApplyPromotion(promo)
	if err != nil {
		return err
	}

	if err := h.orders.Save(ctx, ord); err != nil {
		return err
	}

	for _, e := range domainEvents {
		if err := h.events.Publish(ctx, e); err != nil {
			return err
		}
	}

	return nil
}
```

And the corresponding `RemovePromotionHandler`:

```go
type RemovePromotionRequest struct {
	OrderID string
}

type RemovePromotionHandler struct {
	orders order.Repository
	events EventBus
}

func NewRemovePromotionHandler(orders order.Repository, events EventBus) *RemovePromotionHandler {
	return &RemovePromotionHandler{orders: orders, events: events}
}

func (h *RemovePromotionHandler) Handle(ctx context.Context, req RemovePromotionRequest) error {
	ord, err := h.orders.FindByID(ctx, order.NewOrderID(req.OrderID))
	if err != nil {
		return err
	}
	ord.RemovePromotion()
	return h.orders.Save(ctx, ord)
}
```

**Why separate handlers?** Each use case is a single struct with a single method. This makes dependencies explicit (you see exactly what each use case needs), testable (mock only what's needed), and composable (mix and match in composition root).

**What's NOT in the handler:** Business logic. The handler doesn't calculate discounts or check promotion validity — it delegates to `Order.ApplyPromotion()`. This is the defining characteristic of an anemic vs. rich domain model.

---

## Step 4: Add Persistence

### 4a. Migration

**`internal/infrastructure/persistence/sqlite/db.go`** — add to schema:

```sql
CREATE TABLE IF NOT EXISTS promotions (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    discount_type TEXT NOT NULL,
    discount_value INTEGER NOT NULL,
    min_order_amount INTEGER NOT NULL,
    min_order_currency TEXT NOT NULL DEFAULT 'USD',
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

Also add discount columns to orders:

No — since `AppliedDiscount` is a value object stored within the `Order` aggregate, it goes into the `orders` table as nullable columns:

```sql
ALTER TABLE orders ADD COLUMN discount_promotion_id TEXT;
ALTER TABLE orders ADD COLUMN discount_code TEXT;
ALTER TABLE orders ADD COLUMN discount_type TEXT;
ALTER TABLE orders ADD COLUMN discount_value INTEGER;
ALTER TABLE orders ADD COLUMN discount_calculated_amount INTEGER;
```

**Why denormalized into orders table?** `AppliedDiscount` is a value object on the Order aggregate. It's not a separate entity — it has no independent lifecycle and is always accessed through an Order. Storing it directly on the `orders` row keeps the aggregate boundary intact and avoids cross-table transactions.

### 4b. SQLite Repository Implementation

**`internal/infrastructure/persistence/sqlite/promotion_repository.go`**

```go
package sqlite

import (
	"context"
	"database/sql"
	"time"
	"practice-ddd/internal/domain/promotion"
	"practice-ddd/internal/domain/shared"
)

type PromotionRepository struct {
	db *sql.DB
}

func NewPromotionRepository(db *sql.DB) *PromotionRepository {
	return &PromotionRepository{db: db}
}

func scanPromotion(row *sql.Row) (*promotion.Promotion, error) {
	var (
		id, code, discountType, currency, createdAt, updatedAt string
		discountValue, minAmount                               int64
		active                                                 bool
	)
	if err := row.Scan(&id, &code, &discountType, &discountValue,
		&minAmount, &currency, &active, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, promotion.ErrPromotionNotFound
		}
		return nil, err
	}
	minMoney, _ := shared.NewMoney(minAmount, currency)
	created, _ := time.Parse(time.RFC3339, createdAt)
	updated, _ := time.Parse(time.RFC3339, updatedAt)
	return promotion.RestorePromotion(
		promotion.NewPromotionID(id),
		promotion.NewPromotionCode(code),
		promotion.DiscountType(discountType),
		discountValue, minMoney, active, created, updated,
	), nil
}

func (r *PromotionRepository) FindByCode(ctx context.Context, code promotion.PromotionCode) (*promotion.Promotion, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, code, discount_type, discount_value,
		        min_order_amount, min_order_currency, active, created_at, updated_at
		 FROM promotions WHERE code = ?`, code.String())
	return scanPromotion(row)
}

func (r *PromotionRepository) Save(ctx context.Context, p *promotion.Promotion) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO promotions (id, code, discount_type, discount_value,
		                         min_order_amount, min_order_currency, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			code = excluded.code, active = excluded.active, updated_at = excluded.updated_at`,
		p.ID().String(), p.Code().String(), string(p.DiscountType()),
		p.DiscountValue(), p.MinOrderAmount().Amount(), p.MinOrderAmount().Currency(),
		p.IsActive(), p.CreatedAt().Format(time.RFC3339), p.UpdatedAt().Format(time.RFC3339),
	)
	return err
}
```

### 4c. Update OrderRepository for Discount

The `FindByID` and `Save` methods in `internal/infrastructure/persistence/sqlite/order_repository.go` need to read/write the discount columns. In `FindByID`, scan the new columns and reconstruct `AppliedDiscount`. In `Save`, write the discount fields when present.

---

## Step 5: Wire Dependencies

### Composition Root

**`cmd/server/main.go`** — add the promotion repository and new handlers:

```go
promotionRepo := sqlite.NewPromotionRepository(db)

applyPromotion := apporder.NewApplyPromotionHandler(orderRepo, promotionRepo, eventBus)
removePromotion := apporder.NewRemovePromotionHandler(orderRepo, eventBus)

srv := rest.NewServer(
    placeOrder, cancelOrder, shipOrder,
    applyPromotion, removePromotion,
    productRepo, customerRepo, promotionRepo,
)
```

The composition root is the only place where dependency injection happens. Domain and application packages never call `New*Repository` or reference `sqlite.*` — they only see interfaces.

---

## Step 6: Expose via HTTP

### Route Registration

Add to `Router()` in `internal/infrastructure/rest/handler.go`:

```go
r.POST("/orders/:id/apply-promotion", s.handleApplyPromotion)
r.DELETE("/orders/:id/apply-promotion", s.handleRemovePromotion)
r.GET("/promotions", s.handleListPromotions)
r.POST("/promotions", s.handleCreatePromotion)
```

### Handler Implementation

```go
func (s *Server) handleApplyPromotion(c *gin.Context) {
    id := c.Param("id")
    var req struct {
        Code string `json:"code" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := s.applyPromotion.Handle(c.Request.Context(),
        apporder.ApplyPromotionRequest{OrderID: id, Code: req.Code}); err != nil {
        c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "promotion_applied"})
}
```

---

## Step 7: Write Tests

### 7a. Domain Tests — Pure Unit Tests (Fastest)

**`internal/domain/promotion/entity_test.go`**

```go
func TestNewPromotion_Percentage(t *testing.T) {
    p, err := promotion.NewPromotion(
        promotion.NewPromotionID("p1"),
        promotion.NewPromotionCode("SAVE10"),
        promotion.DiscountTypePercentage, 10,
        mustMoney(0, "USD"), true,
    )
    if err != nil {
        t.Fatal(err)
    }
    if p.Code().String() != "SAVE10" {
        t.Fatalf("expected SAVE10, got %s", p.Code().String())
    }
}

func TestNewPromotion_InvalidPercentage(t *testing.T) {
    _, err := promotion.NewPromotion(
        promotion.NewPromotionID("p2"),
        promotion.NewPromotionCode("INVALID"),
        promotion.DiscountTypePercentage, 200, // > 100
        mustMoney(0, "USD"), true,
    )
    if err != promotion.ErrInvalidDiscountValue {
        t.Fatal("expected ErrInvalidDiscountValue")
    }
}
```

**`internal/domain/order/entity_test.go`** — add:

```go
func TestApplyPromotion(t *testing.T) {
    ord := newTestOrder(t)
    ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 2, mustMoney(1000, "USD"))
    // ord.Total = 2000

    promo := testPromotion(t, promotion.DiscountTypePercentage, 10)
    events, err := ord.ApplyPromotion(promo)
    if err != nil {
        t.Fatal(err)
    }
    // Discount: 2000 * 10% = 200
    // Total: 2000 - 200 = 1800
    if ord.Total().Amount() != 1800 {
        t.Fatalf("expected total 1800, got %d", ord.Total().Amount())
    }
    if len(events) != 1 {
        t.Fatalf("expected 1 event, got %d", len(events))
    }
}
```

**Why domain tests first?** These tests exercise pure business logic with zero infrastructure — no database, no HTTP, no mocks. They run in milliseconds. When a test fails, the bug is in the business logic, not in configuration.

### 7b. Application Tests — With Mocks

**`internal/application/order/apply_promotion_test.go`**

```go
func TestApplyPromotionHandler(t *testing.T) {
    // Set up mocks
    orders := &mockOrderRepo{...}
    promos := &mockPromotionRepo{...}
    events := &mockEventBus{}

    handler := NewApplyPromotionHandler(orders, promos, events)
    err := handler.Handle(ctx, ApplyPromotionRequest{OrderID: "o1", Code: "SAVE10"})
    // assert: order saved, event published
}
```

---

## Step 8: Run and Verify

```bash
# Create a promotion
curl -X POST :8080/promotions -d '{
    "code": "WELCOME10",
    "type": "percentage",
    "value": 10
}'

# List promotions
curl :8080/promotions

# Place an order and get the order ID
curl -X POST :8080/orders -d '{
    "customer_id": "<alice-id>",
    "items": [{"product_id": "<laptop-id>", "quantity": 1}],
    "shipping_address": {
        "line1": "123 Main St", "city": "Portland",
        "state": "OR", "zip_code": "97201", "country": "US"
    }
}'

# Apply promotion
curl -X POST :8080/orders/<order-id>/apply-promotion -d '{"code": "WELCOME10"}'

# Run all tests
go test ./...
```

---

## Recap: DDD Principles Applied

| Step | DDD Concept | What We Did |
|------|-------------|-------------|
| 1 | **Aggregate** | Created `Promotion` as its own aggregate with its own identity and lifecycle |
| 1 | **Value Object** | `PromotionID`, `PromotionCode`, `DiscountType` — immutable, self-validating |
| 2 | **Rich Domain Model** | `Order.ApplyPromotion()` encapsulates business rules (status check, minimum amount, calculation) |
| 2 | **Aggregate Boundary** | `AppliedDiscount` is a value object inside Order — accessed only through Order |
| 2 | **Domain Event** | `PromotionApplied` emitted by the aggregate, published by the application layer |
| 3 | **Application Service** | `ApplyPromotionHandler` orchestrates — no business logic, just coordination |
| 4 | **Repository (Port)** | Interface in domain, implementation in infrastructure |
| 5 | **Composition Root** | `main.go` wires concrete implementations; inner layers know only interfaces |
| 7 | **Test Pyramid** | Domain tests (fast, no infra) > Application tests (mocked) > Integration tests |

## Anti-Patterns We Avoided

| Anti-Pattern | What We Did Instead |
|-------------|-------------------|
| **Anemic Domain Model** | Discount calculation lives on `Order`, not in the handler |
| **Leaking Infrastructure** | Domain imports zero external packages — no `sql`, no `gin` |
| **Cross-Aggregate Transaction** | Order stores a snapshot (`AppliedDiscount`) instead of locking the Promotion row |
| **Repository per Table** | One repository per aggregate (`PromotionRepository`, `OrderRepository`) |
| **Service-Layer Logic** | The handler calls `ord.ApplyPromotion()` — it doesn't compute the discount itself |
