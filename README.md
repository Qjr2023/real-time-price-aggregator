# Real-Time Price Aggregator

A scalable distributed system that aggregates real-time financial asset prices from multiple trading platforms. Designed to handle high-concurrency user queries with low latency and high availability, while balancing cost and data freshness.

## 🧠 Project Overview

This system fetches prices for financial assets (e.g., BTC, TSLA, ETH) from multiple APIs, caches the latest data, and provides users with the best aggregated quote in real time.

**Key Features:**
- High-throughput API fetchers (e.g., Binance, Coinbase, etc.)
- Kafka-based distributed data ingestion
- Redis for real-time price caching
- MongoDB / DynamoDB for historical trend storage
- Fault-tolerant and horizontally scalable microservices
- Prometheus + Grafana monitoring stack

## 🏗️ System Architecture


## 🔧 Tech Stack

- **Backend:** Go / Java (microservices)
- **Queue:** Kafka
- **Cache:** Redis
- **Database:** MongoDB or DynamoDB
- **Monitoring:** Prometheus + Grafana
- **Testing:** Apache JMeter (for load and latency testing)
- **Deployment:** Docker + AWS EC2 / Load Balancer

## ⚖️ Design Trade-offs

- **Latency vs Cost:** Caching with Redis minimizes API cost and latency, at the cost of slight data staleness.
- **Availability vs Consistency:** The system prioritizes availability—if an API source fails, it serves partial or last known data.
- **Hot vs Cold Data:** Frequently queried assets are refreshed more often; others updated on-demand.

## 📊 Performance Goals

- Handle 1000+ concurrent queries/sec
- Sub-100ms response time for hot assets
- >90% Redis cache hit rate under load

## 🚀 How to Run

> _Coming soon: Setup instructions, Docker Compose file, and load test configs._

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

**Author:** Fang Liu
**Course:** Scalable Distributed Systems Final Project  
