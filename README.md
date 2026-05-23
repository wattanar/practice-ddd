# DDD Sample: E-Commerce Order Management

A reference application demonstrating **Domain-Driven Design** tactical patterns in Go, with Clean Architecture and Hexagonal (Ports & Adapters) style layering.

## Architecture

```
┌──────────────────────────────────────────────┐
│              Infrastructure                   │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │   REST   │  │  SQLite   │  │ Event Bus  │  │
│  │ Handlers │  │  Repos    │  │ (console)  │  │
│  └────┬─────┘  └────┬─────┘  └──────┬─────┘  │
├───────┼──────────────┼───────────────┼────────┤
│       ▼              ▼               ▼        │
│            Application Layer                   │
│  ┌─────────────────────────────────────────┐  │
│  │ PlaceOrder │ CancelOrder │ ShipOrder    │  │
│  └─────────────────────────────────────────┘  │
├───────────────────────────────────────────────┤
│              Domain Layer                      │
│  ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │  Order   │ │ Product  │ │   Customer    │  │
│  │ Aggregate│ │Aggregate │ │   Aggregate   │  │
│  └──────────┘ └──────────┘ └───────────────┘  │
│  ┌─────────────────────────────────────────┐  │
│  │   Shared: Money, DomainEvent            │  │
│  └─────────────────────────────────────────┘  │
└───────────────────────────────────────────────┘
```

**Three layers, one rule: dependencies point inward.** Domain imports nothing external. Application imports only domain. Infrastructure implements ports defined by inner layers.

## DDD Concepts Demonstrated

### 1. Entities — Identity + Behavior

Entities have a unique identity that persists across time and state changes.

```go
// Order is an Entity — identity is OrderID, not its field values
type Order struct {
    id         OrderID
    status     OrderStatus
    items      []*OrderItem
    total      shared.Money
    // ...
}

// Behavior lives on the entity, enforcing invariants
func (o *Order) Cancel() ([]DomainEvent, error) {
    if !o.status.CanTransitionTo(OrderStatusCancelled) {
        return nil, errors.New("cannot cancel order from current status")
    }
    o.status = OrderStatusCancelled
    o.updatedAt = time.Now()
    return []DomainEvent{OrderCancelled{...}}, nil
}
```

**Key files:**
- `internal/domain/order/entity.go:60` — Order entity with behavior
- `internal/domain/order/entity.go:19` — OrderItem child entity

### 2. Value Objects — Immutable, Self-Validating

Value objects have no identity — two `Money(100, "USD")` are interchangeable. They are immutable and self-validating.

```go
type Money struct {
    amount   int64   // cents (unexported — no direct mutation)
    currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
    if amount < 0 { return Money{}, ErrNegativeAmount }
    if currency == "" { return Money{}, ErrInvalidCurrency }
    return Money{amount: amount, currency: currency}, nil
}

func (m Money) Add(o Money) (Money, error) {
    if m.currency != o.currency { return Money{}, ErrCurrencyMismatch }
    return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}

func (m Money) Multiply(factor int) Money {
    return Money{amount: m.amount * int64(factor), currency: m.currency}
}
```

**Key files:**
- `internal/domain/shared/value_objects.go` — Money (shared across aggregates)
- `internal/domain/order/value_objects.go` — OrderID, OrderStatus, Address

### 3. Aggregates — Consistency Boundaries

An aggregate is a cluster of domain objects treated as a single unit. The **aggregate root** is the only entry point — external code never holds a reference to child entities directly.

```
┌───────────────────────────────────────┐
│            Order Aggregate            │
│                                       │
│  Order (Root) ──── OrderItem         │
│  - status        - productID          │
│  - total         - quantity           │
│  - shippingAddr  - unitPrice          │
│                                       │
│  Invariants enforced by root:         │
│  • Must have ≥1 item to place         │
│  • Status transitions are validated   │
│  • Total always equals sum of items   │
└───────────────────────────────────────┘
```

Rules:
- One aggregate per transaction
- Other aggregates reference by ID only (Order stores `customer.CustomerID`, not a `*Customer`)
- Repository per aggregate, not per table

**Key file:** `internal/domain/order/entity.go:60`

### 4. Domain Events — Recording What Happened

Behavior methods return events describing what changed. The application layer publishes them.

