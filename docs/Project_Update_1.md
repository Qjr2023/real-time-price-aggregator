# Project Update 1: Real-Time Price Aggregator

## Overview
This update summarizes my work to the **Real-Time Price Aggregator** project for the CS6650 Scalable Distributed Systems course. The project is a distributed system that aggregates real-time financial asset prices from multiple mock exchanges, caches them in Redis for low-latency access, and stores historical data in DynamoDB for persistence. I completed the core functionality, focusing on scalability, automation, and performance.

## Literature Review
To inform the design of my price aggregation system, I researched the following platforms:
- **CoinGecko (https://www.coingecko.com/)**: A leading cryptocurrency data aggregator that fetches prices from multiple exchanges and calculates a weighted average based on trading volume. This inspired my decision to use a weighted average for price aggregation, ensuring the aggregated price reflects market consensus.
- **TradingView (https://www.tradingview.com/)**: A real-time financial data platform that visualizes price trends. It highlighted the importance of low-latency data access for user queries, leading me to implement Redis caching with a 5-second TTL to balance freshness and performance.

These platforms helped me understand key trade-offs in distributed systems, such as latency vs. data freshness and availability vs. consistency, which shaped my design decisions.

## Project Backlog
### Completed Tasks
- **System Design**: Architected a microservices-based system with three mock exchanges, a price aggregation server, Redis caching, and DynamoDB storage.
- **Mock Exchanges**: Implemented three mock exchanges in Go (`mocks/mock_server.go`) to simulate real-world trading platforms, supporting the top 10 cryptocurrencies by market cap (e.g., BTC, ETH, USDT).
- **Price Aggregation**: Built the main server (`cmd/main.go`) to fetch prices from exchanges, calculate a weighted average based on trading volume, and manage caching/storage.
- **Redis Caching**: Integrated Redis (`internal/cache/redis.go`) to cache prices for low-latency access, with a 5-second TTL to ensure data freshness.
- **DynamoDB Storage**: Integrated DynamoDB (`internal/storage/dynamodb.go`) for persistent storage, with automatic table creation on startup to simplify setup.
- **Automation**: Used Docker Compose (`docker-compose.yml`) to automate the deployment of Redis, Localstack (DynamoDB), three mock exchanges, and the main server, allowing all services to start with a single command.
- **API Design**: Defined a REST API with two endpoints:
  - `GET /prices/{asset}`: Retrieve the aggregated price of an asset.
  - `POST /refresh/{asset}`: Manually refresh the price of an asset.
- **Documentation**: Documented the API using OpenAPI 3.0 (`openapi.yaml`) and provided detailed setup instructions in the README.
- **Testing**: Conducted manual tests with `curl` to verify API functionality and DynamoDB storage, and planned load testing with JMeter.

### Remaining Tasks
- **Load Testing**: Complete load testing with Apache JMeter to verify performance goals (1000+ queries/sec, <100ms response time, >90% Redis cache hit rate).
- **Screenshots**: Add screenshots of API responses, DynamoDB data, and JMeter results to this report for visual documentation.
- **Final Documentation**: Polish the documentation and prepare for submission.

### Future Work
- **Kafka Integration**: Add Kafka for distributed data ingestion to decouple price fetching and processing, improving scalability for real-time price streams.
- **Monitoring**: Integrate Prometheus and Grafana for monitoring system performance and visualizing price trends.
- **AWS Deployment**: Deploy the system to AWS (e.g., using EC2, ECS, or EKS) with a load balancer to handle production-level traffic.
- **Advanced Aggregation**: Support additional aggregation methods, such as lowest bid and highest ask prices, to provide more options for users.

## Code
The project is written in Go and consists of the following key components:
- **Mock Exchanges** (`mocks/mock_server.go`): Simulate three exchanges running on ports 8081, 8082, and 8083, providing price, volume, and timestamp data for the top 10 cryptocurrencies.
- **Main Server** (`cmd/main.go`): Fetches prices from exchanges, calculates weighted averages, and manages caching/storage.
- **Fetcher** (`internal/fetcher/fetcher.go`): Handles parallel price fetching from multiple exchanges using goroutines for efficiency.
- **Redis Cache** (`internal/cache/redis.go`): Caches prices for low-latency access with a 5-second TTL.
- **DynamoDB Storage** (`internal/storage/dynamodb.go`): Stores historical price data with automatic table creation on startup.
- **API Handler** (`internal/api/handler.go`): Implements the REST API endpoints with proper error handling.

### Code Snippet: Weighted Average Calculation
Here’s how the weighted average is calculated in `internal/fetcher/fetcher.go`:
```go
var totalPrice, totalVolume float64
var latestTimestamp int64
for data := range prices {
    totalPrice += data.Price * data.Volume
    totalVolume += data.Volume
    if data.Timestamp > latestTimestamp {
        latestTimestamp = data.Timestamp
    }
}
if totalVolume == 0 {
    return AggregatedPrice{}, fmt.Errorf("no valid data received")
}
return AggregatedPrice{
    Price:     totalPrice / totalVolume,
    Timestamp: latestTimestamp,
}, nil
```
## Documentation
- **README**: Includes setup instructions, API design, data structure, and testing steps.
- **OpenAPI Spec** (`openapi.yaml`): Documents the API endpoints with request/response formats:
  - `GET /prices/{asset}`: Returns the aggregated price with a human-readable timestamp.
  - `POST /refresh/{asset}`: Forces a price refresh and updates cache/storage.
- **Data Structure**: Documented the DynamoDB table structure in the README:
  - Table Name: `prices`
  - Fields: `asset` (partition key), `timestamp` (sort key), `price`, `updated_at`.

## Testing
### Manual Testing
- **Mock Exchanges**:
  ```bash
  curl http://localhost:8081/mock/ticker/BTCUSDT
  ```
  Response:
  ```json
  {
    "symbol": "BTCUSDT",
    "price": 79850.12,
    "volume": 4500000000,
    "timestamp": 1696118400
  }
  ```
- **API Endpoints**:
  - Get price:
    ```bash
    curl http://localhost:8080/prices/BTCUSDT
    ```
    Response:
    ```json
    {
      "asset": "BTCUSDT",
      "price": 79450.12,
      "last_updated": "2023-10-01 12:00:00"
    }
    ```
  - Refresh price:
    ```bash
    curl -X POST http://localhost:8080/refresh/BTCUSDT
    ```
    Response:
    ```json
    {
      "message": "Price for BTCUSDT refreshed"
    }
    ```
- **DynamoDB Storage**:
  ```bash
  aws --endpoint-url=http://localhost:4566 dynamodb scan --table-name prices
  ```
  Output:
  ```json
  {
    "Items": [
      {
        "asset": {"S": "BTCUSDT"},
        "timestamp": {"N": "1696118400"},
        "price": {"N": "79450.12"},
        "updated_at": {"N": "1696118405"}
      }
    ]
  }
  ```

### Load Testing (Planned)
- Planned to use Apache JMeter to test the `GET /prices/{asset}` endpoint with 1000 concurrent users.
- Performance goals:
  - Throughput: 1000+ queries/sec
  - Response Time: <100ms for cached requests
  - Redis Cache Hit Rate: >90%
- Test plan is included in the README under "Testing the System."

## Challenges Overcome
- **Tight Timeline**: Simplified the original design by omitting Kafka, Prometheus, and Grafana to focus on core functionality within two weeks.
- **Manual Setup**: Initially required manual setup of DynamoDB and services; I automated this by integrating table creation in code and using Docker Compose for service deployment.
- **Learning Curve**: Quickly learned Redis, DynamoDB, and Docker Compose to build a functional system, leveraging Grok’s guidance to accelerate the process.

## Skills Gained
- **Distributed Systems**: Designed a microservices-based system with caching and persistent storage.
- **Go Programming**: Wrote efficient Go code for concurrent price fetching and API handling.
- **Docker and Automation**: Used Docker Compose to automate service deployment, making the system easy to run.
- **API Design**: Created a REST API and documented it with OpenAPI 3.0.
- **Testing**: Planned and executed manual tests, with load testing in progress.

## Future Work
- **Kafka Integration**: Add Kafka for distributed data ingestion to decouple price fetching and processing, improving scalability for real-time price streams.
- **Monitoring**: Integrate Prometheus and Grafana for monitoring system performance and visualizing price trends.
- **AWS Deployment**: Deploy the system to AWS (e.g., using EC2, ECS, or EKS) with a load balancer to handle production-level traffic.
- **Advanced Features**: Support additional aggregation methods (e.g., lowest bid, highest ask) and add more cryptocurrencies.

## Conclusion
This project showcases my ability to design, implement, and test a scalable distributed system under tight constraints. I learned valuable skills in Go programming, microservices architecture, Docker automation, and API design, which I believe will be highly applicable in a professional software engineering role. The automation with Docker Compose, integration of Redis and DynamoDB, and focus on performance make this project a strong demonstration of my technical and problem-solving abilities.