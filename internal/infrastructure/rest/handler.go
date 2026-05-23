package rest

import (
	"errors"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apporder "practice-ddd/internal/application/order"
	"practice-ddd/internal/domain/customer"
	"practice-ddd/internal/domain/order"
	"practice-ddd/internal/domain/product"
	"practice-ddd/internal/domain/shared"
)

func writeError(c *gin.Context, status int, err error) {
	var de *shared.DomainError
	if errors.As(err, &de) {
		c.JSON(status, gin.H{"code": de.Code, "message": de.Message})
		return
	}
	c.JSON(status, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
}

func writeRequestError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
}

type Server struct {
	placeOrder  *apporder.PlaceOrderHandler
	cancelOrder *apporder.CancelOrderHandler
	shipOrder   *apporder.ShipOrderHandler
	products    product.Repository
	customers   customer.Repository
}

func NewServer(
	placeOrder *apporder.PlaceOrderHandler,
	cancelOrder *apporder.CancelOrderHandler,
	shipOrder *apporder.ShipOrderHandler,
	products product.Repository,
	customers customer.Repository,
) *Server {
	return &Server{
		placeOrder: placeOrder, cancelOrder: cancelOrder, shipOrder: shipOrder,
		products: products, customers: customers,
	}
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()
	r.POST("/orders", s.handlePlaceOrder)
	r.POST("/orders/:id/cancel", s.handleCancelOrder)
	r.POST("/orders/:id/ship", s.handleShipOrder)
	r.GET("/products", s.handleListProducts)
	r.POST("/products", s.handleCreateProduct)
	r.POST("/customers", s.handleCreateCustomer)
	return r
}

func (s *Server) handlePlaceOrder(c *gin.Context) {
	var req struct {
		CustomerID string `json:"customer_id" binding:"required"`
		Items      []struct {
			ProductID string `json:"product_id" binding:"required"`
			Quantity  int    `json:"quantity" binding:"required,min=1"`
		} `json:"items" binding:"required,min=1"`
		ShippingAddress struct {
			Line1   string `json:"line1" binding:"required"`
			City    string `json:"city" binding:"required"`
			State   string `json:"state"`
			ZipCode string `json:"zip_code"`
			Country string `json:"country" binding:"required"`
		} `json:"shipping_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeRequestError(c, err)
		return
	}
	appReq := apporder.PlaceOrderRequest{
		CustomerID: req.CustomerID,
		ShippingAddress: apporder.AddressInput{
			Line1: req.ShippingAddress.Line1, City: req.ShippingAddress.City,
			State: req.ShippingAddress.State, ZipCode: req.ShippingAddress.ZipCode,
			Country: req.ShippingAddress.Country,
		},
	}
	for _, item := range req.Items {
		appReq.Items = append(appReq.Items, apporder.PlaceOrderItemInput{
			ProductID: item.ProductID, Quantity: item.Quantity,
		})
	}

	ord, err := s.placeOrder.Handle(c.Request.Context(), appReq)
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, err)
		return
	}

	c.JSON(http.StatusCreated, orderToJSON(ord))
}

func (s *Server) handleCancelOrder(c *gin.Context) {
	id := c.Param("id")
	if err := s.cancelOrder.Handle(c.Request.Context(), apporder.CancelOrderRequest{OrderID: id}); err != nil {
		writeError(c, http.StatusUnprocessableEntity, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (s *Server) handleShipOrder(c *gin.Context) {
	id := c.Param("id")
	if err := s.shipOrder.Handle(c.Request.Context(), apporder.ShipOrderRequest{OrderID: id}); err != nil {
		writeError(c, http.StatusUnprocessableEntity, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "shipped"})
}

func orderToJSON(o *order.Order) gin.H {
	items := make([]gin.H, 0, len(o.Items()))
	for _, item := range o.Items() {
		items = append(items, gin.H{
			"product_id": item.ProductID().String(),
			"quantity":   item.Quantity(),
			"unit_price": item.UnitPrice().Amount(),
			"currency":   item.UnitPrice().Currency(),
		})
	}
	return gin.H{
		"id":          o.ID().String(),
		"customer_id": o.CustomerID().String(),
		"status":      string(o.Status()),
		"items":       items,
		"total":       o.Total().Amount(),
		"currency":    o.Total().Currency(),
	}
}

func (s *Server) handleListProducts(c *gin.Context) {
	products, err := s.products.FindAll(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(products))
	for _, p := range products {
		out = append(out, gin.H{
			"id":       p.ID().String(),
			"name":     p.Name(),
			"price":    p.Price().Amount(),
			"currency": p.Price().Currency(),
			"stock":    p.Stock(),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) handleCreateProduct(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Price int64  `json:"price" binding:"required"`
		Stock int    `json:"stock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeRequestError(c, err)
		return
	}
	price, err := shared.NewMoney(req.Price, "USD")
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	p, err := product.NewProduct(product.NewProductID(uuid.New().String()), req.Name, price, req.Stock)
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, err)
		return
	}
	if err := s.products.Save(c.Request.Context(), p); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": p.ID().String(), "name": p.Name(),
		"price": p.Price().Amount(), "stock": p.Stock(),
	})
}

func (s *Server) handleCreateCustomer(c *gin.Context) {
	var req struct {
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeRequestError(c, err)
		return
	}
	cust, err := customer.NewCustomer(
		customer.NewCustomerID(uuid.New().String()),
		customer.NewCustomerName(req.FirstName, req.LastName),
		customer.NewEmail(req.Email),
	)
	if err != nil {
		writeError(c, http.StatusUnprocessableEntity, err)
		return
	}
	if err := s.customers.Save(c.Request.Context(), cust); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": cust.ID().String(),
		"name": cust.Name().FullName(),
		"email": cust.Email().String(),
	})
}
