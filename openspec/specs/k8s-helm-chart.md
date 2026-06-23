# Spec: K8s Helm Chart

**版本:** v0.7.0-k8s
**日期:** 2026-06-22
**优先级:** P1
**状态:** 待确认

---

## 1. 背景

headroom-go 的单二进制 + 零依赖特性使其天然适合 K8s sidecar 模式部署。提供 Helm Chart 可以降低企业级采用门槛。

---

## 2. 目标

- 提供标准 Helm Chart
- 支持 Deployment + Sidecar 两种模式
- 支持通过 values.yaml 配置所有参数
- 推送到 GitHub Pages Helm Repo

---

## 3. Chart 结构

```
charts/headroom/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment.yaml      # 独立 Deployment 模式
│   ├── service.yaml
│   ├── configmap.yaml
│   └── NOTES.txt
└── README.md
```

### 3.1 values.yaml

```yaml
replicaCount: 1

image:
  repository: ghcr.io/superops-team/headroom-go
  tag: latest
  pullPolicy: IfNotPresent

mode: standalone  # standalone | sidecar

# standalone 模式配置
service:
  type: ClusterIP
  port: 18787

# headroom 配置
config:
  aggressiveness: 0.5
  reversible: true
  alignPrefix: false
  tokenLimit: 0
  enablePipeline: false
  upstream: ""  # 上游 LLM API（空则从环境变量读取）

# sidecar 模式配置（当 mode=sidecar 时生效）
sidecar:
  targetPort: 8080  # 主容器端口

resources:
  limits:
    memory: 64Mi
    cpu: 100m
  requests:
    memory: 16Mi
    cpu: 10m

# 环境变量（用于上游 API key 等）
env: []
# - name: HEADROOM_API_KEY
#   valueFrom:
#     secretKeyRef:
#       name: llm-secret
#       key: api-key
```

### 3.2 使用方式

```bash
# 安装
helm repo add headroom https://superops-team.github.io/headroom-go
helm install headroom headroom/headroom

# 自定义配置
helm install headroom headroom/headroom \
  --set config.aggressiveness=0.7 \
  --set config.enablePipeline=true

# Sidecar 模式
helm install headroom headroom/headroom \
  --set mode=sidecar \
  --set sidecar.targetPort=8080
```

---

## 4. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `charts/headroom/Chart.yaml` | Chart 元数据 |
| **新建** | `charts/headroom/values.yaml` | 默认配置 |
| **新建** | `charts/headroom/templates/_helpers.tpl` | 模板辅助 |
| **新建** | `charts/headroom/templates/deployment.yaml` | Deployment |
| **新建** | `charts/headroom/templates/service.yaml` | Service |
| **新建** | `charts/headroom/templates/configmap.yaml` | ConfigMap |
| **新建** | `charts/headroom/templates/NOTES.txt` | 安装提示 |
| **新建** | `charts/headroom/README.md` | Chart 文档 |
| **新建** | `.github/workflows/helm.yml` | Helm Chart 发布 |

---

## 5. 验收标准

- [ ] `helm install headroom ./charts/headroom` 成功
- [ ] `kubectl port-forward svc/headroom 18787:18787` 可访问 healthz
- [ ] standalone 模式正常运行
- [ ] sidecar 模式可注入到 Pod
- [ ] `helm lint charts/headroom` 无警告
- [ ] Helm Repo 可通过 `helm repo add` 添加

---

## 6. 时间估算

| 阶段 | 预估 |
|------|------|
| Chart 模板编写 | 1.5h |
| values.yaml 设计 | 0.5h |
| 本地验证 | 0.5h |
| Helm Repo 发布 | 0.5h |
| **总计** | **~3h** |
