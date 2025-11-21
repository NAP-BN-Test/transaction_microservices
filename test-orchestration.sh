#!/bin/bash

echo "=== Testing Orchestration Pattern ==="

BASE_URL="http://localhost:8081"

echo "1. Test Orchestration - Success Case"
WORKFLOW_RESPONSE=$(curl -s -X POST $BASE_URL/orchestrate/order \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {"product_id": 1, "quantity": 2},
      {"product_id": 2, "quantity": 1}
    ],
    "voucher_code": "SAVE10"
  }')

echo "Response: $WORKFLOW_RESPONSE"

WORKFLOW_ID=$(echo $WORKFLOW_RESPONSE | jq -r '.workflow_id')
echo "Workflow ID: $WORKFLOW_ID"

echo ""
echo "2. Check workflow status (wait 3 seconds)"
sleep 3

curl -s -X GET $BASE_URL/orchestrate/$WORKFLOW_ID | jq '.'

echo ""
echo "3. Test Orchestration - Failure Case (invalid product)"
FAIL_RESPONSE=$(curl -s -X POST $BASE_URL/orchestrate/order \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {"product_id": 999, "quantity": 1}
    ]
  }')

echo "Response: $FAIL_RESPONSE"

FAIL_WORKFLOW_ID=$(echo $FAIL_RESPONSE | jq -r '.workflow_id')
echo "Failed Workflow ID: $FAIL_WORKFLOW_ID"

echo ""
echo "4. Check failed workflow status (wait 3 seconds)"
sleep 3

curl -s -X GET $BASE_URL/orchestrate/$FAIL_WORKFLOW_ID | jq '.'

echo ""
echo "5. Test manual compensation"
curl -s -X POST $BASE_URL/orchestrate/$FAIL_WORKFLOW_ID/compensate | jq '.'

echo ""
echo "6. Check compensated workflow status"
curl -s -X GET $BASE_URL/orchestrate/$FAIL_WORKFLOW_ID | jq '.'

echo ""
echo "=== Orchestration Tests Completed ==="