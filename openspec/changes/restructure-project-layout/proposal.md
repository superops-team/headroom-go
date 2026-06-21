# Spec: Headroom Go 项目结构规范化

**版本:** v0.5.0-spec
**日期:** 2026-06-21
**状态:** 待确认

---

## 1. 背景与动机

### 1.1 现状问题

当前 `headroom-go` 项目所有 `.go` 源文件（40+ 个）散落在根路径下，全部声明为 `package headroom`，存在以下问题：

| 问题 | 影响 |
|------|------|
| **根目录文件过多** | 40+ 个 `.go` 文件平铺，难以快速定位模块边界 |
| **职责边界模糊** | 压缩引擎、tokenizer、CCR、tag protector 等不同职责的代码混在一起 |
| **不符合 Go 惯例** | Go 社区标准：根包仅暴露公共 API，实现细节放 `internal/` |
| **测试文件混杂** | `_test.go` 与源码交织，`spec_*_e2e_test.go` 等 E2E 测试无独立目录 |
| **可维护性差** | 新人难以理解模块划分，新增功能不知道放哪里 |

### 1.2 目标

按照 Go 项目布局标准（`golang-project-layout`），将项目重构为清晰的模块化结构：

- **根包**：仅保留公共 API 类型和入口函数
- **`internal/`**：封装所有实现细节
- **`cmd/headroom/`**：CLI 入口（已正确）
- **`proxy/`**：HTTP 代理包（已正确）
- **零破坏性**：外部 `import "github.com/superops-team/headroom-go"` 的 API 完全不变

---

## 2. 目标目录结构

```
headroom-go/
├── cmd/
│   └── headroom/
│       └── main.go                    # CLI 入口（不变）
├── proxy/
│   ├── proxy.go                       # HTTP 代理（不变）
│   └── proxy_test.go                  # 代理测试（不变）
│
├── internal/                          # 内部实现（新增）
│   ├── engine/                        # 压缩引擎核心
│   │   ├── engine.go                  # CompressionEngine 结构体
│   │   ├── pipeline.go               # Pipeline 路径
│   │   ├── policy.go                 # 策略引擎
│   │   └── config.go                 # 引擎配置
│   │
│   ├── compressors/                   # 内容类型压缩器
│   │   ├── compressors.go            # Compressor 接口 + 注册表
│   │   ├── json.go                   # SmartCrusher（JSON）
│   │   ├── code.go                   # 代码压缩
│   │   ├── text.go                   # 文本压缩
│   │   └── transforms.go            # 专用变换（diff/log/search/tabular/spreadsheet/html）
│   │
│   ├── router/                        # 内容类型检测
│   │   ├── router.go                 # ContentRouter
│   │   └── contentkind.go           # ContentKind 定义
│   │
│   ├── tokenizer/                     # Tokenizer 实现
│   │   ├── tokenizer.go             # Tokenizer 接口 + fallback
│   │   ├── tiktoken.go              # tiktoken 后端
│   │   └── huggingface.go           # HuggingFace 后端
│   │
│   ├── ccr/                           # 可逆压缩
│   │   ├── ccr.go                    # CCR 核心逻辑
│   │   └── store.go                  # CCR 存储
│   │
│   ├── tagprotector/                  # Tag 保护
│   │   └── protector.go
│   │
│   └── cachealigner/                  # KV Cache 对齐
│       └── aligner.go
│
├── headroom.go                        # 公共 API：Message, Options, Result, Compress()
├── version.go                         # 版本常量
├── observability.go                   # 公共接口：Observer, CompressionStep, Warning
│
├── testdata/                          # 测试数据（已有）
├── go.mod                             # 模块定义（不变）
├── Makefile                           # 构建自动化（新增）
├── .gitignore                         # 已有
├── .golangci.yml                      # Linter 配置（新增）
├── README.md
├── CHANGELOG.md
├── LICENSE
└── install.sh
```

---

## 3. 模块职责划分

### 3.1 根包（`package headroom`）— 公共 API

| 文件 | 导出内容 | 说明 |
|------|---------|------|
| `headroom.go` | `Message`, `Options`, `Result`, `Compress()`, `CompressString()`, `DefaultOptions()` | 用户入口 |
| `version.go` | `Version`, `PrefixVersion`, `CCRIDVersion` | 版本信息 |
| `observability.go` | `Observer`, `CompressionStep`, `Warning` | 可观测性接口 |

> **原则：根包只放用户直接使用的类型和函数，实现细节全部下沉 `internal/`。**

### 3.2 `internal/engine/` — 压缩引擎

| 文件 | 职责 |
|------|------|
| `engine.go` | `CompressionEngine` 结构体，协调压缩流程 |
| `pipeline.go` | Pipeline 模式压缩路径（policy-driven） |
| `policy.go` | 策略引擎：token budget、query scoring |
| `config.go` | 引擎配置加载与默认值 |

### 3.3 `internal/compressors/` — 压缩器

| 文件 | 职责 |
|------|------|
| `compressors.go` | `Compressor` 接口定义 + `CompressorRegistry` |
| `json.go` | SmartCrusher：JSON 压缩（去 null、折叠数组、截断浮点） |
| `code.go` | 代码压缩（去注释、折叠长函数、保留错误处理） |
| `text.go` | 文本压缩（去重行、移除停用词、折叠段落） |
| `transforms.go` | Diff/Log/Search/Tabular/Spreadsheet/HTML 专用变换 |

### 3.4 `internal/router/` — 内容检测

