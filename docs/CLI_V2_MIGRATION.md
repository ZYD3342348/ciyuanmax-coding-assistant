# CLI v2 迁移说明

`main` 是 2026-05 的 Electron 客户端，目标为 Windows x64。线上 2026-07 的 `ciyuanmax-*` 二进制已经转为轻量跨平台 CLI，并使用 ElucidRouter `/api/cli`。`cli-v2` 保留原产品历史，但移除 Electron 构建链，避免继续维护两套客户端运行时。

## 迁移顺序

1. 在测试账号执行 `ciyuanmax login`。
2. 执行 `ciyuanmax configure --mode stable --tools claude,codex`。
3. 使用 `ciyuanmax doctor` 检查工具路径、网关和配置文件。
4. 启动新的 Claude Code/Codex 会话，验证一次工具调用。
5. 轮换或吊销旧客户端凭证，确认旧设备不再使用。

旧配置不会被删除。CLI v2 在覆盖前生成 `.ciyuanmax-backup-*` 文件，出问题时可恢复备份。

## 发布前置

- 服务端 `/api/cli` 必须继续兼容旧生产二进制。
- macOS arm64/amd64 需要 Developer ID + notarization。
- Windows amd64 需要 Authenticode。
- 构建产物必须附带 SHA256SUMS.txt，并通过测试账号验证设备登录和配置回滚。
