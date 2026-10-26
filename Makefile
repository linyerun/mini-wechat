SHELL := /bin/sh

.PHONY: help install build build-web build-server run clean docker-build docker-up docker-down docker-logs

help:
	@printf '%s\n' \
		'Mini WeChat 常用命令：' \
		'  make install       安装前端依赖' \
		'  make build         构建前端与后端' \
		'  make build-web     仅构建前端' \
		'  make build-server  仅构建后端' \
		'  make run           构建并启动生产版服务（8090）' \
		'  make clean         删除前端产物与后端二进制' \
		'  make docker-build  构建 Docker 镜像' \
		'  make docker-up     启动应用、Prometheus、Jaeger' \
		'  make docker-down   停止 Docker Compose 服务' \
		'  make docker-logs   查看容器日志'

install:
	cd web && npm ci

build:
	./scripts/build.sh

build-web:
	./scripts/build.sh --skip-server

build-server:
	./scripts/build.sh --skip-web

run: build
	cd server && ./mini-wechat

clean:
	rm -rf web/dist server/mini-wechat

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f --tail=100