| 文件 | 职责 |
|------|------|
| `router.go` | `ContentRouter`：自动检测内容类型 |
| `contentkind.go` | `ContentKind` 枚举定义 |

### 3.5 `internal/tokenizer/` — Token 计数

| 文件 | 职责 |
|------|------|
| `tokenizer.go` | `Tokenizer` 接口 + fallback 实现 |
| `tiktoken.go` | tiktoken 兼容后端 |
| `huggingface.go` | HuggingFace tokenizer 后端 |

### 3.6 `internal/ccr/` — 可逆压缩

| 文件 | 职责 |
|------|------|
| `ccr.go` | CCR 核心：压缩缓存、检索、TTL 管理 |
| `store.go` | 存储后端抽象 |

### 3.7 `internal/tagprotector/` — Tag 保护

| 文件 | 职责 |
|------|------|
| `protector.go` | 保护 `<thinking>`, `<tool_call>` 等 XML 标签不被压缩破坏 |

### 3.8 `internal/cachealigner/` — 缓存对齐

| 文件 | 职责 |
|------|------|
| `aligner.go` | 前缀对齐，提升 Provider KV Cache 命中率 |

---

## 4. 公共 API 兼容性保证

### 4.1 不变的外部 API

```go
// 用户代码完全不变
import headroom "github.com/superops-team/headroom-go"

result, _ := headroom.Compress(messages, headroom.Options{...})
```

### 4.2 类型别名桥接

根包通过类型别名暴露内部类型，保持 API 兼容：

```go
// headroom.go
import "github.com/superops-team/headroom-go/internal/engine"

type Message = engine.Message  // 类型别名，完全兼容
```

> **注意：** 如果当前这些类型直接定义在根包中且被外部引用，重构后通过类型别名保持二进制兼容。

### 4.3 不变的文件

| 文件 | 说明 |
|------|------|
| `cmd/headroom/main.go` | CLI 入口，import 路径不变 |
| `proxy/proxy.go` | 代理包，import 路径不变 |
| `go.mod` | 模块路径不变 |
| `testdata/` | 测试数据不变 |

---

## 5. 实施步骤

### Phase 1: 基础设施准备

| 步骤 | 内容 | 验证 |
|------|------|------|
| 1.1 | 创建 `internal/` 子目录结构 | `find internal/ -type d` |
| 1.2 | 创建 `Makefile`（build/test/bench/lint） | `make help` |
| 1.3 | 创建 `.golangci.yml` | `golangci-lint run` |

### Phase 2: 代码迁移（按依赖顺序）

| 步骤 | 内容 | 说明 |
|------|------|------|
| 2.1 | 迁移 `internal/tokenizer/` | 无内部依赖，最先迁移 |
| 2.2 | 迁移 `internal/router/` | 仅依赖 ContentKind |
| 2.3 | 迁移 `internal/ccr/` | 独立模块 |
| 2.4 | 迁移 `internal/tagprotector/` | 独立模块 |
| 2.5 | 迁移 `internal/cachealigner/` | 独立模块 |
| 2.6 | 迁移 `internal/compressors/` | 依赖 router |
| 2.7 | 迁移 `internal/engine/` | 依赖以上所有 |
| 2.8 | 重构根包 | 类型别名 + 精简 |

### Phase 3: 测试迁移

| 步骤 | 内容 |
|------|------|
| 3.1 | `_test.go` 文件随源码迁移到对应 `internal/` 子包 |
| 3.2 | E2E 测试（`spec_*_e2e_test.go`）保留在根目录或移入 `internal/engine/` |
| 3.3 | 确保 `go test -race ./...` 全部通过 |

### Phase 4: 质量保障

| 步骤 | 内容 |
|------|------|
| 4.1 | `golangci-lint run ./...` 零错误 |
| 4.2 | `go test -race -count=1 ./...` 全部通过 |
| 4.3 | `go test -coverprofile=coverage.out ./...` 覆盖率 ≥ 92% |
| 4.4 | `go build -o headroom ./cmd/headroom` 构建成功 |
| 4.5 | 功能验证：`echo '{"test": true}' | ./headroom compress --stats` |

---

## 6. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **类型别名导致 API 不兼容** | 外部用户编译失败 | 每个 Phase 完成后立即 `go build` + `go test` |
| **循环依赖** | 编译失败 | 严格按依赖顺序迁移，先底层后上层 |
| **测试覆盖下降** | 质量退化 | 迁移过程中不修改测试逻辑，仅调整 package 和 import |
| **Git 历史丢失** | 难以追溯 | 使用 `git mv` 而非新建文件 |

---

## 7. 验收标准

- [ ] 根目录 `.go` 文件从 40+ 减少到 ≤ 5 个
- [ ] `internal/` 下所有子包职责清晰、单一
- [ ] `go build ./...` 编译成功
- [ ] `go test -race -count=1 ./...` 138 个测试全部通过
- [ ] `golangci-lint run ./...` 零新增问题
- [ ] 覆盖率 ≥ 92%（不下降）
- [ ] `headroom compress --stats` CLI 功能正常
- [ ] 外部 import API 完全兼容
- [ ] CHANGELOG 更新

---

## 8. 时间估算

| Phase | 预估时间 |
|-------|---------|
| Phase 1: 基础设施 | 15 分钟 |
| Phase 2: 代码迁移 | 45 分钟 |
| Phase 3: 测试迁移 | 20 分钟 |
| Phase 4: 质量保障 | 15 分钟 |
| **总计** | **~1.5 小时** |

---

*本 spec 待教主确认后进入执行阶段。*
