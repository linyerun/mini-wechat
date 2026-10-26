package observability

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "mini-wechat"

// TracingConfig 链路导出配置。
type TracingConfig struct {
	ServiceName string
	Endpoint    string
	SampleRatio float64
}

// InitTracing 配置全局 TracerProvider，返回关闭函数与是否启用。
// Endpoint 为空时不导出 span，但仍安装 W3C 传播器，
// 这样业务代码无需判空就能安全地读取或传递链路上下文。
func InitTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, bool, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	noop := func(context.Context) error { return nil }
	if cfg.Endpoint == "" {
		return noop, false, nil
	}

	opts, err := exporterOptions(cfg.Endpoint)
	if err != nil {
		return noop, false, err
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return noop, false, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", buildVersion()),
	))
	if err != nil {
		return noop, false, fmt.Errorf("build tracing resource: %w", err)
	}

	ratio := cfg.SampleRatio
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	// ParentBased：上游已采样的请求继续采样，根 span 按比例采样。
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(provider)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("otel: %v", err)
	}))
	return provider.Shutdown, true, nil
}

// exporterOptions 同时接受 host:port 和完整 URL。
// WithEndpointURL 会把空路径当作 "/"，OTLP 接收端会返回 404，
// 因此这里显式补上默认的 /v1/traces。
func exporterOptions(endpoint string) ([]otlptracehttp.Option, error) {
	if !strings.Contains(endpoint, "://") {
		return []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		}, nil
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse otlp endpoint: %w", err)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/v1/traces"
	}
	return []otlptracehttp.Option{otlptracehttp.WithEndpointURL(u.String())}, nil
}

// SkipTracePath 排除被高频抓取、又没有业务排查价值的路径，避免制造噪声 span。
func SkipTracePath(r *http.Request) bool {
	path := r.URL.Path
	return path == "/metrics" ||
		path == "/api/health" ||
		path == "/favicon.svg" ||
		strings.HasPrefix(path, "/uploads/") ||
		strings.HasPrefix(path, "/assets/")
}

// Tracing 为每个请求创建服务端 span：延续上游 traceparent，
// 并把 trace ID 通过 X-Trace-ID 返回给客户端，便于在 Jaeger 中定位。
func Tracing() gin.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	return func(c *gin.Context) {
		if SkipTracePath(c.Request) {
			c.Next()
			return
		}

		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)
		// 故意不记录查询串：WebSocket 握手把 JWT 放在 token 参数里。
		attrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("url.path", c.Request.URL.Path),
			attribute.String("url.scheme", scheme(c.Request)),
			attribute.String("url.route", c.FullPath()),
			attribute.String("user_agent.original", c.Request.UserAgent()),
			attribute.String("network.protocol.version", c.Request.Proto),
			attribute.String("client.address", c.ClientIP()),
		}
		if host := c.Request.Host; host != "" {
			attrs = append(attrs, attribute.String("server.address", hostname(host)))
		}

		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
		)
		defer func() {
			status := c.Writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
			span.SetAttributes(attribute.Int("http.response.status_code", status))
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}
			if last := c.Errors.Last(); last != nil {
				span.RecordError(last.Err)
				span.SetStatus(codes.Error, last.Error())
			}
			span.End()
		}()

		c.Request = c.Request.WithContext(ctx)
		if sc := span.SpanContext(); sc.IsValid() {
			c.Header("X-Trace-ID", sc.TraceID().String())
		}
		c.Next()
	}
}

// SetUser 在鉴权成功后给当前 span 补上用户标识。
// 未启用链路追踪时拿到的是 noop span，调用无副作用。
func SetUser(r *http.Request, id int64) {
	trace.SpanFromContext(r.Context()).SetAttributes(attribute.Int64("enduser.id", id))
}

// scheme 优先依据 TLS 判断，其次参考反向代理传来的 X-Forwarded-Proto。
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}

// hostname 去掉 Host 中的端口部分。
func hostname(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
