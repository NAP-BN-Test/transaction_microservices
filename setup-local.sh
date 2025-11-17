#!/bin/bash

echo "Setting up local development environment..."

# Install dependencies if not present
if ! command -v brew &> /dev/null; then
    echo "Homebrew not found. Please install: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    exit 1
fi

# Install PostgreSQL and Redis
echo "Installing PostgreSQL and Redis..."
brew install postgresql@15 redis

# Start services
echo "Starting services..."
brew services start postgresql@15
brew services start redis

# Wait for PostgreSQL to start
sleep 3

# Create databases
echo "Creating databases..."
createdb orders 2>/dev/null || echo "Database 'orders' already exists"
createdb sales 2>/dev/null || echo "Database 'sales' already exists"

# Initialize databases
echo "Initializing databases..."
psql -d postgres -f init.sql

echo "Setup complete! Now run:"
echo "Terminal 1: cd order-service && go run ."
echo "Terminal 2: cd sales-service && npm install && npm start"