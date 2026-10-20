package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal   *prometheus.CounterVec   // 带标签的计数器
	httpRequestDuration *prometheus.HistogramVec // 带标签的直方图
	batchQueueSize      prometheus.Gauge
	batchSize           prometheus.Histogram
	// TODO: add jobOutcome *prometheus.CounterVec with label "result"
	// (delivered / orphaned / cancelled) to expose orphaned requests.
)

// Init 小写的 init 函数会在程序跑起来之前隐式调用
// 这里我们选择显示调用
// Init explicitly registers all metrics. It is called from main() once at startup.
// NOTE: MustRegister panics on duplicate registration; calling Init twice
// (e.g. in tests) will crash. Consider sync.Once or a package-level guard.
func Init() {
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests"}, // Help is exported via /metrics, keep it ASCII/English.
		// WARNING: "path" must be a route template (e.g. "/api/v1/check"),
		// never the raw URL path. Raw paths explode label cardinality
		// (e.g. /user/123, /user/124 -> unbounded series) and can OOM Prometheus.
		[]string{"method", "path", "status"})
	// MustRegister 把指标注册到一个全局的“指标注册表”（Registry）里。
	// 这个注册表本质上就是一个 map，由 Prometheus 客户端库内部维护。
	// /metrics 接口被请求时，promhttp.Handler() 会去遍历这个注册表，把里面所有已注册的指标序列化成文本格式，返回给 Prometheus。
	// 没注册的指标，/metrics 里就看不到。
	prometheus.MustRegister(httpRequestsTotal)

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5}},
		// NOTE: no "status" label here, so you cannot compute p99 latency
		// for 5xx vs 2xx separately. Consider adding it.
		[]string{"method", "path"})
	prometheus.MustRegister(httpRequestDuration)

	// Gauge is set only by the batcher tick. If the batcher is blocked in
	// processBatch (up to 10s), this Gauge is stale while the queue keeps
	// growing -> dashboards under-report pressure. Prefer Inc/Dec at
	// enqueue/dequeue points instead of periodic Set().
	batchQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "batch_queue_size",
		Help: "Number of requests currently waiting in the job queue",
	})
	prometheus.MustRegister(batchQueueSize)

	batchSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "batch_size",
		Help:    "Number of prompts sent to the model service per batch",
		Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
	})
	prometheus.MustRegister(batchSize)
}

// 创建两个导出函数，通过中间件来调用这两个函数来对指标对象进行修改
// .WithLabelValues() 这个函数相当于带自动创建的 map 查找，负责定位

func RecordRequest(method, path, status string) {
	// 这个函数会根据 httpRequestsTotal 的参数数量来确定 WithLabelValues 里需要几个参数，少了会报错，但要是位置传反了不管！！！
	// NOTE: argument order is checked only by count, not by meaning.
	// A swapped (method, path) silently produces garbage labels; add a unit test.
	httpRequestsTotal.WithLabelValues(method, path, status).Inc() // 计数 +1，因为只是用来计数的
}

func ObserveDuration(method, path string, duration float64) {
	httpRequestDuration.WithLabelValues(method, path).Observe(duration) // Observe 函数负责记录 duration 的值
}

func SetQueueSize(size int) {
	batchQueueSize.Set(float64(size))
}

func ObserveBatchSize(size int) {
	batchSize.Observe(float64(size))
}
