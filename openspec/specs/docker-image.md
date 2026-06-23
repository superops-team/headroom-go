# Spec: Docker 镜像发布

**版本:** v0.7.0-docker
**日期:** 2026-06-22
**优先级:** P0
**状态:** 待确认

---

## 1. 背景

headroom-go 目前没有预构建 Docker 镜像，用户需自行编译。Headroom (Python) 已推送到 ghcr.io。Go 的单二进制特性让 Docker 镜像可以做到极致精简（<10MB）。

---

## 2. 目标

- 多阶段构建，最终镜像 <15MB
- 支持 linux/amd64 + linux/arm64
- 推送到 ghcr.io/superops-team/headroom-go
- Tag 策略：`v0.7.0` + `latest`

---

## 3. Dockerfile

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /headroom ./cmd/headroom/

# Stage 2: Runtime
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /headroom /headroom
EXPOSE 18787
ENTRYPOINT ["/headroom"]
CMD ["proxy", "--port=18787"]
```

### 3.1 镜像大小

| 层 | 大小 |
|----|------|
| scratch base | 0 MB |
| ca-certificates | ~0.2 MB |
| headroom binary (stripped) | ~8 MB |
| **总计** | **~8.2 MB** |

对比 Headroom (Python) Docker 镜像 ~200MB+。

---

## 4. 使用方式

```bash
# 拉取
docker pull ghcr.io/superops-team/headroom-go:latest

# 运行 proxy
docker run -p 18787:18787 ghcr.io/superops-team/headroom-go:latest

# 运行 compress
echo '{"key":"value"}' | docker run -i ghcr.io/superops-team/headroom-go:latest compress

# 自定义参数
docker run -p 18787:18787 ghcr.io/superops-team/headroom-go:latest \
  proxy --port=18787 --aggressiveness=0.7 --enable-pipeline
```

---

## 5. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `Dockerfile` | 多阶段构建 |
| **新建** | `.dockerignore` | 排除不必要文件 |
| **修改** | `.github/workflows/docker.yml` | 自动构建推送（见 ci-cd spec） |

### 5.1 .dockerignore

```
.git/
.hermes/
openspec/
testdata/
*.test
*.md
!README.md
```

---

## 6. 验收标准

- [ ] `docker build -t headroom-go .` 成功
- [ ] 镜像大小 <15MB
- [ ] `docker run headroom-go version` 输出正确版本
- [ ] `docker run -p 18787:18787 headroom-go proxy` 可访问 healthz
- [ ] `echo '{"a":1}' | docker run -i headroom-go compress` 正常输出
- [ ] ghcr.io 自动推送（Tag push 触发）

---

## 7. 时间估算

| 阶段 | 预估 |
|------|------|
| Dockerfile + .dockerignore | 0.25h |
| 本地验证 | 0.25h |
| CI 集成 | 见 ci-cd spec |
| **总计** | **~0.5h** |
