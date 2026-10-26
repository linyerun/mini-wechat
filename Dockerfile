ARG NPM_REGISTRY=https://registry.npmjs.org
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

FROM node:24-alpine AS web-build
ARG NPM_REGISTRY
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm config set registry "$NPM_REGISTRY" && npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS server-build
ARG GOPROXY
ARG GOSUMDB
ENV GOPROXY=$GOPROXY \
    GOSUMDB=$GOSUMDB
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mini-wechat .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=server-build /out/mini-wechat /app/mini-wechat
COPY --from=web-build /src/web/dist /app/web/dist
COPY server/uploads/seed /app/seed-uploads
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /data/uploads \
    && chmod +x /app/docker-entrypoint.sh \
    && chown -R app:app /app /data

ENV PORT=8090 \
    DB_PATH=/data/wechat.db \
    UPLOADS_DIR=/data/uploads \
    WEB_DIST=/app/web/dist

USER app
EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8090/api/health || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/mini-wechat"]
