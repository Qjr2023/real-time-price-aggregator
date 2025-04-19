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

// SystemMetrics 收集系统级指标
type SystemMetrics struct {
	// Go 运行时指标
	goRoutines  prometheus.Gauge
	goMemAlloc  prometheus.Gauge
	goMemSys    prometheus.Gauge
	goGCCount   prometheus.Counter
	goGCPauseNs prometheus.Histogram

	// 系统指标
	cpuUsage  prometheus.Gauge
	memUsage  prometheus.Gauge
	diskUsage prometheus.Gauge
	diskIOPS  prometheus.Gauge
	networkIO *prometheus.GaugeVec // Changed from prometheus.GaugeVec to *prometheus.GaugeVec

	// DynamoDB 指标
	dynamoReadLatency  prometheus.Histogram
	dynamoWriteLatency prometheus.Histogram
	dynamoReadUnits    prometheus.Counter
	dynamoWriteUnits   prometheus.Counter
	dynamoErrors       prometheus.Counter
}

// NewSystemMetrics 创建一个新的系统指标收集器
func NewSystemMetrics() *SystemMetrics {
	m := &SystemMetrics{
		// Go 运行时指标
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

		// 系统指标
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
		diskUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_system_disk_usage_percent",
				Help: "Disk usage percentage",
			},
		),
		diskIOPS: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "price_system_disk_iops",
				Help: "Disk IO operations per second",
			},
		),
		networkIO: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "price_system_network_io_bytes",
				Help: "Network IO bytes per second",
			},
			[]string{"direction"}, // "in" or "out"
		),

		// DynamoDB 指标
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

// StartCollecting 开始定期收集系统指标
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

// collectGoMetrics 收集 Go 运行时指标
func (m *SystemMetrics) collectGoMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.goRoutines.Set(float64(runtime.NumGoroutine()))
	m.goMemAlloc.Set(float64(memStats.Alloc))
	m.goMemSys.Set(float64(memStats.Sys))
	m.goGCCount.Add(float64(memStats.NumGC))

	// 这里的 GC 暂停时间计算是简化的，实际中应该更精确
	if memStats.NumGC > 0 {
		m.goGCPauseNs.Observe(float64(memStats.PauseNs[(memStats.NumGC-1)%256]))
	}
}

// collectSystemMetrics 收集系统指标
func (m *SystemMetrics) collectSystemMetrics() {
	// CPU 使用率
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		m.cpuUsage.Set(cpuPercent[0])
	}

	// 内存使用率
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		m.memUsage.Set(memInfo.UsedPercent)
	}

	// 磁盘使用率
	diskInfo, err := disk.Usage("/")
	if err == nil {
		m.diskUsage.Set(diskInfo.UsedPercent)
	}

	// 注意：磁盘IOPS和网络IO需要特定的库和计算
	// 这里只是占位，实际项目中应根据需要实现
	m.diskIOPS.Set(0)
	m.networkIO.WithLabelValues("in").Set(0)
	m.networkIO.WithLabelValues("out").Set(0)
}

// RecordDynamoDBReadLatency 记录 DynamoDB 读取延迟
func (m *SystemMetrics) RecordDynamoDBReadLatency(duration time.Duration) {
	m.dynamoReadLatency.Observe(duration.Seconds())
}

// RecordDynamoDBWriteLatency 记录 DynamoDB 写入延迟
func (m *SystemMetrics) RecordDynamoDBWriteLatency(duration time.Duration) {
	m.dynamoWriteLatency.Observe(duration.Seconds())
}

// RecordDynamoDBReadUnits 记录 DynamoDB 消耗的读取容量单位
func (m *SystemMetrics) RecordDynamoDBReadUnits(units float64) {
	m.dynamoReadUnits.Add(units)
}

// RecordDynamoDBWriteUnits 记录 DynamoDB 消耗的写入容量单位
func (m *SystemMetrics) RecordDynamoDBWriteUnits(units float64) {
	m.dynamoWriteUnits.Add(units)
}

// RecordDynamoDBError 记录 DynamoDB 操作错误
func (m *SystemMetrics) RecordDynamoDBError() {
	m.dynamoErrors.Inc()
}
