#!/bin/bash

echo "=== Testing Transaction Microservices ==="

# Wait for services to start
sleep 5

echo "1. Getting products..."
curl -X GET http://localhost:8080/products | jq

echo -e "\n2. Getting vouchers..."
curl -X GET http://localhost:3000/vouchers | jq

echo -e "\n3. Creating order..."
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {
        "product_id": 1,
        "quantity": 2
      },
      {
        "product_id": 2,
        "quantity": 1
      }
    ]
  }')

echo $ORDER_RESPONSE | jq

ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.id')

echo -e "\n4. Checking order status..."
sleep 2
curl -X GET http://localhost:8080/orders/$ORDER_ID | jq

echo -e "\n5. Checking sales transaction..."
curl -X GET http://localhost:3000/sales/$ORDER_ID | jq

echo -e "\n6. Processing sales with voucher..."
curl -X POST http://localhost:3000/sales/process \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": $ORDER_ID,
    \"customer_id\": 1,
    \"original_amount\": 2025.00,
    \"voucher_code\": \"SAVE10\"
  }" | jq

echo -e "\n=== Test completed ==="