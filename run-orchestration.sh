#!/bin/bash

echo "=== Starting Orchestration Demo ==="

# Function to check if service is running
check_service() {
    local url=$1
    local name=$2
    echo "Checking $name..."
    if curl -s $url > /dev/null; then
        echo "✓ $name is running"
        return 0
    else
        echo "✗ $name is not running"
        return 1
    fi
}

# Check if all services are running
echo "1. Checking services status..."
check_service "http://localhost:8080/products" "Order Service"
ORDER_STATUS=$?

check_service "http://localhost:3000/vouchers" "Sales Service"
SALES_STATUS=$?

check_service "http://localhost:8081/orchestrate/health" "Orchestrator Service"
ORCHESTRATOR_STATUS=$?

if [ $ORDER_STATUS -ne 0 ] || [ $SALES_STATUS -ne 0 ]; then
    echo ""
    echo "Please start the required services first:"
    echo "Terminal 1: cd order-service && go run ."
    echo "Terminal 2: cd sales-service && npm start"
    echo "Terminal 3: cd orchestrator-service && go run ."
    exit 1
fi

echo ""
echo "2. All services are running! Starting orchestration tests..."

# Make the test script executable
chmod +x test-orchestration.sh

# Run the tests
./test-orchestration.sh

echo ""
echo "3. Running Node.js test script..."
if command -v node &> /dev/null; then
    node orchestration-demo.js
else
    echo "Node.js not found. Skipping Node.js tests."
fi

echo ""
echo "=== Orchestration Demo Completed ==="