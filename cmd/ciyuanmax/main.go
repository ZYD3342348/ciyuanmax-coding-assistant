package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/api"
	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/configure"
	"github.com/ZYD3342348/ciyuanmax-coding-assistant/cli/internal/store"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ciyuanmax: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return interactive()
	}
	switch args[0] {
	case "login":
		return login(context.Background())
	case "configure", "setup":
		return configureCommand(args[1:])
	case "status":
		return statusCommand()
	case "modes":
		return modesCommand()
	case "keys":
		return keysCommand()
	case "rotate":
		return rotateCommand(args[1:])
	case "revoke":
		return revokeCommand(args[1:])
	case "doctor":
		return doctorCommand()
	case "logout":
		return logoutCommand()
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage() {
	fmt.Println(`CiYuanMax 编程助手 CLI

用法:
  ciyuanmax                 打开交互菜单
  ciyuanmax login           浏览器登录并授权这台电脑
  ciyuanmax configure       创建凭证并写入 Claude Code / Codex 配置
  ciyuanmax status          查看账户、余额和工具状态
  ciyuanmax modes           查看可用使用模式
  ciyuanmax keys            查看当前设备凭证（只显示掩码）
  ciyuanmax rotate          轮换凭证并更新本地配置
  ciyuanmax revoke --id ID  吊销一张凭证
  ciyuanmax doctor          检查登录、工具安装和本地配置
  ciyuanmax logout          清除本机 CLI 登录状态

configure / rotate 选项:
  --mode stable|economy     使用模式（默认 stable）
  --tools claude,codex      要配置的工具（默认 claude,codex）`)
}

func interactive() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nCiYuanMax 编程助手")
		fmt.Println("1. 首次配置 / 重新配置")
		fmt.Println("2. 查看状态")
		fmt.Println("3. 查看使用模式")
		fmt.Println("4. 诊断")
		fmt.Println("5. 管理凭证")
		fmt.Println("0. 退出")
		choice, err := prompt(reader, "请选择: ")
		if err != nil {
			return err
		}
		switch choice {
		case "1":
			if err := configureCommand(nil); err != nil {
				fmt.Printf("配置失败: %v\n", err)
			}
		case "2":
			if err := statusCommand(); err != nil {
				fmt.Printf("查询失败: %v\n", err)
			}
		case "3":
			if err := modesCommand(); err != nil {
				fmt.Printf("查询失败: %v\n", err)
			}
		case "4":
			if err := doctorCommand(); err != nil {
				fmt.Printf("诊断发现问题: %v\n", err)
			}
		case "5":
			if err := keysCommand(); err != nil {
				fmt.Printf("查询失败: %v\n", err)
			}
		case "0", "q", "quit", "exit":
			return nil
		default:
			fmt.Println("请输入菜单编号")
		}
	}
}

