# Real-Time Price Aggregator - Project Proposal

## Project Overview
A distributed system that aggregates financial asset prices from multiple exchanges in real-time, providing a weighted average price through a high-performance API. The system balances data freshness, low latency, and resource efficiency.

## Key Challenges
- Ensuring price data accuracy across multiple exchanges
- Meeting sub-100ms response time requirements
- Handling varying popularity levels among thousands of assets
- Maintaining availability during exchange outages
- Optimizing system resources efficiently

## Proposed Solution
A tiered architecture with these components:
- **Concurrent Price Fetcher**: Retrieves data from multiple exchanges simultaneously
- **Tiered Refresh Strategy**:
  - Hot assets (top 20): 5-second refresh
  - Medium assets (next 180): 30-second refresh
  - Cold assets (remaining 800): 5-minute refresh
- **Redis Caching**: TTLs aligned with refresh intervals
- **DynamoDB Storage**: Historical price persistence
- **Circuit Breaker Pattern**: Fault tolerance for exchange failures
- **Prometheus/Grafana**: Performance monitoring

## Technology Stack
- Go (backend)
- Redis (caching)
- DynamoDB (storage)
- Docker & Terraform (deployment)
- JMeter (performance testing)

## Performance Goals
- P95 Latency: < 80ms
- Throughput: 1000+ requests/second
- Cache Hit Rate: > 95%
- API Error Rate: < 0.1%

## Implementation Approach
Four development phases:
1. Infrastructure and mock exchanges setup
2. Core services implementation
3. Performance optimization and testing
4. Deployment configuration and documentation

## Expected Outcomes
A scalable system that demonstrates:
- Efficient resource utilization through tiered refresh rates
- High throughput with low latency for popular assets
- Fault tolerance during exchange failures
- Comprehensive performance metrics and analysis