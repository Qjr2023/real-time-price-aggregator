# Real-Time Price Aggregator

A scalable distributed system that aggregates real-time financial asset prices from multiple trading platforms. Designed to handle high-concurrency user queries with low latency and high availability, while balancing cost and data freshness.

## 🧠 Project Overview

This system fetches prices for financial assets (e.g., BTC, ETH) from multiple mock APIs, caches the latest data in Redis, and provides users with the aggregated price in real time using a weighted average based on trading volume. Historical price data is stored in DynamoDB for persistence.

**Key Features:**
- High-throughput API fetchers from three mock exchanges
- Redis for real-time price caching
- DynamoDB for historical price storage
- Fault-tolerant and horizontally scalable microservices
- Tested with Apache JMeter for load and latency

## 🏗️ System Architecture

### Mock Exchanges
The system simulates three exchanges to mimic real-world trading platforms:
- **Exchange 1**: Runs on `http://localhost:8081/mock/ticker/{symbol}`
- **Exchange 2**: Runs on `http://localhost:8082/mock/ticker/{symbol}`
- **Exchange 3**: Runs on `http://localhost:8083/mock/ticker/{symbol}`

Each exchange supports the top 10 cryptocurrencies by market cap (e.g., BTC, ETH, USDT, etc.) and provides price, volume, and timestamp data.

### Data Flow
1. The main server fetches price data from the three mock exchanges.
2. Prices are aggregated using a weighted average based on trading volume.
3. Aggregated prices are cached in Redis and stored in DynamoDB.
4. Users can query the aggregated price via the API or manually refresh it.

### Data Structure
#### DynamoDB Table
- **Table Name**: `prices`
- **Structure**:

| Field Name  | Type   | Description                  | Example Value  |
|-------------|--------|------------------------------|----------------|
| asset       | String | Partition key, asset symbol  | BTCUSDT        |
| timestamp   | Number | Sort key, exchange timestamp | 1696118400     |
| price       | Number | Weighted average price       | 79450.12       |
| updated_at  | Number | Record update time (system)  | 1696118405     |

**Note**: The table is automatically created when the server starts, using code in `internal/storage/dynamodb.go`.

## 🔧 Tech Stack
- **Backend**: Go (microservices)
- **Cache**: Redis
- **Database**: DynamoDB (via Localstack for local testing)
- **Testing**: Apache JMeter (for load and latency testing)
- **Deployment**: Docker and Docker Compose

**Note**: Kafka, MongoDB, and Prometheus + Grafana were part of the initial design but were omitted to simplify the project.

## ⚖️ Design Trade-offs
- **Latency vs Cost**: Caching with Redis minimizes API call costs and latency, at the expense of slight data staleness (5-second TTL).
- **Availability vs Consistency**: Prioritizes availability—if an exchange fails, the system serves partial or cached data.
- **Hot vs Cold Data**: Frequently queried assets are cached in Redis; historical data is stored in DynamoDB.

## 📊 Performance Goals
- Handle 1000+ concurrent queries/sec
- Sub-100ms response time for hot assets
- >90% Redis cache hit rate under load

## 🚀 How to Run

