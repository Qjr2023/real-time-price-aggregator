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
- Supports 10,000 assets loaded from `symbols.csv`

**Key Changes (Updated Design):**
- `GET /prices/{asset}` now retrieves prices directly from Redis cache (no fetcher calls).
- Assets are validated against a predefined list in `symbols.csv` (10,000 assets).
- Manual `POST /refresh/{asset}` for price refresh, with plans for future automation.

## 🏗️ System Architecture

### Mock Exchanges
The system simulates three exchanges to mimic real-world trading platforms:
- **Exchange 1**: Runs on `http://localhost:8081/mock/ticker/{symbol}`
- **Exchange 2**: Runs on `http://localhost:8082/mock/ticker/{symbol}`
- **Exchange 3**: Runs on `http://localhost:8083/mock/ticker/{symbol}`

Each exchange supports 10,000 assets defined in `symbols.csv` and provides mock price and timestamp data.

### Data Flow
1. The main server fetches price data from the three mock exchanges (via `POST /refresh/{asset}`).
2. Prices are stored in Redis (cache) and DynamoDB (persistent storage).
3. Users query the aggregated price via `GET /prices/{asset}`, which retrieves data from Redis.

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
- **Database**: DynamoDB (via AWS DynamoDB)
- **Testing**: Apache JMeter (for load and latency testing)
- **Deployment**: Docker and Docker Compose

**Note**: Kafka, MongoDB, and Prometheus + Grafana were part of the initial design but were omitted to simplify the project.

## ⚖️ Design Trade-offs
- **Latency vs Cost**: Caching with Redis minimizes latency, at the expense of slight data staleness (5-minute TTL).
- **Availability vs Consistency**: Prioritizes availability—if Redis cache misses, returns 404.
- **Scalability**: Supports 10,000 assets, with plans for distributed processing (e.g., Kafka) in the future.

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

2. **Prepare `symbols.csv`**:
   - Ensure `symbols.csv` exists in the project root with 10,000 asset symbols:
     ```csv
     symbol
     btcusdt
     ethusdt
     adausdt
     ...
     symbol10000
     ```

3. **Start All Services with Docker Compose**:
   ```bash
   docker-compose up --build
   ```
   This command will:
   - Build the Go services (main server and mock exchanges).
   - Start Redis on port `6379`.
   - Start the three mock exchanges on ports `8081`, `8082`, and `8083`.
   - Start the main server on port `8080`.

### API Design
#### Endpoints
- **GET /prices/{asset}**  
  - **Description**: Retrieve the cached price of an asset.
  - **Parameters**:
     - `asset` (path parameter): Asset symbol (e.g., `btcusdt`).
  - **Responses**:
    - **200**: Success
      ```json
      {
        "asset": "btcusdt",
        "price": 79450.12,
        "last_updated": 1696118400
      }
      ```
    - **400**: Invalid asset symbol
      ```json
      {"msg": "Invalid asset symbol"}
      ```
    - **404**: Asset not found in cache
      ```json
      {"msg": "Asset not found"}
      ```

- **POST /refresh/{asset}**  
  - **Description**: Manually refresh the price of an asset.
  - **Parameters**:
    - `asset` (path parameter): Asset symbol (e.g., `btcusdt`).
  - **Responses**:
    - **200**: Success
      ```json
      {
        "message": "Price for btcusdt refreshed"
      }
      ```
    - **400**: Invalid asset symbol
      ```json
      {"msg": "Invalid asset symbol"}
      ```
    - **404**: Asset not found
      ```json
      {"msg": "Asset not found"}
      ```

#### Logic Flow
- **GET /prices/{asset}**:
  1. Validate the asset against `symbols.csv`.
  2. Retrieve price from Redis cache.
  3. If cache miss, return 404.
- **POST /refresh/{asset}**:
  1. Validate the asset against `symbols.csv`.
  2. Fetch mock price data (random price for simplicity).
  3. Update Redis and DynamoDB.
  4. Return confirmation message.

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
   - Test the three mock exchanges (`8081`, `8082`, `8083`).
   - Test the `GET /prices/btcusdt` endpoint.
   - Test the `POST /refresh/btcusdt` endpoint.
   - Verify that price data is stored in DynamoDB.

#### Manual Testing  
1. **Test the Mock Exchanges**:
   ```bash
   curl http://localhost:8081/mock/ticker/btcusdt
   ```
   Expected response:
   ```json
   {
     "symbol": "btcusdt",
     "price": 79850.12,
     "timestamp": 1696118400
   }
   ```

2. **Test the API**:
   - Refresh the price of an asset:
     ```bash
     curl -X POST http://localhost:8080/refresh/btcusdt
     ```
     Expected response:
     ```json
     {
       "message": "Price for btcusdt refreshed"
     }
     ```
   - Get the price of an asset:
     ```bash
     curl http://localhost:8080/prices/btcusdt
     ```
     Expected response:
     ```json
     {
       "asset": "btcusdt",
       "price": 79450.12,
       "last_updated": 1696118400
     }
     ```

3. **Check DynamoDB Data**:
   Verify that price data is stored in DynamoDB:
   ```bash
   aws dynamodb scan --table-name prices --region us-west-2
   ```
   Expected output:
   ```json
   {
     "Items": [
       {
         "asset": {"S": "btcusdt"},
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

### Deployment to AWS EC2
1. **Create an EC2 Instance**:
   - Choose `t3.medium` (2 vCPU, 4 GiB memory).
   - Configure security group to allow inbound traffic on ports `8080`, `8081`, `8082`, `8083`.

2. **Install Docker and Docker Compose**:
   ```bash
   sudo apt update
   sudo apt install docker.io docker-compose -y
   sudo systemctl start docker
   sudo systemctl enable docker
   sudo usermod -aG docker ec2-user
   ```

3. **Configure DynamoDB Access**:
   - Option 1: Use IAM Role (recommended):
     - Create an IAM role with `AmazonDynamoDBFullAccess` policy.
     - Attach the role to the EC2 instance.
   - Option 2: Copy AWS credentials:
     ```bash
     scp -i your-key.pem ~/.aws/credentials ec2-user@your-ec2-ip:/home/ec2-user/.aws/
     scp -i your-key.pem ~/.aws/config ec2-user@your-ec2-ip:/home/ec2-user/.aws/
     ```

4. **Deploy with Docker Compose**:
   - Upload the project to EC2:
     ```bash
     scp -i your-key.pem -r ./real-time-price-aggregator ec2-user@your-ec2-ip:/home/ec2-user/
     ```
   - Run Docker Compose:
     ```bash
     cd /home/ec2-user/real-time-price-aggregator
     docker-compose up -d
     ```

5. **Test the Deployment**:
   ```bash
   curl -X POST http://your-ec2-ip:8080/refresh/btcusdt
   curl http://your-ec2-ip:8080/prices/btcusdt
   ```

## Research
- [CoinGecko](https://www.coingecko.com/) - Used for understanding price aggregation and market data.
- [TradingView](https://www.tradingview.com/) - Reference for real-time financial data visualization.

## Project Updates
For a detailed overview of my individual contributions, including literature review, project backlog, code, documentation, testing, and future plans, please see the following updates:

- [Project Update 1](docs/project_update/Project_Update_1.md): Initial implementation, automation with Docker Compose, and testing plan.


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
│   ├── storage/
│   │   └── dynamodb.go
│   └── types/
│       └── types.go
├── mocks/
│   │── mock_server.go
│   └──Dockerfile
├── symbols.csv
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
