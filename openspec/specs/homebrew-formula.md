# Spec: Homebrew Formula

**版本:** v0.8.0-brew
**日期:** 2026-06-22
**优先级:** P2
**状态:** 待确认

---

## 1. 背景

Homebrew 是 macOS 用户最主流的包管理器。提供 Homebrew formula 可以让 macOS 用户一键安装 headroom-go。

---

## 2. 目标

- 创建 Homebrew formula
- 推送到 homebrew-core 或自建 tap
- `brew install headroom` 一键安装

---

## 3. 方案选择

| 方案 | 优点 | 缺点 |
|------|------|------|
| homebrew-core | 官方源，用户最多 | 审核严格，需满足人气/star 等条件 |
| **自建 tap** | 完全自主控制 | 需要 `brew tap superops-team/headroom` |

推荐先自建 tap，达到一定人气后申请 homebrew-core。

### 3.1 Formula

```ruby
# headroom.rb
class Headroom < Formula
  desc "Intelligent context compression for AI agents"
  homepage "https://github.com/superops-team/headroom-go"
  url "https://github.com/superops-team/headroom-go/releases/download/v0.7.0/headroom-darwin-amd64"
  sha256 "..."

  def install
    bin.install "headroom-darwin-amd64" => "headroom"
  end

  test do
    assert_match "headroom-go", shell_output("#{bin}/headroom version")
  end
end
```

### 3.2 使用方式

```bash
# 添加 tap
brew tap superops-team/headroom

# 安装
brew install headroom

# 验证
headroom version
```

---

## 4. 自动化

在 Release workflow 中自动更新 formula 的 URL 和 SHA256：

```yaml
# .github/workflows/release.yml 中增加
- name: Update Homebrew formula
  run: |
    SHA256=$(sha256sum headroom-darwin-amd64 | cut -d' ' -f1)
    sed -i "s|url \".*\"|url \"${{ github.event.release.assets[0].browser_download_url }}\"|" homebrew/headroom.rb
    sed -i "s|sha256 \".*\"|sha256 \"$SHA256\"|" homebrew/headroom.rb
```

---

## 5. 文件变更

| 操作 | 文件 | 说明 |
|------|------|------|
| **新建** | `homebrew/headroom.rb` | Homebrew formula |
| **新建** | `homebrew/README.md` | Tap 说明 |
| **修改** | `.github/workflows/release.yml` | 自动更新 formula |

---

## 6. 验收标准

- [ ] `brew tap superops-team/headroom` 成功
- [ ] `brew install headroom` 成功
- [ ] `headroom version` 输出正确
- [ ] Release 时 formula 自动更新

---

## 7. 时间估算

| 阶段 | 预估 |
|------|------|
| formula 编写 | 0.5h |
| tap 仓库创建 | 0.25h |
| CI 自动更新 | 0.5h |
| 验证 | 0.25h |
| **总计** | **~1.5h** |
