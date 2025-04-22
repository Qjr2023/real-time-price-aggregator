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



# test in AWS

# Get mock ticker data from three exchanges for asset1
# curl -s http://18.237.209.98:8081/mock/ticker/asset1
# curl -s http://54.190.4.137:8082/mock/ticker/asset1
# curl -s http://35.89.207.182:8083/mock/ticker/asset1

# # Post prices for asset1 and asset10001
# curl -X POST http://54.189.92.91:8080/refresh/asset1
# curl -X POST http://35.163.154.115:8080/refresh/asset10001

# # Get prices for asset1 and asset10001
# curl -s http://54.189.92.91:8080/prices/asset1
# curl -s http://35.163.154.115:8080/prices/asset10001


# check health of the monitoring system
# curl http://<monitoring_ip>:9090/-/healthy

# visit Grafana dashboard
# http://<monitoring_ip>:3000 (username: admin, password: admin)

# check if the data is being stored in DynamoDB
# use AWS CLI to scan the DynamoDB table
# aws dynamodb scan --table-name prices --limit 5
# Testing the cache by requesting the same asset multiple times
# After the first request, the response should be faster
# time curl http://<api_server_ip>:8080/prices/asset3
# time curl http://<api_server_ip>:8080/prices/asset3
# request refresh asset
# curl -X POST http://<api_server_ip>:8080/refresh/asset3

# check logs
# ssh -i your-key.pem ec2-user@<api_server_ip>
# docker logs $(docker ps | grep api-server | awk '{print $1}')


# stop the exchange1 container to simulate a failure
# ssh -i your-key.pem ec2-user@<exchange1_ip>
# sudo docker stop $(docker ps -q)

# Then try to get the price, the system should still work, using data from other exchanges
# curl http://35.165.69.248:8080/prices/asset99