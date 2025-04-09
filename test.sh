#!/bin/bash

# Health check for the main service
curl http://localhost:8080/health

# Get mock ticker data from three exchanges for asset1
curl -s http://localhost:8081/mock/ticker/asset1
curl -s http://localhost:8082/mock/ticker/asset1
curl -s http://localhost:8083/mock/ticker/asset1

# Refresh prices for asset1 and asset10001
curl -X POST http://localhost:8080/refresh/asset1
curl -X POST http://localhost:8080/refresh/asset10001

# Get prices for asset1 and asset10001
curl -s http://localhost:8080/prices/asset1
curl -s http://localhost:8080/prices/asset10001

# Scan DynamoDB for asset1
# aws dynamodb scan --table-name prices --region us-west-2 --query "Items[?asset.S=='asset1']" --output json

# Scan all items in DynamoDB
# aws dynamodb scan --table-name prices --region us-west-2