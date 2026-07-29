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

## 2026-07-28 验证证据

- GitHub 仓库：`ZYD3342348/ciyuanmax-coding-assistant`。
- 实现提交：`fb89f60 feat: add cross-platform Go CLI v2`。
- Windows CI 修复：`c070613 fix: make cli tests Windows-compatible`。
- GitHub Actions 运行 `30335392588` 全部通过：Ubuntu、macOS、Windows 测试和 release build 均成功。
- 本机通过 `go test ./...`、`go vet ./...`、Windows 目标编译测试和 macOS arm64/amd64、Windows amd64 交叉构建。
- Windows 不提供 POSIX `0600` 权限位；会话文件依赖 `%AppData%` 的当前用户 ACL。测试只在 Unix 平台断言 `0600`，不降低 Windows 的实际目录访问边界。
- 本次没有替换生产下载页或生产二进制，也没有修改模型、账号、路由、价格和备用账号。
