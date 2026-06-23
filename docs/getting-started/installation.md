---
title: Installation
---

# Installation

## One-Line Install

```bash
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash
```

Install a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/superops-team/headroom-go/main/install.sh | bash -s -- v0.7.0
```

## Go Install

```bash
go install github.com/superops-team/headroom-go/cmd/headroom@latest
```

## Docker

```bash
docker pull ghcr.io/superops-team/headroom-go:latest
docker run -p 18787:18787 ghcr.io/superops-team/headroom-go:latest
```

## From Source

```bash
git clone https://github.com/superops-team/headroom-go.git
cd headroom-go
go build -o headroom ./cmd/headroom/
```

## Verify

```bash
headroom version
```
