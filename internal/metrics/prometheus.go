package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsService manages Prometheus metrics for the application
type MetricsService struct {
	// API request metrics
	apiRequests        *prometheus.CounterVec
	apiRequestDuration *prometheus.HistogramVec

	// Cache metrics
	cacheHits        prometheus.Counter
	cacheMisses      prometheus.Counter
	cacheHitsCount   float64    // Internal counter for hits
	cacheMissesCount float64    // Internal counter for misses
	cacheMutex       sync.Mutex // Mutex to protect internal counters

	// Exchange metrics
	exchangeRequests *prometheus.CounterVec
	exchangeErrors   *prometheus.CounterVec
	exchangeDuration *prometheus.HistogramVec

	// Circuit breaker metrics
	circuitBreakerState *prometheus.GaugeVec

	// Refresh metrics
	refreshCount  *prometheus.CounterVec
	refreshErrors *prometheus.CounterVec

	// Asset metrics
	assetAccessCount *prometheus.CounterVec
}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	m := &MetricsService{
		// API metrics
		apiRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_api_requests_total",
				Help: "Total number of API requests",
			},
			[]string{"endpoint", "status"},
		),
		apiRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "price_api_request_duration_seconds",
				Help:    "API request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // From 1ms to ~16s
			},
			[]string{"endpoint"},
		),

		// Cache metrics
		cacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		cacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),

		// Exchange metrics
		exchangeRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_exchange_requests_total",
				Help: "Total number of requests to exchanges",
			},
			[]string{"exchange"},
		),
		exchangeErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_exchange_errors_total",
				Help: "Total number of exchange request errors",
			},
			[]string{"exchange", "error_type"},
		),
		exchangeDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "price_exchange_request_duration_seconds",
				Help:    "Exchange request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
			},
			[]string{"exchange"},
		),

		// Circuit breaker metrics
		circuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "price_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"exchange"},
		),

		// Refresh metrics
		refreshCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_refresh_total",
				Help: "Total number of price refreshes",
			},
			[]string{"tier", "trigger_type"},
		),
		refreshErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_refresh_errors_total",
				Help: "Total number of price refresh errors",
			},
			[]string{"tier"},
		),

		// Asset metrics
		assetAccessCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "price_asset_access_total",
				Help: "Total number of accesses per asset",
			},
			[]string{"asset", "tier"},
		),
	}

	return m
}

// RecordCacheHit records a cache hit
func (m *MetricsService) RecordCacheHit() {
	m.cacheMutex.Lock()
	m.cacheHitsCount++
	m.cacheMutex.Unlock()
	m.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss
func (m *MetricsService) RecordCacheMiss() {
	m.cacheMutex.Lock()
	m.cacheMissesCount++
	m.cacheMutex.Unlock()
	m.cacheMisses.Inc()
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (m *MetricsService) GetCacheHitRate() float64 {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	hits := m.cacheHitsCount
	misses := m.cacheMissesCount
	total := hits + misses

	if total == 0 {
		return 0
	}

	return (hits / total) * 100
}

// RecordAPIRequest records an API request
func (m *MetricsService) RecordAPIRequest(endpoint string, status int) {
	m.apiRequests.WithLabelValues(endpoint, strconv.Itoa(status)).Inc()
}

// ObserveAPIRequestDuration records the duration of an API request
func (m *MetricsService) ObserveAPIRequestDuration(endpoint string, duration time.Duration) {
	m.apiRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// RecordExchangeRequest records a request to an exchange
func (m *MetricsService) RecordExchangeRequest(exchange string) {
	m.exchangeRequests.WithLabelValues(exchange).Inc()
}

// RecordExchangeError records an error from an exchange
func (m *MetricsService) RecordExchangeError(exchange, errorType string) {
	m.exchangeErrors.WithLabelValues(exchange, errorType).Inc()
}

// ObserveExchangeRequestDuration records the duration of an exchange request
func (m *MetricsService) ObserveExchangeRequestDuration(exchange string, duration time.Duration) {
	m.exchangeDuration.WithLabelValues(exchange).Observe(duration.Seconds())
}

// RecordCircuitBreakerState records the state of a circuit breaker
// state: 0=closed, 1=open, 2=half-open
func (m *MetricsService) RecordCircuitBreakerState(exchange string, state int) {
	m.circuitBreakerState.WithLabelValues(exchange).Set(float64(state))
}

// RecordRefresh records a price refresh
// tier: "hot", "medium", "cold"
// triggerType: "auto", "manual", "force"
func (m *MetricsService) RecordRefresh(tier, triggerType string) {
	m.refreshCount.WithLabelValues(tier, triggerType).Inc()
}

// RecordRefreshError records a price refresh error
func (m *MetricsService) RecordRefreshError(tier string) {
	m.refreshErrors.WithLabelValues(tier).Inc()
}

// RecordAssetAccess records an access to an asset
func (m *MetricsService) RecordAssetAccess(asset, tier string) {
	m.assetAccessCount.WithLabelValues(asset, tier).Inc()
}
