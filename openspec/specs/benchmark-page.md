# Spec: 性能 Benchmark 页

**版本:** v0.8.0-benchmark
**日期:** 2026-06-22
**优先级:** P2
**状态:** 待确认

---

## 1. 背景

headroom-go 的核心卖点是"零依赖、高性能"。需要可量化的 benchmark 数据来支撑这一宣称，并作为社区推广素材。

---

## 2. 目标

- 提供自动化 benchmark 工具
- 生成可视化 benchmark 报告页
- 与 Headroom (Python) 做性能对比
- 部署到 GitHub Pages

---

## 3. Benchmark 维度

### 3.1 压缩性能

| 指标 | 说明 |
|------|------|
| 吞吐量 | MB/s，按内容类型分（JSON/Code/Text/Log/Diff） |
| Token 节省率 | 压缩前后 token 比 |
| 语义保留率 | 压缩后关键信息保留比例 |

### 3.2 运行时性能

| 指标 | 说明 |
|------|------|
| 启动时间 | 冷启动到 ready |
| 内存占用 | idle / 1k QPS / 10k QPS |
| 二进制大小 | stripped binary size |
| Docker 镜像大小 | 与 Python 版对比 |

### 3.3 对比基准

| 对比项 | 版本 |
|--------|------|
| headroom-go | latest |
| headroom (Python) | latest |
| 原始（不压缩） | baseline |

---

## 4. 实现方案

### 4.1 Benchmark CLI

```bash
# 运行 benchmark
headroom benchmark --output report.json

# 生成 HTML 报告
headroom benchmark --report html --output benchmarks/
```

### 4.2 报告页

```
benchmarks/
├── index.html           # 交互式报告页
├── data/
│   ├── compression.json # 压缩性能数据
│   ├── runtime.json     # 运行时性能数据
│   └── comparison.json  # 对比数据
└── charts/
    ├── throughput.js    # Chart.js 图表
    └── memory.js
```

### 4.3 报告页内容

- 压缩吞吐量对比图（柱状图，按内容类型）
- Token 节省率对比（雷达图）
- 内存占用对比（折线图，按 QPS）
- 启动时间对比（单值指标）
- 二进制/镜像大小对比（单值指标）
- 原始数据表格

---

## 5. 自动化

CI 中每次 Release 自动运行 benchmark 并更新报告页：

```yaml
# .github/workflows/benchmark.yml
name: Benchmark
on:
  push:
    tags: ['v*']

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: go run ./cmd/headroom benchmark --report html --output benchmarks/
      - uses: peaceiris/actions-gh-pages@v4
        with:
          publish_dir: ./benchmarks
          destination_dir: benchmarks
```

---

## 6. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `cmd/headroom/benchmark.go` | Benchmark CLI 子命令 |
| **新建** | `internal/benchmark/runner.go` | Benchmark 运行器 |
| **新建** | `internal/benchmark/report.go` | 报告生成 |
| **新建** | `benchmarks/index.html` | 报告页模板 |
| **新建** | `.github/workflows/benchmark.yml` | 自动运行 |

---

## 7. 验收标准

- [ ] `headroom benchmark` 输出 JSON 报告
- [ ] `headroom benchmark --report html` 生成 HTML 报告页
- [ ] 报告页包含吞吐量/内存/启动时间对比
- [ ] 报告页可通过 GitHub Pages 访问
- [ ] Release 时自动更新报告

---

## 8. 时间估算

| 阶段 | 预估 |
|------|------|
| benchmark runner | 1.5h |
| 报告生成 | 1h |
| HTML 报告页 | 1.5h |
| CI 集成 | 0.5h |
| **总计** | **~4.5h** |