```go
func (o *Order) Place() ([]DomainEvent, error) {
    // ... validate, mutate state ...
    return []DomainEvent{OrderPlaced{
        OrderID:    o.id,
        CustomerID: o.customerID,
        Total:      o.total,
        PlacedAt:   o.updatedAt,
    }}, nil
}
```

Flow:
```
Domain method returns []DomainEvent
    → Application handler saves aggregate
        → Application handler publishes events via EventBus port
```

**Key files:**
- `internal/domain/order/events.go` — DomainEvent interface + concrete events
- `internal/domain/order/entity.go:133` — Place() returns OrderPlaced
- `internal/domain/order/entity.go:150` — Cancel() returns OrderCancelled
- `cmd/server/main.go:20` — ConsoleEventBus implementation

### 5. Repositories — Persistence Abstraction (Driven Port)

Repository interfaces live in the domain layer. Implementations live in infrastructure.

```go
// Domain defines the contract (port)
type Repository interface {
    FindByID(ctx context.Context, id OrderID) (*Order, error)
    Save(ctx context.Context, order *Order) error
}

// Infrastructure implements it (adapter)
type OrderRepository struct { db *sql.DB }

func (r *OrderRepository) Save(ctx context.Context, ord *Order) error {
    // SQLite INSERT ... ON CONFLICT DO UPDATE
}
```

**Key files:**
- `internal/domain/order/repository.go` — Port (interface)
- `internal/infrastructure/persistence/sqlite/order_repository.go` — Adapter (implementation)

### 6. Application Services — Orchestration, Not Logic

Use cases orchestrate domain objects and infrastructure. They contain **zero business logic**.

```go
func (h *PlaceOrderHandler) Handle(ctx context.Context, req PlaceOrderRequest) (*order.Order, error) {
    // 1. Look up dependencies
    cust, _ := h.customers.FindByID(ctx, custID)
    prod, _ := h.products.FindByID(ctx, prodID)

    // 2. Delegate to domain
    ord, _ := order.NewOrder(id, custID, addr)
    ord.AddItem(itemID, prodID, qty, prod.Price())

    // 3. Let domain enforce rules
    events, _ := ord.Place()

    // 4. Persist + publish
    h.orders.Save(ctx, ord)
    h.events.Publish(ctx, events)
    return ord, nil
}
```

**Key files:**
- `internal/application/order/place_order.go` — PlaceOrder use case
- `internal/application/order/cancel_order.go` — CancelOrder
- `internal/application/order/ship_order.go` — ShipOrder
- `internal/application/order/ports.go` — EventBus port

### 7. The Dependency Rule (No Leaking)

Domain layer packages import **only the Go standard library**:

```
$ head -1 internal/domain/order/entity.go
package order

$ grep '"' internal/domain/order/*.go | head -5
  "errors"
  "time"
  "practice-ddd/internal/domain/customer"
  "practice-ddd/internal/domain/product"
  "practice-ddd/internal/domain/shared"
```

Notice: domain-to-domain imports are fine (Order imports CustomerID from customer package). What's **absent**: no `database/sql`, no HTTP libraries, no framework imports.

## Project Structure

```
practice-ddd/
├── cmd/server/main.go                      # Composition root
├── internal/
│   ├── domain/                             # Core business logic
│   │   ├── shared/value_objects.go         # Money (shared VO)
│   │   ├── product/                        # Product aggregate
│   │   │   ├── entity.go
│   │   │   ├── value_objects.go
│   │   │   ├── errors.go
│   │   │   └── repository.go              # ProductRepository port
│   │   ├── customer/                       # Customer aggregate
│   │   │   ├── entity.go
│   │   │   ├── value_objects.go
│   │   │   ├── errors.go
│   │   │   └── repository.go
│   │   └── order/                          # Order aggregate (core)
│   │       ├── entity.go                  # Order + OrderItem
│   │       ├── value_objects.go           # OrderID, OrderStatus, Address
│   │       ├── events.go                  # Domain events
│   │       ├── errors.go
│   │       └── repository.go              # OrderRepository port
│   ├── application/order/                  # Use cases
│   │   ├── ports.go                       # EventBus port
│   │   ├── place_order.go                 # PlaceOrder use case
│   │   ├── cancel_order.go                # CancelOrder use case
│   │   └── ship_order.go                  # ShipOrder use case
│   └── infrastructure/                     # Adapters
│       ├── persistence/sqlite/
│       │   ├── db.go                      # DB connection + migration
│       │   ├── order_repository.go
│       │   ├── product_repository.go
│       │   └── customer_repository.go
│       └── rest/handler.go                # HTTP handlers
├── migrations/
└── go.mod
```

