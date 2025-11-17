package main

import "database/sql"

// Repository interfaces
type OrderRepository interface {
	Create(tx *sql.Tx, order *Order) (int, error)
	GetByID(id int) (*Order, error)
	UpdateStatus(id int, status string) error
}

type ProductRepository interface {
	GetAll() ([]Product, error)
	GetByID(tx *sql.Tx, id int) (*Product, error)
	UpdateStock(tx *sql.Tx, id, quantity int) error
}

type OutboxRepository interface {
	Store(tx *sql.Tx, event *SagaEvent) error
	GetUnprocessed() ([]OutboxEvent, error)
	MarkProcessed(id int) error
}

// Service interfaces
type OrderService interface {
	CreateOrder(order *Order) (*Order, error)
	GetOrder(id int) (*Order, error)
	UpdateOrderStatus(id int, status string) error
}

type SagaService interface {
	PublishEvent(event SagaEvent)
	ProcessEvents()
}

type OutboxService interface {
	ProcessEvents()
}

// Event types
type OutboxEvent struct {
	ID          int    `json:"id"`
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	AggregateID int    `json:"aggregate_id"`
	EventData   []byte `json:"event_data"`
}
