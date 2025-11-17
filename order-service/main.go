package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Order struct {
	ID          int         `json:"id"`
	CustomerID  int         `json:"customer_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
}

type OrderItem struct {
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Product struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	StockQuantity int     `json:"stock_quantity"`
}

type SagaEvent struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	OrderID   int         `json:"order_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

var (
	db            *sql.DB
	rdb           *redis.Client
	orderService  OrderService
	sagaService   SagaService
	outboxService OutboxService
)

func main() {
	// Load .env file
	godotenv.Load()
	
	initDB()
	initRedis()
	initServices()

	r := gin.Default()

	r.GET("/products", getProducts)
	r.POST("/orders", createOrder)
	r.GET("/orders/:id", getOrder)
	r.PUT("/orders/:id/status", updateOrderStatus)

	// Background services
	go sagaService.ProcessEvents()
	go outboxService.ProcessEvents()

	r.Run(":8080")
}

func initServices() {
	orderRepo := NewOrderRepository(db)
	productRepo := NewProductRepository(db)
	outboxRepo := NewOutboxRepository(db)

	orderService = NewOrderService(orderRepo, productRepo, outboxRepo, db)
	sagaService = NewSagaService(rdb, orderRepo, productRepo)
	outboxService = NewOutboxService(outboxRepo, sagaService)
}

func initDB() {
	var err error
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
}

func getProducts(c *gin.Context) {
	productRepo := NewProductRepository(db)
	products, err := productRepo.GetAll()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, products)
}

func createOrder(c *gin.Context) {
	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	createdOrder, err := orderService.CreateOrder(&order)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, createdOrder)
}

func getOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	order, err := orderService.GetOrder(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(200, order)
}

func updateOrderStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	err := orderService.UpdateOrderStatus(id, req.Status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Order status updated"})
}
