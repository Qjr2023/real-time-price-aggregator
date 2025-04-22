// internal/metrics/system_metrics.go
package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// SystemMetrics collects and exposes system metrics
type SystemMetrics struct {
	// Go the runtime metrics
	goRoutines  prometheus.Gauge
	goMemAlloc  prometheus.Gauge
	goMemSys    prometheus.Gauge
	goGCCount   prometheus.Counter
	goGCPauseNs prometheus.Histogram
	lastGCCount uint32

	// system metrics
	cpuUsage  prometheus.Gauge
	memUsage  prometheus.Gauge
	diskUsage *prometheus.GaugeVec

	// DynamoDB metrics
	dynamoReadLatency  prometheus.Histogram
	dynamoWriteLatency prometheus.Histogram
	dynamoReadUnits    prometheus.Counter
	dynamoWriteUnits   prometheus.Counter
	dynamoErrors       prometheus.Counter
}

// NewSystemMetrics creates a new SystemMetrics instance
func NewSystemMetrics() *SystemMetrics {
	m := &SystemMetrics{
		// Go the runtime metrics
		goRoutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_go_goroutines",
				Help: "Number of goroutines",
			},
		),
		goMemAlloc: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_go_memory_allocated_bytes",
				Help: "Bytes allocated by Go runtime",
			},
		),
		goMemSys: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_go_memory_system_bytes",
				Help: "Bytes obtained from system by Go runtime",
			},
		),
		goGCCount: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_go_gc_count_total",
				Help: "Number of garbage collections",
			},
		),
		goGCPauseNs: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "price_go_gc_pause_ns",
				Help:    "Garbage collection pause times in nanoseconds",
				Buckets: prometheus.ExponentialBuckets(1000, 2, 20), // 1us to ~500ms
			},
		),

		// system metrics
		cpuUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_system_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),
		memUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_system_memory_usage_percent",
				Help: "Memory usage percentage",
			},
		),
		diskUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "price_system_disk_usage_percent",
				Help: "Disk usage percentage",
			},
			[]string{"path"},
		),

		// DynamoDB metrics
		dynamoReadLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "price_dynamodb_read_latency_seconds",
				Help:    "DynamoDB read operation latency in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
		),
		dynamoWriteLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "price_dynamodb_write_latency_seconds",
				Help:    "DynamoDB write operation latency in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
		),
		dynamoReadUnits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_dynamodb_read_units_total",
				Help: "DynamoDB read capacity units consumed",
			},
		),
		dynamoWriteUnits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_dynamodb_write_units_total",
				Help: "DynamoDB write capacity units consumed",
			},
		),
		dynamoErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "price_dynamodb_errors_total",
				Help: "DynamoDB operation errors",
			},
		),
	}

	return m
}

// StartCollecting starts collecting system metrics at the specified interval
func (m *SystemMetrics) StartCollecting(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			<-ticker.C
			m.collectGoMetrics()
			m.collectSystemMetrics()
		}
	}()
}

// collectGoMetrics collects Go runtime metrics
func (m *SystemMetrics) collectGoMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.goRoutines.Set(float64(runtime.NumGoroutine()))
	m.goMemAlloc.Set(float64(memStats.Alloc))
	m.goMemSys.Set(float64(memStats.Sys))

	// record GC count and pause times
	currentGCCount := memStats.NumGC
	if m.lastGCCount == 0 {
		m.lastGCCount = currentGCCount // initial value
	}
	m.goGCCount.Add(float64(currentGCCount - m.lastGCCount))

	// record GC pause times
	if currentGCCount > m.lastGCCount {
		startIndex := m.lastGCCount % 256
		endIndex := currentGCCount % 256

		// if endIndex < startIndex, it means we have wrapped around
		if endIndex <= startIndex && currentGCCount-m.lastGCCount > 0 {
			for i := startIndex; i < 256; i++ {
				m.goGCPauseNs.Observe(float64(memStats.PauseNs[i]))
			}
			startIndex = 0
		}

		// record the pause times
		for i := startIndex; i < endIndex; i++ {
			m.goGCPauseNs.Observe(float64(memStats.PauseNs[i]))
		}
	}

	// update lastGCCount
	m.lastGCCount = currentGCCount
}

func (m *SystemMetrics) collectSystemMetrics() {
	// cpu usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		m.cpuUsage.Set(cpuPercent[0])
	}

	// memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		m.memUsage.Set(memInfo.UsedPercent)
	}

	// disk usage
	paths := []string{"/", "/data"} // Add more paths as needed
	for _, path := range paths {
		diskInfo, err := disk.Usage(path)
		if err == nil {
			m.diskUsage.WithLabelValues(path).Set(diskInfo.UsedPercent)
		}
	}
}

// RecordDynamoDBReadLatency records DynamoDB read latency
func (m *SystemMetrics) RecordDynamoDBReadLatency(duration time.Duration) {
	m.dynamoReadLatency.Observe(duration.Seconds())
}

// RecordDynamoDBWriteLatency records DynamoDB write latency
func (m *SystemMetrics) RecordDynamoDBWriteLatency(duration time.Duration) {
	m.dynamoWriteLatency.Observe(duration.Seconds())
}

// RecordDynamoDBReadUnits records DynamoDB consumed read capacity units
func (m *SystemMetrics) RecordDynamoDBReadUnits(units float64) {
	m.dynamoReadUnits.Add(units)
}

// RecordDynamoDBWriteUnits records DynamoDB consumed write capacity units
func (m *SystemMetrics) RecordDynamoDBWriteUnits(units float64) {
	m.dynamoWriteUnits.Add(units)
}

// RecordDynamoDBError records a DynamoDB operation error
func (m *SystemMetrics) RecordDynamoDBError() {
	m.dynamoErrors.Inc()
}