### Prerequisites
1. Install [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/).
2. Install [Go](https://golang.org/doc/install) (version 1.23 or later) for local development (optional if using Docker).
3. Install `redis-cli` for testing:
   - On macOS: `brew install redis`
   - On Ubuntu: `sudo apt-get install redis-tools`
   - On Windows (WSL): `sudo apt-get install redis-tools`
4. Install `aws` CLI for interacting with Localstack:
   - Follow instructions at [AWS CLI Installation](https://aws.amazon.com/cli/).
   - On macOS: `brew install awscli`
5. Install `curl` for making HTTP requests:
   - On macOS: `brew install curl`
   - On Ubuntu: `sudo apt-get install curl`
   - On Windows (WSL): `sudo apt-get install curl`

### Setup Instructions
1. **Clone the Repository**:
   ```bash
   git clone https://github.com/Qjr2023/real-time-price-aggregator.git
   cd real-time-price-aggregator

2. **Initialize Go Modules** (if running locally):
   ```bash
   go mod init real-time-price-aggregator
   go mod tidy
   ```

3. **Start All Services with Docker Compose**:
   ```bash
   docker-compose up --build
   ```
   This command will:
   - Build the Go services (main server and mock exchanges).
   - Start Redis on port `6379`.
   - Start Localstack (DynamoDB) on port `4566`.
   - Start the three mock exchanges on ports `8081`, `8082`, and `8083`.
   - Start the main server on port `8080`.
   - Automatically create the DynamoDB table `prices` when the server starts.

### API Design
#### Endpoints
- **GET /prices/{asset}**  
  - **Description**: Retrieve the aggregated price (weighted average) of an asset.
  - **Parameters**:
    - `asset` (path parameter): Asset symbol (e.g., `BTCUSDT`).
  - **Response**:
    ```json
    {
      "asset": "BTCUSDT",
      "price": 79450.12,
      "last_updated": "2023-10-01 12:00:00"
    }
    ```

- **POST /refresh/{asset}**  
  - **Description**: Manually refresh the price of an asset.
  - **Parameters**:
    - `asset` (path parameter): Asset symbol (e.g., `BTCUSDT`).
  - **Response**:
    ```json
    {
      "message": "Price for BTCUSDT refreshed"
    }
    ```

#### Logic Flow
- **GET /prices/{asset}**:
  1. Check Redis cache.
  2. If cache hit, return cached price.
  3. If cache miss, fetch data from the three exchanges, calculate the weighted average.
  4. Update Redis and DynamoDB.
  5. Return the result.
- **POST /refresh/{asset}**:
  1. Forcefully fetch data from the three exchanges, calculate the weighted average.
  2. Update Redis and DynamoDB.
  3. Return confirmation message.

### Testing the System
#### Automated Testing with `test.sh`
A `test.sh` script is provided to automate testing of the system components.

1. **Make the Script Executable**:
   ```bash
   chmod +x test.sh
   ```

2. **Run the Test Script**:
   After starting the services with `docker-compose up --build`, run:
   ```bash
   ./test.sh
   ```
   The script will:
   - Test Redis connectivity.
   - Test Localstack (DynamoDB) and verify the `prices` table exists.
   - Test the three mock exchanges (`8081`, `8082`, `8083`).
   - Test the `GET /prices/BTCUSDT` endpoint.
   - Test the `POST /refresh/BTCUSDT` endpoint.
   - Verify that price data is stored in DynamoDB.

#### Manual Testing  
1. **Test the Mock Exchanges**:
   Use `curl` or a browser to test an exchange:
   ```bash
   curl http://localhost:8081/mock/ticker/BTCUSDT
   ```
   Expected response:
   ```json
   {
     "symbol": "BTCUSDT",
     "price": 79850.12,
     "volume": 4500000000,
     "timestamp": 1696118400
   }
   ```

2. **Test the API**:
   - Get the price of an asset:
     ```bash
     curl http://localhost:8080/prices/BTCUSDT
     ```
     Expected response:
     ```json
     {
       "asset": "BTCUSDT",
       "price": 79450.12,
       "last_updated": "2023-10-01 12:00:00"
     }
     ```
   - Refresh the price of an asset:
     ```bash
     curl -X POST http://localhost:8080/refresh/BTCUSDT
     ```
     Expected response:
     ```json
     {
       "message": "Price for BTCUSDT refreshed"
     }
     ```

3. **Check DynamoDB Data**:
   Verify that price data is stored in DynamoDB:
   ```bash
   aws --endpoint-url=http://localhost:4566 dynamodb scan --table-name prices
   ```
   Expected output:
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

4. **Load Testing with Apache JMeter**:
   - Download and install [Apache JMeter](https://jmeter.apache.org/).
   - Open JMeter and create a new test plan.
   - Add a Thread Group:
     - Number of Threads: 1000 (users)
     - Ramp-up Period: 10 seconds
     - Loop Count: 10
   - Add an HTTP Request under the Thread Group:
     - Server Name: `localhost`
     - Port: `8080`
     - Path: `/prices/BTCUSDT`
     - Method: `GET`
   - Add Listeners (e.g., View Results Tree, Summary Report) to analyze results.
   - Run the test and verify:
     - Throughput: 1000+ queries/sec
     - Response Time: <100ms for cached requests
     - Redis Cache Hit Rate: >90%

## Research
- [CoinGecko](https://www.coingecko.com/) - Used for understanding price aggregation and market data.
- [TradingView](https://www.tradingview.com/) - Reference for real-time financial data visualization.

## Project Updates
For a detailed overview of my individual contributions, including literature review, project backlog, code, documentation, testing, and future plans, please see the following updates:

- [Project Update 1](docs/Project_Update_1.md): Initial implementation, automation with Docker Compose, and testing plan.


### Project Structure
```
real-time-price-aggregator/
├── cmd/
│   └── main.go
├── internal/
│   ├── api/
│   │   └── handler.go
│   ├── fetcher/
│   │   └── fetcher.go
│   ├── cache/
│   │   └── redis.go
│   └── storage/
│       └── dynamodb.go
├── mocks/
│   └── mock_server.go
├── openapi.yaml
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── test.sh
└── README.md
```

## 📄 License
This project is licensed under the [MIT License](LICENSE).
