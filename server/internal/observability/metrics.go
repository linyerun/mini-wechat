// Package observability 提供 Prometheus 指标采集与 OpenTelemetry 链路追踪能力。
package observability

import (
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 持有独立的 Prometheus Registry，
// 对外暴露的内容完全由本服务声明，外加 Go 运行时与进程指标。
type Metrics struct {
	registry *prometheus.Registry

	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge

	wsActive  prometheus.Gauge
	wsTotal   prometheus.Counter
	wsSent    *prometheus.CounterVec
	wsDropped *prometheus.CounterVec

	messagesCreated *prometheus.CounterVec
	buildInfo       *prometheus.GaugeVec
}

// NewMetrics 创建并注册全部指标。使用独立 Registry 而非全局默认注册表，
// 避免引入依赖库偷偷注册的指标。
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		registry: reg,
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mini_wechat",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, route and status code.",
		}, []string{"method", "route", "status"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "mini_wechat",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds by method and route.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
		requestsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "mini_wechat",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently being served.",
		}),
		wsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "mini_wechat",
			Subsystem: "ws",
			Name:      "connections_active",
			Help:      "Number of open WebSocket connections.",
		}),
		wsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mini_wechat",
			Subsystem: "ws",
			Name:      "connections_total",
			Help:      "Total number of accepted WebSocket connections.",
		}),
		wsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mini_wechat",
			Subsystem: "ws",
			Name:      "events_sent_total",
			Help:      "Total number of WebSocket events queued for delivery.",
		}, []string{"event"}),
		wsDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mini_wechat",
			Subsystem: "ws",
			Name:      "events_dropped_total",
			Help:      "Total number of WebSocket events dropped because a client buffer was full.",
		}, []string{"event"}),
		messagesCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mini_wechat",
			Subsystem: "chat",
			Name:      "messages_created_total",
			Help:      "Total number of chat messages persisted by message type.",
		}, []string{"type"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mini_wechat",
			Name:      "build_info",
			Help:      "Build information of the running server.",
		}, []string{"version", "go_version"}),
	}
	m.buildInfo.WithLabelValues(buildVersion(), runtime.Version()).Set(1)

	reg.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.wsActive,
		m.wsTotal,
		m.wsSent,
		m.wsDropped,
		m.messagesCreated,
		m.buildInfo,
	)
	return m
}

// Handler 返回供 /metrics 挂载的抓取处理器。
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware 记录请求数、耗时与在途请求数。
// route 标签使用 Gin 匹配到的路由模板（如 /api/conversations/:id），
// 而不是原始路径，避免标签基数随会话 ID 无限膨胀。
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		m.requestsInFlight.Inc()
		defer func() {
			m.requestsInFlight.Dec()
			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			status := c.Writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
			m.requestsTotal.WithLabelValues(c.Request.Method, route, strconv.Itoa(status)).Inc()
			m.requestDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
		}()
		c.Next()
	}
}

// WSConnected 记录一次新的 WebSocket 连接。
func (m *Metrics) WSConnected() {
	if m == nil {
		return
	}
	m.wsTotal.Inc()
	m.wsActive.Inc()
}

// WSDisconnected 记录连接关闭，与 WSConnected 成对调用。
func (m *Metrics) WSDisconnected() {
	if m == nil {
		return
	}
	m.wsActive.Dec()
}

// WSEventSent 记录成功写入发送队列的事件。
func (m *Metrics) WSEventSent(event string) {
	if m == nil {
		return
	}
	m.wsSent.WithLabelValues(event).Inc()
}

// WSEventDropped 记录因客户端发送缓冲写满而被丢弃的事件，
// 该指标持续增长通常意味着某个连接已经卡死或消费过慢。
func (m *Metrics) WSEventDropped(event string) {
	if m == nil {
		return
	}
	m.wsDropped.WithLabelValues(event).Inc()
}

// MessageCreated 记录一条落库成功的聊天消息。
func (m *Metrics) MessageCreated(kind string) {
	if m == nil {
		return
	}
	m.messagesCreated.WithLabelValues(kind).Inc()
}

// buildVersion 读取构建信息；go run 或本地构建时版本为 (devel)，统一记作 dev。
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
