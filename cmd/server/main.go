package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"github.com/google/uuid"
	apporder "practice-ddd/internal/application/order"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
	"practice-ddd/internal/infrastructure/persistence/sqlite"
	"practice-ddd/internal/infrastructure/rest"
)

type consoleEventBus struct{}

func (b *consoleEventBus) Publish(_ context.Context, event order.DomainEvent) error {
	log.Printf("[EVENT] %s: %+v", event.EventName(), event)
	return nil
}

func seedData(ctx context.Context, customers customer.Repository, products product.Repository) {
	existing, _ := products.FindAll(ctx)
	if len(existing) > 0 {
		return
	}

	cust, _ := customer.NewCustomer(
		customer.NewCustomerID(uuid.New().String()),
		customer.NewCustomerName("Alice", "Wonderland"),
		customer.NewEmail("alice@example.com"),
	)
	customers.Save(ctx, cust)

	products.Save(ctx, mustProduct("Laptop", 99999, 10))
	products.Save(ctx, mustProduct("Mouse", 2499, 50))
	products.Save(ctx, mustProduct("Keyboard", 7499, 30))
	products.Save(ctx, mustProduct("Monitor", 29999, 15))

	fmt.Printf("Seeded customer: %s (Alice Wonderland)\n", cust.ID().String())
}

func mustProduct(name string, priceCents int64, stock int) *product.Product {
	p, err := product.NewProduct(
		product.NewProductID(uuid.New().String()),
		name,
		mustMoney(priceCents, "USD"),
		stock,
	)
	if err != nil {
		panic(err)
	}
	return p
}

func mustMoney(amount int64, currency string) shared.Money {
	m, err := shared.NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func main() {
	dbPath := "orders.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sqlite.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	customerRepo := sqlite.NewCustomerRepository(db)
	productRepo := sqlite.NewProductRepository(db)
	orderRepo := sqlite.NewOrderRepository(db)
	eventBus := &consoleEventBus{}

	ctx := context.Background()
	seedData(ctx, customerRepo, productRepo)

	placeOrder := apporder.NewPlaceOrderHandler(orderRepo, productRepo, customerRepo, eventBus)
	cancelOrder := apporder.NewCancelOrderHandler(orderRepo, eventBus)
	shipOrder := apporder.NewShipOrderHandler(orderRepo, eventBus)

	srv := rest.NewServer(placeOrder, cancelOrder, shipOrder, productRepo, customerRepo)

	addr := ":8080"
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
