package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// OrderServiceImpl implements OrderService
type OrderServiceImpl struct {
	orderRepo   OrderRepository
	productRepo ProductRepository
	outboxRepo  OutboxRepository
	db          *sql.DB
}

func NewOrderService(orderRepo OrderRepository, productRepo ProductRepository, outboxRepo OutboxRepository, db *sql.DB) OrderService {
	return &OrderServiceImpl{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		outboxRepo:  outboxRepo,
		db:          db,
	}
}

func (s *OrderServiceImpl) CreateOrder(order *Order) (*Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Calculate total and check stock
	var totalAmount float64
	for _, item := range order.Items {
		product, err := s.productRepo.GetByID(tx, item.ProductID)
		if err != nil {
			return nil, err
		}

		if product.StockQuantity < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}

		totalAmount += product.Price * float64(item.Quantity)
		
		err = s.productRepo.UpdateStock(tx, item.ProductID, item.Quantity)
		if err != nil {
			return nil, err
		}
	}

	order.TotalAmount = totalAmount
	orderID, err := s.orderRepo.Create(tx, order)
	if err != nil {
		return nil, err
	}

	// Create order items
	for _, item := range order.Items {
		product, _ := s.productRepo.GetByID(tx, item.ProductID)
		_, err = tx.Exec("INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, $3, $4)",
			orderID, item.ProductID, item.Quantity, product.Price)
		if err != nil {
			return nil, err
		}
	}

	// Store saga event in outbox
	sagaEvent := &SagaEvent{
		ID:        uuid.New().String(),
		Type:      "ORDER_CREATED",
		OrderID:   orderID,
		Data:      map[string]interface{}{"customer_id": order.CustomerID, "total_amount": totalAmount},
		Timestamp: time.Now(),
	}

	err = s.outboxRepo.Store(tx, sagaEvent)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	order.ID = orderID
	order.Status = "pending"
	return order, nil
}

func (s *OrderServiceImpl) GetOrder(id int) (*Order, error) {
	return s.orderRepo.GetByID(id)
}

func (s *OrderServiceImpl) UpdateOrderStatus(id int, status string) error {
	return s.orderRepo.UpdateStatus(id, status)
}

// SagaServiceImpl implements SagaService
type SagaServiceImpl struct {
	rdb         *redis.Client
	orderRepo   OrderRepository
	productRepo ProductRepository
}

func NewSagaService(rdb *redis.Client, orderRepo OrderRepository, productRepo ProductRepository) SagaService {
	return &SagaServiceImpl{
		rdb:         rdb,
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (s *SagaServiceImpl) PublishEvent(event SagaEvent) {
	eventJSON, _ := json.Marshal(event)
	s.rdb.Publish(context.Background(), "saga_events", eventJSON)
}

func (s *SagaServiceImpl) ProcessEvents() {
	pubsub := s.rdb.Subscribe(context.Background(), "saga_responses")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var event SagaEvent
		json.Unmarshal([]byte(msg.Payload), &event)

		switch event.Type {
		case "SALES_TRANSACTION_COMPLETED":
			s.orderRepo.UpdateStatus(event.OrderID, "completed")
		case "SALES_TRANSACTION_FAILED":
			s.rollbackOrder(event.OrderID)
		}
	}
}

func (s *SagaServiceImpl) rollbackOrder(orderID int) {
	// Implementation for rollback logic
	s.orderRepo.UpdateStatus(orderID, "cancelled")
}

// OutboxServiceImpl implements OutboxService
type OutboxServiceImpl struct {
	outboxRepo  OutboxRepository
	sagaService SagaService
}

func NewOutboxService(outboxRepo OutboxRepository, sagaService SagaService) OutboxService {
	return &OutboxServiceImpl{
		outboxRepo:  outboxRepo,
		sagaService: sagaService,
	}
}

func (s *OutboxServiceImpl) ProcessEvents() {
	for {
		events, err := s.outboxRepo.GetUnprocessed()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, event := range events {
			sagaEvent := SagaEvent{
				ID:        event.EventID,
				Type:      event.EventType,
				OrderID:   event.AggregateID,
				Timestamp: time.Now(),
			}

			json.Unmarshal(event.EventData, &sagaEvent.Data)
			s.sagaService.PublishEvent(sagaEvent)
			s.outboxRepo.MarkProcessed(event.ID)
		}

		time.Sleep(2 * time.Second)
	}
}