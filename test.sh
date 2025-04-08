#!/bin/bash

# test.sh: Script to test the Real-Time Price Aggregator system

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Function to print success message
success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

# Function to print failure message and exit
failure() {
    echo -e "${RED}[FAILURE] $1${NC}"
    exit 1
}

# Test 1: Check if Redis is running
echo "Testing Redis connectivity..."
if redis-cli -h localhost -p 6379 ping | grep -q "PONG"; then
    success "Redis is running and responding"
else
    failure "Redis is not running or not responding"
fi

# Test 2: Check if Localstack (DynamoDB) is running
echo "Testing Localstack (DynamoDB) connectivity..."
if aws --endpoint-url=http://localhost:4566 dynamodb list-tables | grep -q "prices"; then
    success "Localstack is running and DynamoDB table 'prices' exists"
else
    failure "Localstack is not running or DynamoDB table 'prices' is not created"
fi

# Test 3: Test the mock exchanges
echo "Testing mock exchanges..."
for port in 8081 8082 8083; do
    response=$(curl -s http://localhost:$port/mock/ticker/BTCUSDT)
    if echo "$response" | grep -q '"symbol":"BTCUSDT"'; then
        success "Exchange on port $port is running and responding"
    else
        failure "Exchange on port $port is not running or not responding"
    fi
done

# Test 4: Test the main API (GET /prices/{asset})
echo "Testing GET /prices/BTCUSDT..."
response=$(curl -s http://localhost:8080/prices/BTCUSDT)
if echo "$response" | grep -q '"asset":"BTCUSDT"'; then
    success "GET /prices/BTCUSDT returned a valid response"
else
    failure "GET /prices/BTCUSDT failed"
fi

# Test 5: Test the main API (POST /refresh/{asset})
echo "Testing POST /refresh/BTCUSDT..."
response=$(curl -s -X POST http://localhost:8080/refresh/BTCUSDT)
if echo "$response" | grep -q '"message":"Price for BTCUSDT refreshed"'; then
    success "POST /refresh/BTCUSDT returned a valid response"
else
    failure "POST /refresh/BTCUSDT failed"
fi

# Test 6: Check DynamoDB for stored data
echo "Checking DynamoDB for stored data..."
response=$(aws --endpoint-url=http://localhost:4566 dynamodb scan --table-name prices)
if echo "$response" | grep -q '"asset": {"S": "BTCUSDT"}'; then
    success "DynamoDB contains price data for BTCUSDT"
else
    failure "DynamoDB does not contain expected price data"
fi

echo -e "${GREEN}All tests passed successfully!${NC}"