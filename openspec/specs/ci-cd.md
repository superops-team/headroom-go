# Spec: GitHub Actions CI/CD

**版本:** v0.7.0-ci
**日期:** 2026-06-22
**优先级:** P0
**状态:** 待确认

---

## 1. 背景

headroom-go 目前没有任何 CI/CD，构建、测试、发布全部手动操作。需要建立自动化流水线保障代码质量和发布效率。

---

## 2. 目标

建立完整的 GitHub Actions CI/CD 流水线：

| 触发条件 | 执行内容 |
|----------|---------|
| Push to main / PR | build + test + lint + security scan |
| Tag push (v*) | build + test + lint + release binaries + Docker push |
| Schedule (daily) | 全量测试 + 依赖检查 |

---

## 3. Workflow 设计

### 3.1 CI Workflow (`.github/workflows/ci.yml`)

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ['1.22', '1.23', '1.24']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: ${{ matrix.go }} }
      - run: go build ./...
      - run: go test -race -count=1 ./...
      - run: go vet ./...
      - run: go test -cover ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: gofmt -d . && test -z "$(gofmt -d .)"
      - uses: golangci/golangci-lint-action@v6
        with: { version: latest }

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: gitleaks/gitleaks-action@v2
```

### 3.2 Release Workflow (`.github/workflows/release.yml`)

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        arch: [amd64, arm64]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: |
          GOOS=${{ matrix.os == 'ubuntu-latest' && 'linux' || 'darwin' }}
          GOARCH=${{ matrix.arch }}
          go build -o headroom-$GOOS-$GOARCH ./cmd/headroom/
      - uses: actions/upload-artifact@v4
        with:
          name: headroom-${{ matrix.os }}-${{ matrix.arch }}
          path: headroom-*

  release:
    needs: build
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/download-artifact@v4
      - run: |
          for f in headroom-*/headroom-*; do
            cp "$f" "$(basename "$f")"
            sha256sum "$(basename "$f")" >> checksums.txt
          done
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            headroom-*
            checksums.txt
          generate_release_notes: true
```

### 3.3 Docker Workflow (`.github/workflows/docker.yml`)

```yaml
name: Docker
on:
  push:
    tags: ['v*']

jobs:
  docker:
    runs-on: ubuntu-latest
    permissions:
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ghcr.io/${{ github.repository }}
          tags: type=semver,pattern={{version}}
      - uses: docker/build-push-action@v6
        with:
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          platforms: linux/amd64,linux/arm64
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `.github/workflows/ci.yml` | CI 流水线 |
| **新建** | `.github/workflows/release.yml` | Release 流水线 |
| **新建** | `.github/workflows/docker.yml` | Docker 构建推送 |
| **新建** | `.golangci.yml` | golangci-lint 配置 |
| **新建** | `Dockerfile` | 多阶段构建 |

---

## 5. 验收标准

- [ ] PR 触发 CI：build + test + lint + security 全部通过
- [ ] Tag push 触发 Release：自动构建多平台二进制 + 发布
- [ ] Tag push 触发 Docker：自动推送到 ghcr.io
- [ ] 所有 workflow 在 GitHub Actions 上成功运行
- [ ] Release 页面包含 checksums.txt

---

## 6. 时间估算

| 阶段 | 预估 |
|------|------|
| ci.yml | 0.5h |
| release.yml | 0.5h |
| docker.yml + Dockerfile | 0.5h |
| .golangci.yml | 0.25h |
| 测试触发 | 0.25h |
| **总计** | **~2h** |