func configureCommand(args []string) error {
	flags := flag.NewFlagSet("configure", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	mode := flags.String("mode", "stable", "usage mode")
	tools := flags.String("tools", "claude,codex", "comma-separated tools")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := validateMode(*mode); err != nil {
		return err
	}
	selectedTools, err := parseTools(*tools)
	if err != nil {
		return err
	}
	client, session, err := authenticatedClient()
	if err != nil {
		return err
	}
	modes, err := client.Modes(context.Background())
	if err != nil {
		return err
	}
	if err := ensureModeAvailable(modes, *mode, selectedTools); err != nil {
		return err
	}
	paths, err := configure.DefaultPaths()
	if err != nil {
		return err
	}
	for _, tool := range selectedTools {
		key, err := client.EnsureKey(context.Background(), api.EnsureKeyRequest{Tool: tool, Mode: *mode, DeviceName: session.DeviceName})
		if err != nil {
			return fmt.Errorf("create %s credential: %w", tool, err)
		}
		result, err := configure.Apply(paths, key)
		if err != nil {
			return err
		}
		fmt.Printf("已配置 %-6s %s\n", tool, result.Path)
		if result.BackupPath != "" {
			fmt.Printf("  备份: %s\n", result.BackupPath)
		}
	}
	fmt.Println("配置完成。请重新打开 Claude Code 或 Codex 使配置生效。")
	return nil
}

func statusCommand() error {
	client, _, err := authenticatedClient(false)
	if err != nil {
		return err
	}
	status, err := client.Status(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("账户: %s\n余额: ¥%s\n本月消费: ¥%s\n工具: %s\n", status.Email, status.BalanceCNY, status.MonthSpendCNY, strings.Join(status.AvailableTools, ", "))
	for _, tool := range []string{"claude", "codex"} {
		fmt.Printf("模式 %-6s: %s\n", tool, status.CurrentModes[tool])
	}
	return nil
}

func modesCommand() error {
	client, _, err := authenticatedClient(false)
	if err != nil {
		return err
	}
	modes, err := client.Modes(context.Background())
	if err != nil {
		return err
	}
	for _, mode := range modes {
		marker := " "
		if mode.Recommended {
			marker = "*"
		}
		fmt.Printf("%s %-8s %s [%s]\n  %s\n", marker, mode.ID, mode.Title, strings.Join(mode.Tools, ", "), mode.Description)
	}
	return nil
}

func keysCommand() error {
	client, session, err := authenticatedClient(false)
	if err != nil {
		return err
	}
	keys, err := client.ListKeys(context.Background(), session.DeviceName)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		fmt.Println("当前设备没有 CLI 凭证。请先运行 ciyuanmax configure。")
		return nil
	}
	for _, key := range keys {
		status := key.Status
		if key.Current {
			status += " current"
		}
		fmt.Printf("%d %-6s %-8s %-12s %s %s\n", key.KeyID, key.Tool, key.Mode, status, key.MaskedKey, key.CreatedAt)
	}
	return nil
}

func rotateCommand(args []string) error {
	flags := flag.NewFlagSet("rotate", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	mode := flags.String("mode", "stable", "usage mode")
	tools := flags.String("tools", "claude,codex", "comma-separated tools")
	if err := flags.Parse(args); err != nil {
		return err
	}
	selectedTools, err := parseTools(*tools)
	if err != nil {
		return err
	}
	client, session, err := authenticatedClient()
	if err != nil {
		return err
	}
	paths, err := configure.DefaultPaths()
	if err != nil {
		return err
	}
	for _, tool := range selectedTools {
		key, err := client.RotateKey(context.Background(), api.EnsureKeyRequest{Tool: tool, Mode: *mode, DeviceName: session.DeviceName})
		if err != nil {
			return fmt.Errorf("rotate %s credential: %w", tool, err)
		}
		result, err := configure.Apply(paths, key)
		if err != nil {
			return err
		}
		fmt.Printf("已轮换 %-6s，备份: %s\n", tool, result.BackupPath)
	}
	return nil
}

func revokeCommand(args []string) error {
	flags := flag.NewFlagSet("revoke", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	id := flags.Int64("id", 0, "key id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *id <= 0 {
		return errors.New("--id must be a positive key id")
	}
	client, _, err := authenticatedClient(false)
	if err != nil {
		return err
	}
	if err := client.RevokeKey(context.Background(), *id); err != nil {
		return err
	}
	fmt.Printf("已吊销凭证 %d。\n", *id)
	return nil
}

func doctorCommand() error {
	session, err := store.Load()
	if err != nil {
		return err
	}
	problems := make([]string, 0)
	if strings.TrimSpace(session.AccessToken) == "" || (!session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt)) {
		problems = append(problems, "尚未登录或 CLI 登录已过期")
	} else {
		client := api.NewClient(session.BaseURL, session.AccessToken)
		status, statusErr := client.Status(context.Background())
		if statusErr != nil {
			problems = append(problems, "无法使用本机会话连接 CiYuanMax: "+statusErr.Error())
		} else {
			fmt.Printf("网关连接: 正常 (%s)\n", status.Email)
		}
	}
	for _, tool := range []string{"claude", "codex"} {
		if path, err := exec.LookPath(tool); err == nil {
			fmt.Printf("工具 %-6s: %s\n", tool, path)
		} else {
			problems = append(problems, fmt.Sprintf("未找到 %s 命令", tool))
		}
	}
	paths, err := configure.DefaultPaths()
	if err != nil {
		return err
	}
	for _, path := range []string{paths.ClaudeSettings, paths.CodexConfig} {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("配置 %-6s: 尚未创建 (%s)\n", filepath.Base(filepath.Dir(path)), path)
		} else if err != nil {
			problems = append(problems, fmt.Sprintf("无法读取配置 %s", path))
		} else {
			fmt.Printf("配置 %-6s: %s\n", filepath.Base(filepath.Dir(path)), path)
		}
	}
	if len(problems) > 0 {
		fmt.Println("修复建议:")
		fmt.Println("- 登录问题: ciyuanmax login")
		fmt.Println("- 配置问题: ciyuanmax configure")
		fmt.Println("- Claude Code: https://docs.anthropic.com/en/docs/claude-code/setup")
		fmt.Println("- Codex: npm install -g @openai/codex")
		return errors.New(strings.Join(problems, "; "))
	}
	fmt.Println("诊断通过")
	return nil
}

func logoutCommand() error {
	session, err := store.Load()
	if err != nil {
		return err
	}
	session.AccessToken = ""
	session.ExpiresAt = time.Time{}
	if err := store.Save(session); err != nil {
		return err
	}
	fmt.Println("已清除本机 CLI 登录状态。服务器凭证未吊销；如需吊销请运行 ciyuanmax revoke。")
	return nil
}

func authenticatedClient(autoLogin ...bool) (*api.Client, store.Session, error) {
	shouldLogin := len(autoLogin) == 0 || autoLogin[0]
	session, err := store.Load()
	if err != nil {
		return nil, store.Session{}, err
	}
	if session.AccessToken == "" || (!session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt)) {
		if !shouldLogin {
			return nil, session, errors.New("尚未登录，请先运行 ciyuanmax login")
		}
		if err := login(context.Background()); err != nil {
			return nil, session, err
		}
		session, err = store.Load()
		if err != nil {
			return nil, session, err
		}
	}
	return api.NewClient(session.BaseURL, session.AccessToken), session, nil
}

