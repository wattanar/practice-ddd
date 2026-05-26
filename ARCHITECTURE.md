# DDD Architecture Diagram

## 1. Clean Architecture (Hexagonal) Layers

```mermaid
%%{init: {'theme': 'neutral', 'flowchart': {'curve': 'basis'}}}%%
flowchart LR
    subgraph outer["Infrastructure (Adapters)"]
        direction LR
        R["rest/\nHTTP Handlers\n(Gin)"]
        S["persistence/sqlite/\nSQLite Repos"]
        C["cmd/server/\nComposition Root"]
    end

    subgraph app["Application Layer"]
        A["application/order/\nUse Case Handlers\nPlaceOrderHandler\nCancelOrderHandler\nShipOrderHandler"]
        EB[("EventBus Port")]
    end

    subgraph domain["Domain Layer (Zero Dependencies Outward)"]
        direction TB
        O["domain/order/\nOrder (Aggregate Root)\nOrderItem (Entity)\nOrderStatus (Value Obj)\nAddress (Value Obj)\nOrderPlaced/Cancelled (Events)"]
        P["domain/product/\nProduct (Entity)\nProductID (Value Obj)"]
        C2["domain/customer/\nCustomer (Entity)\nEmail/CustomerName (Value Objs)"]
        S2["domain/shared/\nMoney (Value Obj)\nDomainError"]
        REPO["Repository Ports\n(order.Repository)\n(product.Repository)\n(customer.Repository)"]
    end

    outer -- implements ports --> app
    app -- orchestrates domain --> domain
    outer -- implements ports --> domain
```

## 2. Request Flow: Place Order

```mermaid
%%{init: {'theme': 'neutral', 'sequence': {'showSequenceNumbers': true}}}%%
sequenceDiagram
    participant Client as HTTP Client
    participant Handler as rest.Server<br/>(Infrastructure)
    participant UseCase as PlaceOrderHandler<br/>(Application)
    participant Domain as Order Aggregate<br/>(Domain)
    participant Repo as order.Repository<br/>(Infra: SQLite)
    participant EB as EventBus<br/>(Infra: Console)

    Client->>Handler: POST /orders
    Handler->>UseCase: Handle(ctx, PlaceOrderRequest)
    UseCase->>Domain: customerRepo.FindByID(customerID)
    UseCase->>Domain: order.NewOrder(id, customerID, address)
    UseCase->>Domain: productRepo.FindByID(productID)
    UseCase->>Domain: ord.AddItem(id, productID, qty, price)
    UseCase->>Domain: ord.Place()
    Domain-->>UseCase: [OrderPlaced] events
    UseCase->>Repo: orders.Save(ctx, order)
    UseCase->>EB: events.Publish(ctx, OrderPlaced)
    UseCase-->>Handler: order, nil
    Handler-->>Client: 201 Created
```

## 3. Aggregate: Order

```mermaid
%%{init: {'theme': 'neutral', 'class': {'hideEmptyMembersBox': true}}}%%
classDiagram
    class Order {
        -id OrderID
        -customerID CustomerID
        -items []*OrderItem
        -status OrderStatus
        -shippingAddr Address
        -total Money
        -createdAt time.Time
        -updatedAt time.Time
        +AddItem(id, productID, qty, price) error
        +RemoveItem(productID) error
        +Place() []DomainEvent
        +Cancel() []DomainEvent
        +Ship() []DomainEvent
        -recalculateTotal()
    }

    class OrderItem {
        -id OrderItemID
        -productID ProductID
        -quantity int
        -unitPrice Money
        +Total() Money
    }

    class OrderStatus {
        <<enumeration>>
        pending
        placed
        shipped
        delivered
        cancelled
        +CanTransitionTo(target) bool
    }

    class Address {
        -line1 string
        -city string
        -state string
        -zipCode string
        -country string
    }

    class DomainEvent {
        <<interface>>
        +EventName() string
        +OccurredAt() time.Time
    }

    class OrderPlaced {
        +OrderID OrderID
        +CustomerID CustomerID
        +Total Money
        +PlacedAt time.Time
    }

    class OrderCancelled {
        +OrderID OrderID
        +CustomerID CustomerID
        +CancelledAt time.Time
    }

    Order "1" --> "*" OrderItem : contains
    Order --> OrderStatus : has
    Order --> Address : has
    Order ..> DomainEvent : returns from Place/Cancel
    DomainEvent <|-- OrderPlaced
    DomainEvent <|-- OrderCancelled
```

## 4. Dependency Inversion (Ports & Adapters)

```mermaid
%%{init: {'theme': 'neutral'}}%%
flowchart TB
    subgraph domain["Domain Layer"]
        OR["order.Repository (interface)"]
        PR["product.Repository (interface)"]
        CR["customer.Repository (interface)"]
    end

    subgraph app["Application Layer"]
        EB["EventBus (interface)"]
        UC["Use Case Handlers"]
    end

    subgraph infra["Infrastructure Layer"]
        OR_SQL["sqlite.OrderRepository"]
        PR_SQL["sqlite.ProductRepository"]
        CR_SQL["sqlite.CustomerRepository"]
        CE["consoleEventBus"]
        HTTP["rest.Server"]
    end

    OR_SQL -. implements .-> OR
    PR_SQL -. implements .-> PR
    CR_SQL -. implements .-> CR
    CE -. implements .-> EB

    UC --> OR
    UC --> PR
    UC --> CR
    UC --> EB

    HTTP --> UC
    HTTP --> PR
    HTTP --> CR

    style domain fill:#e1f5e1,stroke:#2e7d32
    style app fill:#e3f2fd,stroke:#1565c0
    style infra fill:#fff3e0,stroke:#e65100
```

## 5. Domain Event Flow

```mermaid
%%{init: {'theme': 'neutral', 'flowchart': {'curve': 'step'}}}%%
flowchart LR
    A["Order.Place()"] --> B["Returns\n[OrderPlaced]"]
    B --> C["Application:\norders.Save()"]
    C --> D["Application:\nevents.Publish()"]
    D --> E["Console log\n(infrastructure)"]

    A2["Order.Cancel()"] --> B2["Returns\n[OrderCancelled]"]
    B2 --> C2["Application:\norders.Save()"]
    C2 --> D2["Application:\nevents.Publish()"]
    D2 --> E2["Console log\n(infrastructure)"]

    style A fill:#e1f5e1,stroke:#2e7d32
    style A2 fill:#e1f5e1,stroke:#2e7d32
    style D fill:#e3f2fd,stroke:#1565c0
    style D2 fill:#e3f2fd,stroke:#1565c0
```
