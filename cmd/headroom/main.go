// Package main — headroom CLI 入口。
// 子命令：compress / proxy / version
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	headroom "github.com/superops-team/headroom-go"
	"github.com/superops-team/headroom-go/internal/mcp"
	"github.com/superops-team/headroom-go/internal/wrap"
	"github.com/superops-team/headroom-go/proxy"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	fs := flag.NewFlagSet(subcmd, flag.ExitOnError)

	switch subcmd {
	case "compress":
		runCompress(fs)
	case "proxy":
		runProxy(fs)
	case "mcp":
		runMCP(fs)
	case "wrap":
		runWrap(fs)
	case "version":
		fmt.Println("headroom-go " + headroom.Version)
	default:
		logger.Error("unknown command", "cmd", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func runCompress(fs *flag.FlagSet) {
	aggressive := fs.Float64("aggressiveness", 0.5, "压缩强度 0.0-1.0（0.5 默认）")
	noRev := fs.Bool("no-reversible", false, "关闭可逆压缩（不附加 retrieve id）")
	noAlign := fs.Bool("no-align", false, "关闭前缀对齐")
	tokenizerBackend := fs.String("tokenizer-backend", "", "tokenizer backend: fallback/tiktoken/huggingface")
	tokenBudget := fs.Int("token-budget", 0, "目标 token budget（0 表示不限制）")
	enablePipeline := fs.Bool("enable-pipeline", false, "启用 Spec A pipeline 压缩路径")
	query := fs.String("query", "", "用于 diff/search scoring 的查询词")
	input := fs.String("input", "", "输入文件（默认 stdin）")
	output := fs.String("output", "", "输出文件（默认 stdout）")
	stats := fs.Bool("stats", false, "打印 token 统计到 stderr")
	fs.Parse(os.Args[2:])
	if err := validateTokenizerBackend(*tokenizerBackend); err != nil {
		logger.Error("invalid tokenizer backend", "err", err)
		os.Exit(1)
	}

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = *aggressive
	opts.Reversible = !*noRev
	opts.AlignPrefix = *noAlign == false
	opts.TokenizerConfig.Backend = headroom.TokenizerBackend(*tokenizerBackend)
	opts.TokenizerConfig.AllowFallback = true
	opts.TokenBudget = *tokenBudget
	opts.EnablePipeline = *enablePipeline
	opts.Query = *query

	var reader io.Reader = os.Stdin
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			logger.Error("failed to open input", "err", err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		logger.Error("failed to read input", "err", err)
		os.Exit(1)
	}

	// 将输入作为一条 user 消息压缩
	out, err := headroom.CompressString(string(data), opts)
	if err != nil {
		logger.Error("compression failed", "err", err)
		os.Exit(1)
	}

	var writer io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			logger.Error("failed to create output", "err", err)
			os.Exit(1)
		}
		defer f.Close()
		writer = f
	}

	io.WriteString(writer, out)

	if *stats {
		tok, _, err := headroom.NewTokenizer(opts.TokenizerConfig)
		if err != nil {
			logger.Error("tokenizer failed", "err", err)
			os.Exit(1)
		}
		orig, _ := tok.Count(string(data))
		comp, _ := tok.Count(out)
		savings := 0.0
		if orig > 0 {
			savings = float64(orig-comp) / float64(orig) * 100
		}
		logger.Info("compression stats", "original_tokens", orig, "compressed_tokens", comp, "savings_pct", fmt.Sprintf("%.1f%%", savings))
	}
}

func validateTokenizerBackend(backend string) error {
	switch headroom.TokenizerBackend(backend) {
	case "", headroom.TokenizerFallback, headroom.TokenizerTiktoken, headroom.TokenizerHF:
		return nil
	default:
		return fmt.Errorf("%q (valid: fallback, tiktoken, huggingface)", backend)
	}
}

func runProxy(fs *flag.FlagSet) {
	port := fs.Int("port", 8787, "监听端口")
	upstream := fs.String("upstream", "https://api.openai.com/v1", "上游 Base URL")
	aggressive := fs.Float64("aggressiveness", 0.5, "压缩强度 0.0-1.0")
	noRev := fs.Bool("no-reversible", false, "关闭可逆压缩")
	enablePipeline := fs.Bool("enable-pipeline", false, "启用 Spec A pipeline 压缩路径")
	tokenBudget := fs.Int("token-budget", 0, "目标 token budget（0 表示不限制）")
	apiKey := os.Getenv("HEADROOM_API_KEY")
	fs.Parse(os.Args[2:])

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = *aggressive
	opts.Reversible = !*noRev
	opts.EnablePipeline = *enablePipeline
	opts.TokenBudget = *tokenBudget

	cfg := proxy.Config{
		UpstreamBaseURL: *upstream,
		APIKey:          apiKey,
		ListenAddr:      fmt.Sprintf(":%d", *port),
		CompressOptions: opts,
	}
	handler := proxy.NewProxy(cfg)

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler,
	}

	// 监听 OS 信号，实现优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", "err", err)
		}
	}()

	logger.Info("proxy listening", "port", *port, "upstream", *upstream)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}

func runMCP(fs *flag.FlagSet) {
	fs.Parse(os.Args[2:])
	if fs.NArg() == 0 || fs.Arg(0) != "serve" {
		logger.Error("mcp requires subcommand: serve")
		fmt.Fprintf(os.Stderr, "Usage: headroom mcp serve\n")
		os.Exit(1)
	}
	if err := mcp.Serve(); err != nil {
		logger.Error("mcp server error", "err", err)
		os.Exit(1)
	}
}

func runWrap(fs *flag.FlagSet) {
	agent := fs.String("agent", "", "Target agent: claude, codex, copilot, generic")
	port := fs.Int("port", 18787, "Proxy port")
	upstream := fs.String("upstream", "", "Upstream LLM API base URL")
	apply := fs.Bool("apply", false, "Auto-apply configuration changes")
	fs.Parse(os.Args[2:])

	// 支持位置参数: headroom wrap claude
	if *agent == "" && fs.NArg() > 0 {
		*agent = fs.Arg(0)
	}

	if *agent == "" {
		logger.Error("wrap requires an agent type")
		fmt.Fprintf(os.Stderr, "Usage: headroom wrap <claude|codex|copilot|generic> [--apply] [--port=18787]\n")
		os.Exit(1)
	}

	if err := wrap.Run(wrap.Config{
		Agent:    *agent,
		Port:     *port,
		Upstream: *upstream,
		Apply:    *apply,
	}); err != nil {
		logger.Error("wrap error", "err", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "headroom-go %s — AI 上下文压缩层\n\n%s", headroom.Version, `Usage:
  headroom <command> [flags]

Commands:
  compress    压缩 stdin 或文件
  proxy       启动 HTTP 代理（OpenAI 兼容）
  mcp         启动 MCP Server（stdio 模式）
  wrap        启动代理 + 配置 IDE/Agent
  version     打印版本

Examples:
  cat long.txt | headroom compress --stats
  headroom compress --input=input.json --output=output.txt
  headroom proxy --port=8787
  headroom mcp serve
  headroom wrap claude
  headroom wrap codex --apply
`)
}
