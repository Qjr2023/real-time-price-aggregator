#!/bin/bash

# debug-dynamodb.sh: Script to debug DynamoDB functionality

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== DynamoDB Debugging Tool ===${NC}"

# Check service logs
echo -e "\n${CYAN}Checking service logs...${NC}"
echo "Last 20 lines of server logs:"
docker logs real-time-price-aggregator-server-1 --tail 20

# Check if LocalStack is running
echo -e "\n${CYAN}Checking LocalStack status...${NC}"
health_response=$(curl -s http://localhost:4566/_localstack/health)
echo "LocalStack health response:"
echo "$health_response"

# Fix the pattern to properly detect if DynamoDB is running
if echo "$health_response" | grep -q "\"dynamodb\": \"running\""; then
    echo -e "${GREEN}LocalStack DynamoDB is running${NC}"
else
    echo -e "${RED}LocalStack DynamoDB is not running correctly${NC}"
    
    # Check LocalStack container logs
    echo "Last 20 lines of LocalStack logs:"
    docker logs real-time-price-aggregator-localstack-1 --tail 20
    
    exit 1
fi

# List all tables
echo -e "\n${CYAN}Listing DynamoDB tables...${NC}"
tables=$(aws --endpoint-url=http://localhost:4566 dynamodb list-tables)
echo "$tables"

# Check if prices table exists
if echo "$tables" | grep -q "prices"; then
    echo -e "${GREEN}Table 'prices' exists${NC}"
else
    echo -e "${RED}Table 'prices' does not exist${NC}"
    echo "Creating table 'prices'..."
    
    aws --endpoint-url=http://localhost:4566 dynamodb create-table \
        --table-name prices \
        --attribute-definitions \
            AttributeName=asset,AttributeType=S \
            AttributeName=timestamp,AttributeType=N \
        --key-schema \
            AttributeName=asset,KeyType=HASH \
            AttributeName=timestamp,KeyType=RANGE \
        --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
        
    echo "Table creation result: $?"
    
    # Wait for table to be created
    echo "Waiting for table to be created..."
    aws --endpoint-url=http://localhost:4566 dynamodb wait table-exists --table-name prices
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Table 'prices' created successfully${NC}"
    else
        echo -e "${RED}Failed to create table 'prices'${NC}"
        exit 1
    fi
fi

# Insert a test record
echo -e "\n${CYAN}Inserting a test record into 'prices' table...${NC}"
current_time=$(date +%s)
test_price=$(echo "scale=2; $RANDOM / 100" | bc)

insert_result=$(aws --endpoint-url=http://localhost:4566 dynamodb put-item \
    --table-name prices \
    --item '{
        "asset": {"S": "TEST_DEBUG"},
        "timestamp": {"N": "'$current_time'"},
        "price": {"N": "'$test_price'"},
        "updated_at": {"N": "'$current_time'"}
    }')
    
echo "Insert result: $?"

# Scan the table
echo -e "\n${CYAN}Scanning 'prices' table for all records...${NC}"
scan_result=$(aws --endpoint-url=http://localhost:4566 dynamodb scan --table-name prices)
echo "$scan_result"

# Count items
item_count=$(echo "$scan_result" | grep -o '"Count": [0-9]*' | cut -d' ' -f2)
echo -e "\n${CYAN}Found $item_count items in the table${NC}"

# Try a query for BTCUSDT
echo -e "\n${CYAN}Querying for BTCUSDT records...${NC}"
query_result=$(aws --endpoint-url=http://localhost:4566 dynamodb query \
    --table-name prices \
    --key-condition-expression "asset = :asset" \
    --expression-attribute-values '{":asset":{"S":"BTCUSDT"}}')
    
echo "$query_result"

# Try our test record
echo -e "\n${CYAN}Querying for our test record (TEST_DEBUG)...${NC}"
test_query_result=$(aws --endpoint-url=http://localhost:4566 dynamodb query \
    --table-name prices \
    --key-condition-expression "asset = :asset" \
    --expression-attribute-values '{":asset":{"S":"TEST_DEBUG"}}')
    
echo "$test_query_result"

# Manual test of the API to force a record creation
echo -e "\n${CYAN}Testing API to create a BTCUSDT record...${NC}"
refresh_result=$(curl -s -X POST http://localhost:8080/refresh/BTCUSDT)
echo "API response: $refresh_result"

# Wait a bit for potential background processing
echo "Waiting 5 seconds for record to be saved..."
sleep 5

# Check again for BTCUSDT records
echo -e "\n${CYAN}Checking again for BTCUSDT records...${NC}"
final_query_result=$(aws --endpoint-url=http://localhost:4566 dynamodb query \
    --table-name prices \
    --key-condition-expression "asset = :asset" \
    --expression-attribute-values '{":asset":{"S":"BTCUSDT"}}')
    
echo "$final_query_result"

# Check network connectivity
echo -e "\n${CYAN}Checking network connectivity from application container to LocalStack...${NC}"
docker exec real-time-price-aggregator-server-1 ping -c 2 localstack

# Check DynamoDB endpoint reachability
echo -e "\n${CYAN}Checking DynamoDB endpoint reachability from application container...${NC}"
docker exec real-time-price-aggregator-server-1 curl -v http://localstack:4566

echo -e "\n${CYAN}Debug complete.${NC}"