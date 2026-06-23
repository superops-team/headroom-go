# headroom-go K8s Sidecar

将 headroom-go 作为 sidecar 部署，透明压缩 Pod 内所有 LLM API 请求。

## 架构

```
┌─────────────────────────────────────┐
│ Pod                                 │
│  ┌──────────┐    ┌──────────────┐   │
│  │ Your App │───▶│  headroom    │───▶ LLM API
│  │          │    │  :18787      │   │
│  │ localhost│    │  proxy       │   │
│  └──────────┘    └──────────────┘   │
└─────────────────────────────────────┘
```

## 部署

```bash
kubectl apply -f sidecar.yaml
```

## 配置

应用通过 `http://localhost:18787/v1` 访问 LLM API，headroom 自动压缩所有请求。
