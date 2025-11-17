# Transaction Microservices Example

Dự án demo về quản lý transaction giữa 2 microservices sử dụng Saga pattern.

## Kiến trúc

### Golang Order Service (Port 8080)
- Quản lý sản phẩm và tồn kho
- Tạo và quản lý đơn hàng
- Xử lý Saga events

### Node.js Sales Service (Port 3000)
- Quản lý voucher giảm giá
- Xử lý giao dịch bán hàng
- Tính toán giá cuối cùng

### Saga Pattern
- Sử dụng Redis pub/sub để giao tiếp giữa services
- Đảm bảo tính nhất quán dữ liệu
- Rollback tự động khi có lỗi

## Chạy hệ thống

```bash
# Khởi động tất cả services
docker-compose up --build

# Hoặc chạy từng service riêng lẻ
cd order-service && go run main.go
cd sales-service && npm start
```

## API Endpoints

### Order Service (http://localhost:8080)
- `GET /products` - Lấy danh sách sản phẩm
- `POST /orders` - Tạo đơn hàng mới
- `GET /orders/:id` - Lấy thông tin đơn hàng
- `PUT /orders/:id/status` - Cập nhật trạng thái đơn hàng

### Sales Service (http://localhost:3000)
- `GET /vouchers` - Lấy danh sách voucher
- `POST /vouchers` - Tạo voucher mới
- `POST /sales/process` - Xử lý giao dịch bán hàng
- `GET /sales/:orderId` - Lấy thông tin giao dịch

## Test Flow

1. Lấy danh sách sản phẩm
2. Tạo đơn hàng
3. Hệ thống tự động tạo sales transaction
4. Kiểm tra kết quả

## Database Schema

### Orders DB
- products: sản phẩm và tồn kho
- orders: đơn hàng
- order_items: chi tiết đơn hàng

### Sales DB
- vouchers: mã giảm giá
- sales_transactions: giao dịch bán hàng# transaction_microservices
