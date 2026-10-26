// Command mini-wechat 是后端服务入口：初始化链路追踪与指标、打开 SQLite 数据库、
// 启动 HTTP 服务（API + WebSocket + 前端静态文件），并在收到退出信号时优雅关闭。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"mini-wechat/server/internal/api"
	"mini-wechat/server/internal/observability"
	"mini-wechat/server/internal/realtime"
	"mini-wechat/server/internal/store"
)

// env 读取环境变量，未设置或为空时使用 fallback。
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envFloat 读取浮点型环境变量，解析失败时使用 fallback。
func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return fallback
	}
	return v
}

// dirExists 判断目录是否存在，用于决定是否需要托管前端构建产物。
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func main() {
	port := env("PORT", "8090")
	dbPath := env("DB_PATH", "data/wechat.db")
	uploadsDir := env("UPLOADS_DIR", "uploads")
	secret := env("JWT_SECRET", "mini-wechat-dev-secret-change-me")

	// 链路追踪必须在注册路由之前初始化：Gin 中间件在创建时就会绑定全局 TracerProvider。
	// 未配置 OTEL_EXPORTER_OTLP_ENDPOINT 时导出自动关闭，本地开发无需依赖 Jaeger。
	shutdownTracing, tracingEnabled, err := observability.InitTracing(context.Background(), observability.TracingConfig{
		ServiceName: env("OTEL_SERVICE_NAME", "mini-wechat-server"),
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		SampleRatio: envFloat("OTEL_TRACES_SAMPLER_ARG", 1),
	})
	if err != nil {
		log.Fatalf("init tracing: %v", err)
	}

	// 指标采集器同时供 HTTP 中间件和 WebSocket Hub 使用。
	metrics := observability.NewMetrics()

	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer st.Close()

	// Hub 通过回调获取会话成员，避免 realtime 包反向依赖 store 包。
	hub := realtime.NewHub()
	hub.Members = st.MemberIDs
	hub.Metrics = metrics

	server := api.New(st, hub, metrics, secret, uploadsDir)
	router := server.Router()

	// 前端产物存在时由 Go 进程直接托管，实现单二进制/单容器部署。
	if dist := env("WEB_DIST", "../web/dist"); dirExists(dist) {
		router.Static("/assets", filepath.Join(dist, "assets"))
		router.StaticFile("/favicon.svg", filepath.Join(dist, "favicon.svg"))
		// SPA 回退：非后端路径统一返回 index.html，交给前端路由处理。
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") || strings.HasPrefix(path, "/uploads") {
				c.Status(http.StatusNotFound)
				return
			}
			c.File(filepath.Join(dist, "index.html"))
		})
	}

	log.Printf("Mini WeChat server listening on http://localhost:%s", port)
	log.Printf("Demo accounts: alice / bob / carol / dave / emma, password: 123456")
	if tracingEnabled {
		log.Printf("Tracing enabled: exporting OTLP/HTTP to %s", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	} else {
		log.Printf("Tracing disabled: set OTEL_EXPORTER_OTLP_ENDPOINT to enable")
	}

	srv := &http.Server{Addr: ":" + port, Handler: router}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 优雅关闭：先停止接收新请求，再刷新未导出的 span，避免容器停止时丢链路数据。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if err := shutdownTracing(ctx); err != nil {
		log.Printf("tracing shutdown: %v", err)
	}
}