func login(ctx context.Context) error {
	session, err := store.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(session.BaseURL, "")
	device, err := client.StartDevice(ctx)
	if err != nil {
		return fmt.Errorf("start browser login: %w", err)
	}
	fmt.Printf("请在浏览器确认这台电脑:\n%s\n验证码: %s\n", device.VerificationURIComplete, device.UserCode)
	if err := openBrowser(device.VerificationURIComplete); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动复制上面的链接 (%v)\n", err)
	}
	interval := time.Duration(device.Interval) * time.Second
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}
	expires := time.Duration(device.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = 10 * time.Minute
	}
	pollContext, cancel := context.WithTimeout(ctx, expires)
	defer cancel()
	for {
		poll, err := client.PollDevice(pollContext, device.DeviceCode)
		if err != nil {
			return fmt.Errorf("poll browser login: %w", err)
		}
		if poll.Status == "approved" && poll.AccessToken != "" {
			session.AccessToken = poll.AccessToken
			session.ExpiresAt = time.Now().Add(time.Duration(poll.ExpiresIn) * time.Second)
			if session.DeviceName == "" {
				session.DeviceName = store.NewDeviceName()
			}
			if err := store.Save(session); err != nil {
				return err
			}
			fmt.Printf("登录成功，设备名: %s\n", session.DeviceName)
			return nil
		}
		fmt.Print("等待网页确认...\r")
		select {
		case <-pollContext.Done():
			return errors.New("浏览器登录超时，请重新运行 ciyuanmax login")
		case <-time.After(interval):
		}
	}
}

func openBrowser(target string) error {
	command := ""
	args := []string{}
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{target}
	case "windows":
		command, args = "rundll32", []string{"url.dll,FileProtocolHandler", target}
	default:
		command, args = "xdg-open", []string{target}
	}
	return exec.Command(command, args...).Start()
}

func prompt(reader *bufio.Reader, message string) (string, error) {
	fmt.Print(message)
	line, err := reader.ReadString('\n')
	return strings.TrimSpace(line), err
}

func parseTools(value string) ([]string, error) {
	seen := map[string]bool{}
	tools := make([]string, 0, 2)
	for _, item := range strings.Split(value, ",") {
		tool := strings.ToLower(strings.TrimSpace(item))
		if tool == "gpt" {
			tool = "codex"
		}
		if tool != "claude" && tool != "codex" {
			return nil, fmt.Errorf("unsupported tool %q", tool)
		}
		if !seen[tool] {
			seen[tool] = true
			tools = append(tools, tool)
		}
	}
	if len(tools) == 0 {
		return nil, errors.New("at least one tool is required")
	}
	return tools, nil
}

func validateMode(mode string) error {
	if mode != "stable" && mode != "economy" {
		return fmt.Errorf("unsupported mode %q; use stable or economy", mode)
	}
	return nil
}

func ensureModeAvailable(modes []api.Mode, wanted string, tools []string) error {
	for _, mode := range modes {
		if mode.ID != wanted || !mode.Available {
			continue
		}
		available := map[string]bool{}
		for _, tool := range mode.Tools {
			available[tool] = true
		}
		for _, tool := range tools {
			if !available[tool] {
				return fmt.Errorf("mode %s does not support %s", wanted, tool)
			}
		}
		return nil
	}
	return fmt.Errorf("mode %s is unavailable for this account", wanted)
}
