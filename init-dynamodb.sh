#!/bin/bash
set -e

echo "Initializing DynamoDB table..."

# Create the prices table
echo "Creating 'prices' table in DynamoDB..."
awslocal dynamodb create-table \
    --table-name prices \
    --attribute-definitions \
        AttributeName=asset,AttributeType=S \
        AttributeName=timestamp,AttributeType=N \
    --key-schema \
        AttributeName=asset,KeyType=HASH \
        AttributeName=timestamp,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

# Insert a test record
echo "Adding test record to 'prices' table..."
TIMESTAMP=$(date +%s)
awslocal dynamodb put-item \
    --table-name prices \
    --item '{
        "asset": {"S": "TEST"},
        "timestamp": {"N": "'$TIMESTAMP'"},
        "price": {"N": "1234.56"},
        "updated_at": {"N": "'$TIMESTAMP'"}
    }'

# Verify the table exists
echo "Verifying table creation..."
awslocal dynamodb describe-table --table-name prices

echo "DynamoDB initialization complete"