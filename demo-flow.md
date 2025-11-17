# Demo Transaction Flow Between 2 Services

## Luồng hoạt động:

### 1. Tạo đơn hàng (Order Service)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {
        "product_id": 1,
        "quantity": 2
      }
    ]
  }'
```

### 2. Saga Pattern Flow:
1. **Order Service**: Tạo order → Lưu event vào outbox table
2. **Outbox Processor**: Đọc event từ outbox → Publish lên Redis
3. **Sales Service**: Nhận event → Tạo sales transaction
4. **Sales Service**: Lưu response event vào outbox → Publish success/failure
5. **Order Service**: Nhận response → Cập nhật order status

### 3. Kiểm tra kết quả:
```bash
# Xem order status
curl http://localhost:8080/orders/1

# Xem sales transaction
curl http://localhost:3000/sales/1
```

## Database Changes:

### Orders DB:
- `orders` table: Tạo record mới
- `outbox_events` table: Lưu ORDER_CREATED event
- `products` table: Giảm stock quantity

### Sales DB:
- `sales_transactions` table: Tạo transaction record
- `outbox_events` table: Lưu SALES_TRANSACTION_COMPLETED event

## Rollback Scenario:
Nếu Sales Service fail → Order Service sẽ:
- Rollback stock quantity
- Set order status = 'cancelled'