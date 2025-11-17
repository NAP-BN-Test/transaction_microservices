#!/bin/bash

echo "Starting services locally..."

# Start PostgreSQL (requires Homebrew installation)
echo "Starting PostgreSQL..."
brew services start postgresql@15

# Start Redis (requires Homebrew installation)  
echo "Starting Redis..."
brew services start redis

# Wait for services to start
sleep 3

# Create databases
echo "Creating databases..."
createdb orders 2>/dev/null || true
createdb sales 2>/dev/null || true

# Run init script
echo "Initializing databases..."
psql -d orders -f init.sql

echo "Services started. You can now run:"
echo "cd order-service && go run ."
echo "cd sales-service && npm start"