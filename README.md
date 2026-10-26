# 微信 Mini

一个使用 **Go + Vue 3 + TypeScript** 搭建的本地版微信演示项目。支持注册登录、单聊、群聊、实时消息、图片消息、朋友圈和资料编辑，桌面端为三栏布局，移动端为底部四标签布局。

## 功能

- 账号注册 / 登录（JWT，密码 bcrypt 加密）
- 单聊与群聊，会话列表、未读角标、已读标记
- WebSocket 实时收发消息、输入中提示、断线自动重连
- 文字、Emoji、图片消息，图片最大 8MB
- 朋友圈：发布动态、多图、点赞、评论
- 通讯录、发起单聊、创建群聊
- 个人资料编辑（昵称、签名、头像上传）
- 退出登录：侧边栏头像菜单与「我」页面均可退出，二次确认后断开 WebSocket 并清空本地会话
- 内置演示账号与种子聊天记录、朋友圈数据
- 响应式界面：桌面端微信三栏布局，窄屏自动切换为移动端布局

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.26、Gin、gorilla/websocket、golang-jwt/v5、bcrypt |
| 数据库 | SQLite（`modernc.org/sqlite`，纯 Go 无需 CGO） |
| 前端 | Vue 3.5、TypeScript 5.7、Vite 6、Pinia、Vue Router 4 |
| 样式 | 手写 CSS，微信绿 `#07c160`，无 UI 框架 |
| 可观测性 | Prometheus、OpenTelemetry（OTLP/HTTP）、Jaeger |

## 快速开始

需要 Go 1.26+ 和 Node.js 20+；使用 Docker 时只需 Docker 24+ 与 Docker Compose v2。

**1. 启动后端**（默认端口 `8090`）

```bash
cd server
go run .
```

首次启动会自动创建 `data/wechat.db` 并写入演示数据。

**2. 启动前端**

```bash
cd web
npm install
npm run dev
```

打开 <http://localhost:5173>。Vite 已配置代理 `/api`、`/ws`、`/uploads` 到 `http://127.0.0.1:8090`。

## 演示账号

密码统一为 `123456`。

| 用户名 | 昵称 | 说明 |
| --- | --- | --- |
| `alice` | 林小敏 | 默认登录账号，已有单聊 + 群聊 + 未读消息 |
| `bob` | 陈杰 | 与 alice 有单聊记录 |
| `carol` | 苏晴 | 与 alice 有单聊记录 |
| `dave` | 王叔 | 群聊成员、朋友圈互动 |
| `emma` | 周雨桐 | 群聊成员、朋友圈互动 |

想体验实时消息，可同时用两个浏览器（或普通 + 无痕窗口）分别登录 `alice` 和 `bob`。

## 编译脚本

一键构建前端与后端：

```bash
./scripts/build.sh
```

等价于：

```bash
make build
```

构建产物：

- 前端：`web/dist`
- 后端：`server/mini-wechat`（静态链接，数据库驱动无需 CGO）

脚本支持按需跳过：

```bash
./scripts/build.sh --skip-web      # 只构建后端
./scripts/build.sh --skip-server   # 只构建前端
```

常用 Make 目标：

```bash
make help          # 查看全部命令
make install       # 安装前端依赖
make build         # 构建前后端
make run           # 构建并启动生产服务
make clean         # 清理构建产物
```

## 生产运行

```bash
make run
```

或手动启动：

```bash
./scripts/build.sh
cd server
./mini-wechat
```

`web/dist` 存在时，Go 服务会直接托管前端静态文件。访问 <http://localhost:8090> 即可，无需 Vite。

## Docker 运行

项目使用单容器部署：前端在镜像内完成构建，Go 服务同时提供 API、WebSocket、上传文件和静态页面。Compose 还会启动 Prometheus 与 Jaeger。

```bash
cp .env.example .env        # 按需修改 JWT_SECRET 和端口
docker compose up -d --build
```

打开 <http://localhost:8090>。可观测性入口：

- Prometheus：<http://localhost:9090>
- Jaeger：<http://localhost:16686>

常用命令：

```bash
docker compose ps                  # 查看状态与健康检查
docker compose logs -f --tail=100  # 查看日志
docker compose restart             # 重启
docker compose down                # 停止并删除容器
```

说明：

