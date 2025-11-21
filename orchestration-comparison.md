# So sánh Saga Pattern vs Orchestration Pattern

## Saga Pattern (Hiện tại)

### Đặc điểm:
- **Decentralized**: Mỗi service tự quản lý logic của mình
- **Event-driven**: Giao tiếp qua Redis pub/sub
- **Autonomous**: Services độc lập, không phụ thuộc lẫn nhau
- **Outbox Pattern**: Đảm bảo eventual consistency

### Luồng hoạt động:
1. Order Service tạo đơn hàng → publish `ORDER_CREATED` event
2. Sales Service nhận event → xử lý transaction → publish `SALES_COMPLETED`
3. Order Service nhận response → cập nhật trạng thái

### Ưu điểm:
- Loose coupling giữa các services
- Dễ mở rộng (thêm services mới)
- Fault tolerance tốt
- Phù hợp với microservices architecture

### Nhược điểm:
- Khó debug và monitor
- Eventual consistency
- Phức tạp trong việc handle compensation

## Orchestration Pattern (Mới thêm)

### Đặc điểm:
- **Centralized**: Orchestrator điều phối toàn bộ workflow
- **Synchronous**: Gọi API trực tiếp giữa services
- **Controlled**: Orchestrator quản lý toàn bộ transaction flow
- **Immediate consistency**: Biết ngay kết quả success/failure

### Luồng hoạt động:
1. Client gọi Orchestrator
2. Orchestrator → Order Service (tạo đơn hàng)
3. Orchestrator → Sales Service (xử lý transaction)
4. Orchestrator → Order Service (confirm đơn hàng)
5. Nếu có lỗi → Orchestrator thực hiện compensation

### Ưu điểm:
- Dễ debug và monitor workflow
- Immediate feedback
- Centralized error handling
- Dễ implement business logic phức tạp

### Nhược điểm:
- Single point of failure
- Tight coupling với Orchestrator
- Khó scale khi có nhiều workflows

## Khi nào sử dụng?

### Saga Pattern:
- Hệ thống lớn với nhiều microservices
- Cần high availability
- Business logic phức tạp, domain-driven
- Long-running transactions

### Orchestration Pattern:
- Workflows đơn giản, rõ ràng
- Cần control chặt chẽ transaction flow
- Team nhỏ, dễ maintain
- Short-running transactions

## Test Cases

### Saga Pattern Test:
```bash
# Test qua events
curl -X POST http://localhost:8080/orders -d '{"customer_id":1,"items":[...]}'
# Kiểm tra events trong Redis
# Kiểm tra kết quả trong DB
```

### Orchestration Pattern Test:
```bash
# Test qua orchestrator
curl -X POST http://localhost:8081/orchestrate/order -d '{"customer_id":1,"items":[...]}'
# Kiểm tra workflow status
curl -X GET http://localhost:8081/orchestrate/{workflow_id}
```

## Kết luận

Cả hai pattern đều có giá trị trong các tình huống khác nhau:

- **Saga**: Phù hợp cho hệ thống microservices mature, cần tính autonomous cao
- **Orchestration**: Phù hợp cho workflows đơn giản, cần control tập trung

Trong thực tế, có thể kết hợp cả hai:
- Orchestration cho user-facing workflows
- Saga cho internal business processes