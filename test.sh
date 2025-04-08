#!/bin/bash

# Test script for Real-Time Price Aggregator API

# Base URL
BASE_URL="http://localhost:8080"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}PASS: $2${NC}"
    else
        echo -e "${RED}FAIL: $2${NC}"
        exit 1
    fi
}

# Test 1: Health check
echo "Testing health endpoint..."
http_code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/health")
[ "$http_code" -eq 200 ]
print_result $? "Health check should return 200"

# Test 2: POST refresh asset1
echo "Testing POST /refresh/asset1..."
response=$(curl -s -X POST "${BASE_URL}/refresh/asset1")
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/refresh/asset1")
echo "$response" | grep -q '"message":"Price for asset1 refreshed"'
print_result $? "POST /refresh/asset1 should return success message"
[ "$http_code" -eq 200 ]
print_result $? "POST /refresh/asset1 should return status 200"

# Test 3: GET asset1 (should succeed, from cache)
echo "Testing GET /prices/asset1 (from cache)..."
response=$(curl -s "${BASE_URL}/prices/asset1")
http_code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/prices/asset1")
echo "$response" | grep -q '"asset":"asset1"'
print_result $? "GET /prices/asset1 should return price data"
[ "$http_code" -eq 200 ]
print_result $? "GET /prices/asset1 should return status 200"

# Test 4: GET asset10001 (not in CSV)
echo "Testing GET /prices/asset10001 (not in CSV)..."
response=$(curl -s "${BASE_URL}/prices/asset10001")
http_code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/prices/asset10001")
echo "$response" | grep -q '"msg":"Asset not found"'
print_result $? "GET /prices/asset10001 should return 404 (Asset not found)"
[ "$http_code" -eq 404 ]
print_result $? "GET /prices/asset10001 should return status 404"

# Test 5: POST refresh asset10001 (not in CSV)
echo "Testing POST /refresh/asset10001 (not in CSV)..."
response=$(curl -s -X POST "${BASE_URL}/refresh/asset10001")
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/refresh/asset10001")
echo "$response" | grep -q '"msg":"Asset not found"'
print_result $? "POST /refresh/asset10001 should return 404 (Asset not found)"
[ "$http_code" -eq 404 ]
print_result $? "POST /refresh/asset10001 should return status 404"

# Test 6: Check DynamoDB
echo "Checking DynamoDB for asset1..."
result=$(aws dynamodb scan \
  --table-name prices \
  --region us-west-2 \
  --query "Items[?asset.S=='asset1']" \
  --output json)

count=$(echo "$result" | jq 'length')
if [ "$count" -gt 0 ]; then
  print_result 0 "DynamoDB contains asset1 record"
else
  print_result 1 "DynamoDB should contain asset1 record"
fi


echo -e "${GREEN}All tests passed!${NC}"