## Domain Model

### Order Status Machine

```
pending ──► placed ──► shipped ──► delivered
   │            │
   └──► cancelled └──► cancelled
```

Defined in `internal/domain/order/value_objects.go:33`:

```go
var validTransitions = map[OrderStatus][]OrderStatus{
    OrderStatusPending:   {OrderStatusPlaced, OrderStatusCancelled},
    OrderStatusPlaced:    {OrderStatusShipped, OrderStatusCancelled},
    OrderStatusShipped:   {OrderStatusDelivered},
    OrderStatusDelivered: {},
    OrderStatusCancelled: {},
}
```

### Aggregate Cross-References

Aggregates reference each other **by ID only**, never by object reference:

| Aggregate | References |
|-----------|-----------|
| Order | `CustomerID`, `ProductID` |
| Product | — |
| Customer | — |

## Testing Strategy

```
Tests by Layer:
Domain:       30 unit tests (no infra, pure logic)
Application:   2 unit tests (mocked repositories)
Infrastructure: 7 integration tests (real SQLite)
```

### Domain Tests

Test entities without any infrastructure. No database, no HTTP, no mocks.

**File:** `internal/domain/order/entity_test.go`
```go
func TestCancelOrder(t *testing.T) {
    ord := newTestOrder(t)
    ord.AddItem(order.NewOrderItemID("i1"), product.NewProductID("p1"), 1, mustMoney(500, "USD"))
    events, err := ord.Cancel()
    // assert: status changed, event emitted, errors on invalid transitions
}
```

### Application Tests

Test use cases with mocked repository ports to verify orchestration.

**File:** `internal/application/order/place_order_test.go`
```go
func TestPlaceOrderHandler(t *testing.T) {
    handler := NewPlaceOrderHandler(mockOrders, mockProducts, mockCustomers, mockEvents)
    ord, err := handler.Handle(ctx, req)
    // assert: order placed, correct total, event published
}
```

## Running

```bash
# Start server (SQLite database, seeded on first run)
go run ./cmd/server/

# Server on :8080 with 4 products and 1 customer seeded
curl :8080/products

# Create an order
curl -X POST :8080/orders -d '{
  "customer_id": "<customer-id-from-seed>",
  "items": [{"product_id": "<product-id>", "quantity": 1}],
  "shipping_address": {
    "line1": "123 Main St", "city": "Portland",
    "state": "OR", "zip_code": "97201", "country": "US"
  }
}'

# Cancel or ship
curl -X POST :8080/orders/<order-id>/cancel
curl -X POST :8080/orders/<order-id>/ship

# Create more products/customers
curl -X POST :8080/products -d '{"name":"Desk","price":49999,"stock":5}'
curl -X POST :8080/customers -d '{"first_name":"Bob","last_name":"Smith","email":"bob@example.com"}'

# Run tests
go test ./...
```

## Design Decisions

| Decision | Why |
|----------|-----|
| Go over TypeScript/Rust | Familiar to most engineers, explicit error handling, clear package boundaries |
| SQLite over in-memory | Shows real repository pattern with transactions; trivial to swap |
| UUID generation in application layer | Domain remains dependency-free; ID generation is an infrastructure concern |
| Events returned from domain methods | Cleaner than injecting an event bus into entities; makes domain testable without mocking events |
| Separate packages per aggregate | Clear boundaries; prevents cyclic dependencies when well-designed |
| `Restore*` constructor pattern | Repositories need to reconstitute entities from DB without re-validating timestamps or re-running creation logic |

## When to Use DDD (and When Not To)

**Use DDD when:** Complex business rules, long-lived systems, multiple entry points (API + CLI + events), team of 5+ developers.

**Skip DDD for:** Simple CRUD, prototypes, internal tools, solo projects. Start simple and evolve complexity only when needed.
