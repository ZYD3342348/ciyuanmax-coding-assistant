# CiYuanMax 编程助手 CLI

这是 CiYuanMax 的跨平台命令行客户端，面向 macOS 和 Windows。它负责把 Claude Code、Codex CLI 与 CiYuanMax 网关连接起来：浏览器授权、创建凭证、写入本地配置、查看状态、诊断和轮换凭证。

当前分支 `cli-v2` 是纯 Go 实现。原 Electron 客户端保留在 `main` 分支，方便追溯历史；两者不共用构建入口。

## 使用

```bash
ciyuanmax login
ciyuanmax configure --mode stable --tools claude,codex
ciyuanmax status
ciyuanmax doctor
ciyuanmax keys
ciyuanmax rotate --mode stable --tools claude,codex
ciyuanmax revoke --id 123
ciyuanmax logout
```

直接运行 `ciyuanmax` 会打开交互菜单。首次配置会：

1. 在终端显示一次性验证码并打开浏览器。
2. 用户核对验证码后，在 CiYuanMax 网页明确点击授权。
3. 轮询设备登录结果并保存本机 CLI 会话。
4. 为 Claude Code 和 Codex 创建或复用当前设备凭证。
5. 写入本地配置；覆盖前自动生成带时间戳的备份。

当前写入的配置：

| 工具 | 文件 | 连接方式 |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、模型 |
| Codex CLI | `~/.codex/config.toml` | `model_providers.ciyuanmax`、Responses API、模型 |

本机 CLI 会话默认保存到：

- macOS：`~/Library/Application Support/CiYuanMax/session.json`
- Windows：`%AppData%\\ciyuanmax\\session.json`

文件权限为用户私有（`0600`）。不要把该文件、配置备份或 API Key 提交到 Git。

## 构建

需要 Go 1.25 或更新版本：

```bash
go test ./...
go vet ./...
make release VERSION=0.1.0
```

`make release` 生成 Mac Intel、Mac Apple Silicon 和 Windows x64 构建，并生成 SHA256 清单。

## 发布边界

- 先发布测试构建，不替换旧客户端下载目录。
- macOS 必须使用 Developer ID 签名并完成公证。
- Windows 必须使用 Authenticode 签名。
- 服务端 `/api/cli` 变更必须先向后兼容旧二进制，再发布新 CLI。