- SQLite 数据库和上传文件保存在命名卷 `mini-wechat-data` 的 `/data` 目录中，重启或重新创建容器不会丢失。
- Prometheus 数据保存在命名卷 `mini-wechat-prometheus-data`；Jaeger 数据只保存在内存中，重启后清空。
- 首次启动会把内置的示例图片复制到上传目录，保证种子朋友圈图片可访问。
- 容器以内置非 root 用户 `app` 运行，并配置了 `/api/health` 健康检查。
- 修改 `.env` 中的 `APP_PORT` 可更换宿主机端口，容器内部固定为 `8090`。
- 网络受限时可使用国内镜像源构建：

```bash
NPM_REGISTRY=https://registry.npmmirror.com \
GOPROXY=https://goproxy.cn,direct \
GOSUMDB=sum.golang.google.cn \
docker compose build
```

Prometheus / Jaeger 镜像无法从 Docker Hub 拉取时，可在 `.env` 中设置 `PROMETHEUS_IMAGE`、`JAEGER_IMAGE` 指向可达的镜像代理。

如需彻底重置 Docker 数据：

```bash
docker compose down -v
docker compose up -d --build
```

## 可观测性

### Prometheus 指标

Go 服务在 `/metrics` 暴露 Prometheus 格式指标，Compose 中的 Prometheus 每 15 秒抓取一次，配置见 `deploy/prometheus/prometheus.yml`。

| 指标 | 类型 | 说明 |
| --- | --- | --- |
| `mini_wechat_http_requests_total` | Counter | 按 `method`、`route`、`status` 统计请求数 |
| `mini_wechat_http_request_duration_seconds` | Histogram | 按 `method`、`route` 统计请求耗时 |
| `mini_wechat_http_requests_in_flight` | Gauge | 当前正在处理的请求数 |
| `mini_wechat_ws_connections_active` | Gauge | 当前 WebSocket 连接数 |
| `mini_wechat_ws_connections_total` | Counter | 累计接受的 WebSocket 连接数 |
| `mini_wechat_ws_events_sent_total` | Counter | 按事件类型统计已投递的 WS 事件 |
| `mini_wechat_ws_events_dropped_total` | Counter | 客户端发送缓冲满时丢弃的事件数 |
| `mini_wechat_chat_messages_created_total` | Counter | 按消息类型统计已保存的消息 |
| `mini_wechat_build_info` | Gauge | 版本与 Go 版本 |

同时包含 Go runtime 与进程指标（`go_*`、`process_*`）。生产环境建议只在内网暴露 `/metrics`，或通过反向代理限制访问。

Prometheus 查询示例：

```promql
sum(rate(mini_wechat_http_requests_total[5m])) by (route)
histogram_quantile(0.95, sum(rate(mini_wechat_http_request_duration_seconds_bucket[5m])) by (le, route))
mini_wechat_ws_connections_active
rate(mini_wechat_chat_messages_created_total[5m])
```

### 分布式链路

- 服务通过 OTLP/HTTP 把 span 发送到 Jaeger（`OTEL_EXPORTER_OTLP_ENDPOINT`，Compose 内为 `http://jaeger:4318`）。
- 支持 W3C `traceparent` / `tracestate` 透传；每个响应都会带 `X-Trace-ID` 响应头，可直接用它到 Jaeger 搜索。
- 每个 HTTP 请求生成服务端 span，记录路由、状态码、客户端地址与已登录用户 ID。
- `/ws` 升级后生成覆盖整个连接生命周期的 `ws.connect` span；客户端发送 typing 时生成 `ws.typing` 子 span。
- `/metrics`、`/api/health` 与静态资源不生成 span，避免噪声。
- 未设置 `OTEL_EXPORTER_OTLP_ENDPOINT` 时自动关闭导出，业务代码无需改动。

本地调试时先启动 Jaeger，再以默认 100% 采样运行服务（`PORT=8091` 避免与已运行的容器冲突）：

```bash
docker compose up -d jaeger
cd server
PORT=8091 OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run .
```

验证：

```bash
# 健康检查被排除在追踪之外，不会返回 X-Trace-ID
curl -si http://localhost:8091/api/health -o /dev/null

# 登录请求会返回 trace ID
curl -si -X POST http://localhost:8091/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"123456"}' | grep -i x-trace-id
```

