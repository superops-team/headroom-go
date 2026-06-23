# Spec: 文档站 + Discord 社区

**版本:** v0.7.0-community
**日期:** 2026-06-22
**优先级:** P1
**状态:** 待确认

---

## 1. 背景

Headroom (Python) 有独立文档站 (headroom-docs.vercel.app)、Discord 社区、Star History 图表、Trendshift 徽章等完善的社区运营。headroom-go 目前只有 README.md，缺少独立文档站和社区渠道。

---

## 2. 目标

### 2.1 文档站

- 使用 GitHub Pages（免费，零运维）
- 内容：安装指南、快速开始、API 参考、集成指南、FAQ
- 支持暗色模式
- 响应式设计

### 2.2 Discord 社区

- 创建 Discord 服务器
- README 显著展示 Discord 邀请链接
- 频道结构：`#announcements`、`#general`、`#help`、`#showcase`

### 2.3 README 增强

- 添加 Star History 图表
- 添加 Discord 徽章
- 添加 Go Report Card 徽章
- 添加 CI 状态徽章

---

## 3. 文档站方案

### 3.1 技术选型

使用 **Mintlify** 或 **VitePress** 或直接 **GitHub Pages + Markdown**：

| 方案 | 优点 | 缺点 |
|------|------|------|
| GitHub Pages + Jekyll | 零配置，免费 | 样式有限 |
| VitePress | 现代化，暗色模式 | 需要 Node.js |
| **Docusaurus** | 功能最全 | 较重 |
| **Starlight (Astro)** | 轻量，暗色模式内置 | 推荐 ✅ |

推荐 **Starlight**：基于 Astro，内置暗色模式、搜索、i18n，构建产物纯静态。

### 3.2 文档结构

```
docs/
├── index.md              # 首页
├── getting-started/
│   ├── installation.md   # 安装方式（go install / 脚本 / Docker）
│   └── quickstart.md     # 5 分钟快速开始
├── guide/
│   ├── compression.md    # 压缩模式详解
│   ├── proxy.md          # Proxy 模式
│   ├── mcp.md            # MCP Server
│   ├── wrap.md           # Wrap 命令
│   └── pipeline.md       # Pipeline 模式
├── integrations/
│   ├── go-openai.md
│   ├── langchaingo.md
│   ├── ollama.md
│   └── k8s.md
├── reference/
│   ├── cli.md            # CLI 参考
│   ├── api.md            # Go API 参考
│   └── config.md         # 配置参考
└── community/
    ├── contributing.md
    └── changelog.md
```

### 3.3 部署

```yaml
# .github/workflows/docs.yml
name: Deploy Docs
on:
  push:
    branches: [main]
    paths: ['docs/**']
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: withastro/action@v3
```

---

## 4. Discord 社区

### 4.1 频道结构

```
📢 announcements     # 版本发布、重要通知
💬 general           # 一般讨论
❓ help               # 使用问题
🎨 showcase           # 用户案例展示
🔧 development        # 开发讨论
```

### 4.2 README 徽章

```markdown
[![Discord](https://img.shields.io/discord/XXXXXXXXX?color=5865F2&logo=discord&logoColor=white)](https://discord.gg/xxxxx)
[![CI](https://github.com/superops-team/headroom-go/actions/workflows/ci.yml/badge.svg)](https://github.com/superops-team/headroom-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/superops-team/headroom-go)](https://goreportcard.com/report/github.com/superops-team/headroom-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/superops-team/headroom-go.svg)](https://pkg.go.dev/github.com/superops-team/headroom-go)
```

---

## 5. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `docs/` 目录 | 文档站源文件 |
| **新建** | `docs/astro.config.mjs` | Starlight 配置 |
| **新建** | `.github/workflows/docs.yml` | 文档部署 |
| **修改** | `README.md` | 添加徽章 + Discord 链接 |

---

## 6. 验收标准

- [ ] 文档站可访问（GitHub Pages 域名）
- [ ] 文档覆盖安装/快速开始/API/集成/CLI
- [ ] 暗色模式正常
- [ ] Discord 服务器可加入
- [ ] README 展示所有徽章
- [ ] Go Report Card A+ 评级

---

## 7. 时间估算

| 阶段 | 预估 |
|------|------|
| Starlight 初始化 + 首页 | 1h |
| 核心文档撰写 | 2h |
| 集成文档 | 1h |
| GitHub Pages 部署 | 0.5h |
| Discord 创建 + 配置 | 0.5h |
| README 徽章更新 | 0.25h |
| **总计** | **~5.25h** |
