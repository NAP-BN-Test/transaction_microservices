#!/bin/bash

echo "=== DEMO TRANSACTION FLOW ==="

echo "1. Lấy danh sách sản phẩm:"
curl -s http://localhost:8080/products | jq

echo -e "\n2. Tạo đơn hàng:"
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {
        "product_id": 1,
        "quantity": 1
      },
      {
        "product_id": 2,
        "quantity": 2
      }
    ]
  }')

echo $ORDER_RESPONSE | jq
ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.id')

echo -e "\n3. Chờ Saga processing..."
sleep 3

echo -e "\n4. Kiểm tra order status:"
curl -s http://localhost:8080/orders/$ORDER_ID | jq

echo -e "\n5. Kiểm tra sales transaction:"
curl -s http://localhost:3000/sales/$ORDER_ID | jq

echo -e "\n6. Kiểm tra stock sau khi tạo order:"
curl -s http://localhost:8080/products | jq

echo -e "\n=== DEMO HOÀN THÀNH ==="