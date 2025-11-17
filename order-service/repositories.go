package main

import (
	"database/sql"
	"encoding/json"
)

// OrderRepositoryImpl implements OrderRepository
type OrderRepositoryImpl struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &OrderRepositoryImpl{db: db}
}

func (r *OrderRepositoryImpl) Create(tx *sql.Tx, order *Order) (int, error) {
	var orderID int
	err := tx.QueryRow("INSERT INTO orders (customer_id, total_amount, status) VALUES ($1, $2, 'pending') RETURNING id",
		order.CustomerID, order.TotalAmount).Scan(&orderID)
	return orderID, err
}

func (r *OrderRepositoryImpl) GetByID(id int) (*Order, error) {
	var order Order
	err := r.db.QueryRow("SELECT id, customer_id, total_amount, status, created_at FROM orders WHERE id = $1", id).
		Scan(&order.ID, &order.CustomerID, &order.TotalAmount, &order.Status, &order.CreatedAt)
	return &order, err
}

func (r *OrderRepositoryImpl) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec("UPDATE orders SET status = $1 WHERE id = $2", status, id)
	return err
}

// ProductRepositoryImpl implements ProductRepository
type ProductRepositoryImpl struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &ProductRepositoryImpl{db: db}
}

func (r *ProductRepositoryImpl) GetAll() ([]Product, error) {
	rows, err := r.db.Query("SELECT id, name, price, stock_quantity FROM products")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.StockQuantity)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepositoryImpl) GetByID(tx *sql.Tx, id int) (*Product, error) {
	var product Product
	err := tx.QueryRow("SELECT id, name, price, stock_quantity FROM products WHERE id = $1", id).
		Scan(&product.ID, &product.Name, &product.Price, &product.StockQuantity)
	return &product, err
}

func (r *ProductRepositoryImpl) UpdateStock(tx *sql.Tx, id, quantity int) error {
	_, err := tx.Exec("UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2", quantity, id)
	return err
}

// OutboxRepositoryImpl implements OutboxRepository
type OutboxRepositoryImpl struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) OutboxRepository {
	return &OutboxRepositoryImpl{db: db}
}

func (r *OutboxRepositoryImpl) Store(tx *sql.Tx, event *SagaEvent) error {
	eventData, _ := json.Marshal(event.Data)
	_, err := tx.Exec("INSERT INTO outbox_events (event_id, event_type, aggregate_id, event_data) VALUES ($1, $2, $3, $4)",
		event.ID, event.Type, event.OrderID, eventData)
	return err
}

func (r *OutboxRepositoryImpl) GetUnprocessed() ([]OutboxEvent, error) {
	rows, err := r.db.Query("SELECT id, event_id, event_type, aggregate_id, event_data FROM outbox_events WHERE processed = false ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventID, &event.EventType, &event.AggregateID, &event.EventData)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *OutboxRepositoryImpl) MarkProcessed(id int) error {
	_, err := r.db.Exec("UPDATE outbox_events SET processed = true WHERE id = $1", id)
	return err
}