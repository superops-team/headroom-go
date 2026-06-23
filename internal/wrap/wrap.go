// Package wrap 实现 headroom wrap 命令，启动本地代理并自动配置 IDE/Agent。
//
// 支持的 Agent：
//   - claude (Claude Code)
//   - codex (OpenAI Codex)
//   - copilot (GitHub Copilot CLI)
//   - generic (打印通用配置指令)
package wrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	headroom "github.com/superops-team/headroom-go"
	"github.com/superops-team/headroom-go/proxy"
)

// Agent 类型常量。
const (
	AgentClaude  = "claude"
	AgentCodex   = "codex"
	AgentCopilot = "copilot"
	AgentGeneric = "generic"
)

// Config 是 wrap 命令的配置。
type Config struct {
	Agent    string // 目标 Agent 类型
	Port     int    // 代理端口
	Upstream string // 上游 LLM API
	Apply    bool   // 是否自动修改配置文件
}

// Run 启动 wrap 流程。
func Run(cfg Config) error {
	if cfg.Port == 0 {
		cfg.Port = 18787
	}

	// 1. 启动 proxy
	proxyCfg := proxy.Config{
		ListenAddr:      fmt.Sprintf(":%d", cfg.Port),
		UpstreamBaseURL: cfg.Upstream,
		CompressOptions: headroom.DefaultOptions(),
	}
	handler := proxy.NewProxy(proxyCfg)
	srv := &http.Server{Addr: proxyCfg.ListenAddr, Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "proxy error: %v\n", err)
		}
	}()

	// 等待 proxy 就绪
	time.Sleep(200 * time.Millisecond)

	// 2. 配置 Agent
	backup, err := configureAgent(cfg)
	if err != nil {
		return fmt.Errorf("configure agent: %w", err)
	}

	// 3. 打印启动信息
	printBanner(cfg)

	// 4. 等待退出信号
	<-ctx.Done()

	// 5. 恢复配置
	if backup != nil {
		if err := restoreConfig(backup); err != nil {
			fmt.Fprintf(os.Stderr, "restore config warning: %v\n", err)
		}
	}

	fmt.Println("\nheadroom wrap stopped.")
	return nil
}

func printBanner(cfg Config) {
	fmt.Printf(`
╔══════════════════════════════════════════════════════════╗
║  headroom wrap — %-8s                              ║
╠══════════════════════════════════════════════════════════╣
║  Proxy:  http://127.0.0.1:%d/v1                          ║
║  Status: running                                         ║
║  Press Ctrl+C to stop                                    ║
╚══════════════════════════════════════════════════════════╝

`, cfg.Agent, cfg.Port)
}
