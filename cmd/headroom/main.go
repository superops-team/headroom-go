// Package main — headroom CLI 入口。
// 子命令：compress / proxy / version
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	headroom "github.com/superops-team/headroom-go"
	"github.com/superops-team/headroom-go/proxy"
)

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
	case "version":
		fmt.Println("headroom-go v0.1.0")
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func runCompress(fs *flag.FlagSet) {
	aggressive := fs.Float64("aggressiveness", 0.5, "压缩强度 0.0-1.0（0.5 默认）")
	noRev := fs.Bool("no-reversible", false, "关闭可逆压缩（不附加 retrieve id）")
	noAlign := fs.Bool("no-align", false, "关闭前缀对齐")
	input := fs.String("input", "", "输入文件（默认 stdin）")
	output := fs.String("output", "", "输出文件（默认 stdout）")
	stats := fs.Bool("stats", false, "打印 token 统计到 stderr")
	fs.Parse(os.Args[2:])

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = *aggressive
	opts.Reversible = !*noRev
	opts.AlignPrefix = *noAlign == false

	var reader io.Reader = os.Stdin
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to open input:", err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read input:", err)
		os.Exit(1)
	}

	// 将输入作为一条 user 消息压缩
	out, err := headroom.CompressString(string(data), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "compression failed:", err)
		os.Exit(1)
	}

	var writer io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to create output:", err)
			os.Exit(1)
		}
		defer f.Close()
		writer = f
	}

	io.WriteString(writer, out)

	if *stats {
		orig := len(data) / 4
		comp := len(out) / 4
		savings := 0.0
		if orig > 0 {
			savings = float64(orig-comp) / float64(orig) * 100
		}
		fmt.Fprintf(os.Stderr, "Original: %d tokens | Compressed: %d tokens | Savings: %.1f%%\n",
			orig, comp, savings)
	}
}

func runProxy(fs *flag.FlagSet) {
	port := fs.Int("port", 8787, "监听端口")
	upstream := fs.String("upstream", "https://api.openai.com/v1", "上游 Base URL")
	aggressive := fs.Float64("aggressiveness", 0.5, "压缩强度 0.0-1.0")
	noRev := fs.Bool("no-reversible", false, "关闭可逆压缩")
	apiKey := os.Getenv("HEADROOM_API_KEY")
	fs.Parse(os.Args[2:])

	opts := headroom.DefaultOptions()
	opts.Aggressiveness = *aggressive
	opts.Reversible = !*noRev

	cfg := proxy.Config{
		UpstreamBaseURL: *upstream,
		APIKey:          apiKey,
		ListenAddr:      fmt.Sprintf(":%d", *port),
		CompressOptions: opts,
	}
	handler := proxy.NewProxy(cfg)

	fmt.Fprintf(os.Stderr, "headroom proxy listening on :%d (upstream: %s)\n", *port, *upstream)
	if err := http.ListenAndServe(cfg.ListenAddr, handler); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `headroom-go v0.1.0 — AI 上下文压缩层

Usage:
  headroom <command> [flags]

Commands:
  compress    压缩 stdin 或文件
  proxy       启动 HTTP 代理（OpenAI 兼容）
  version     打印版本

Examples:
  cat long.txt | headroom compress --stats
  headroom compress --input=input.json --output=output.txt
  headroom proxy --port=8787
`)
}