把输出的 trace ID 粘贴到 Jaeger UI 的 Trace ID 搜索框，或直接访问 `http://localhost:16686/trace/<trace-id>`。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8090` | 后端监听端口 |
| `DB_PATH` | `data/wechat.db` | SQLite 文件路径（相对启动目录） |
| `UPLOADS_DIR` | `uploads` | 上传文件目录 |
| `JWT_SECRET` | `mini-wechat-dev-secret-change-me` | JWT 签名密钥，部署时务必修改 |
| `WEB_DIST` | `../web/dist` | 前端构建产物目录，存在时由 Go 托管 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 空 | OTLP/HTTP 上报地址；为空时关闭链路导出 |
| `OTEL_SERVICE_NAME` | `mini-wechat-server` | Jaeger 中显示的服务名 |
| `OTEL_TRACES_SAMPLER_ARG` | `1.0` | 链路采样比例（0-1） |
| `APP_PORT` | `8090` | Docker Compose 映射到宿主机的应用端口 |
| `PROMETHEUS_PORT` | `9090` | Prometheus 宿主机端口 |
| `JAEGER_UI_PORT` | `16686` | Jaeger UI 宿主机端口 |
| `JAEGER_OTLP_HTTP_PORT` | `4318` | Jaeger OTLP/HTTP 宿主机端口 |

Docker 构建参数（通过 `.env` 或命令行传入）：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `NPM_REGISTRY` | `https://registry.npmjs.org` | 前端依赖源 |
| `GOPROXY` | `https://proxy.golang.org,direct` | Go 模块代理 |
| `GOSUMDB` | `sum.golang.org` | Go 校验数据库 |

示例：

```bash
cd server
PORT=9000 JWT_SECRET=my-secret go run .
```

## 重置演示数据

停止后端后删除数据库和测试上传文件，再启动即可重新生成种子数据：

```bash
rm -f server/data/wechat.db server/data/wechat.db-shm server/data/wechat.db-wal
rm -f server/uploads/*.jpg
cd server && go run .
```

`server/uploads/seed/` 内的示例照片不会被删除，种子数据会重新引用它们。

Docker 环境重置：

```bash
docker compose down -v
docker compose up -d --build
```

## 目录结构

```text
mini-wechat/
├── Dockerfile                   # 前端 + 后端多阶段构建
├── docker-compose.yml           # 应用 + Prometheus + Jaeger 编排
├── Makefile                     # 常用构建/运行/Docker 命令
├── .env.example                 # Compose 环境变量示例
├── deploy/
│   └── prometheus/prometheus.yml # Prometheus 抓取配置
├── scripts/
│   ├── build.sh                 # 本地编译脚本
│   └── docker-entrypoint.sh     # 容器启动时注入种子图片
├── server/
│   ├── main.go                  # 服务入口、静态文件托管、优雅关闭
│   ├── internal/
│   │   ├── api/                 # 路由、鉴权、聊天、朋友圈、上传
│   │   ├── auth/                # bcrypt 与 JWT
│   │   ├── models/              # 数据模型
│   │   ├── observability/       # Prometheus 指标与 OpenTelemetry 链路
│   │   ├── realtime/            # WebSocket Hub
│   │   └── store/               # SQLite 存储与种子数据
│   ├── data/                    # 运行时生成数据库
│   └── uploads/seed/            # 内置朋友圈示例图片
└── web/
    ├── src/
    │   ├── api/                 # HTTP / WebSocket 封装
    │   ├── components/          # 导航、会话列表、聊天窗口等
    │   ├── stores/              # Pinia：鉴权、聊天、提示
    │   ├── views/               # 登录、主页、朋友圈、个人页
    │   ├── styles/              # 全局样式
    │   └── types.ts
    └── vite.config.ts           # 端口与代理配置
```

## 主要接口

公开接口：

- `GET /api/health`
- `GET /metrics`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /ws?token=<JWT>`

需要 `Authorization: Bearer <token>`：

- `GET /api/me`、`PATCH /api/me`
- `GET /api/users`
- `GET /api/conversations`
- `POST /api/conversations/direct`
- `POST /api/conversations/group`
- `GET /api/conversations/:id/messages`
- `POST /api/conversations/:id/messages`
- `POST /api/conversations/:id/read`
- `GET /api/moments`、`POST /api/moments`
- `POST /api/moments/:id/like`、`POST /api/moments/:id/comments`
- `POST /api/upload`

WebSocket 事件类型：`ready`、`message`、`typing`。

## 说明

这是一个本地演示项目：JWT 密钥、演示账号和 SQLite 单连接配置都面向单机运行。若要部署到公网，请修改 `JWT_SECRET`、配置 HTTPS/WSS、限制上传类型与大小，并改用更合适的数据库与连接池配置。
