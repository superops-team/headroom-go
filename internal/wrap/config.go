package wrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// configBackup 保存原始配置以便恢复。
type configBackup struct {
	Agent  string
	Path   string
	Data   []byte
	IsJSON bool
}

// configureAgent 根据 Agent 类型修改配置文件。
func configureAgent(cfg Config) (*configBackup, error) {
	switch cfg.Agent {
	case AgentClaude:
		return configureClaude(cfg)
	case AgentCodex:
		return configureCodex(cfg)
	case AgentCopilot:
		return configureCopilot(cfg)
	case AgentGeneric:
		printGenericConfig(cfg)
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported agent: %s (try: claude, codex, copilot, generic)", cfg.Agent)
	}
}

// restoreConfig 恢复原始配置。
func restoreConfig(b *configBackup) error {
	if b == nil || b.Path == "" {
		return nil
	}
	return os.WriteFile(b.Path, b.Data, 0644)
}

// ── Claude Code ─────────────────────────────────────────────────────────────

func configureClaude(cfg Config) (*configBackup, error) {
	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	if !cfg.Apply {
		fmt.Printf(`
To use headroom with Claude Code, add to %s:

{
  "api": {
    "baseUrl": "http://127.0.0.1:%d/v1"
  }
}

Or run: headroom wrap claude --apply
`, settingsPath, cfg.Port)
		return nil, nil
	}

	// 备份原配置
	var backup *configBackup
	if data, err := os.ReadFile(settingsPath); err == nil {
		backup = &configBackup{Agent: AgentClaude, Path: settingsPath, Data: data, IsJSON: true}
	}

	// 读取或创建配置
	settings := make(map[string]any)
	if backup != nil {
		json.Unmarshal(backup.Data, &settings)
	}

	api, _ := settings["api"].(map[string]any)
	if api == nil {
		api = make(map[string]any)
	}
	api["baseUrl"] = fmt.Sprintf("http://127.0.0.1:%d/v1", cfg.Port)
	settings["api"] = api

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return backup, err
	}

	os.MkdirAll(filepath.Dir(settingsPath), 0755)
	if err := os.WriteFile(settingsPath, newData, 0644); err != nil {
		return backup, err
	}

	fmt.Printf("✓ Claude Code configured: %s\n", settingsPath)
	return backup, nil
}

// ── Codex ───────────────────────────────────────────────────────────────────

func configureCodex(cfg Config) (*configBackup, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".codex", "config.yaml")

	if !cfg.Apply {
		fmt.Printf(`
To use headroom with Codex, add to %s:

api_base: http://127.0.0.1:%d/v1

Or run: headroom wrap codex --apply
`, configPath, cfg.Port)
		return nil, nil
	}

	var backup *configBackup
	if data, err := os.ReadFile(configPath); err == nil {
		backup = &configBackup{Agent: AgentCodex, Path: configPath, Data: data}
	}

	content := fmt.Sprintf("api_base: http://127.0.0.1:%d/v1\n", cfg.Port)
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return backup, err
	}

	fmt.Printf("✓ Codex configured: %s\n", configPath)
	return backup, nil
}

// ── Copilot CLI ─────────────────────────────────────────────────────────────

func configureCopilot(cfg Config) (*configBackup, error) {
	if !cfg.Apply {
		fmt.Printf(`
To use headroom with GitHub Copilot CLI, set environment variable:

  export OPENAI_BASE_URL=http://127.0.0.1:%d/v1

Or run: headroom wrap copilot --apply
`, cfg.Port)
		return nil, nil
	}

	// Copilot CLI 通过环境变量配置，打印设置指令
	fmt.Printf(`
✓ Set environment variable for Copilot CLI:

  export OPENAI_BASE_URL=http://127.0.0.1:%d/v1

Run your Copilot commands in this shell session.
`, cfg.Port)
	return nil, nil
}

// ── Generic ─────────────────────────────────────────────────────────────────

func printGenericConfig(cfg Config) {
	fmt.Printf(`
╔══════════════════════════════════════════════════════════╗
║  Generic Configuration                                   ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║  Set your LLM client's API base URL to:                  ║
║                                                          ║
║    http://127.0.0.1:%d/v1                                ║
║                                                          ║
║  OpenAI SDK:                                             ║
║    client := openai.NewClient("sk-xxx")                  ║
║    client.BaseURL = "http://127.0.0.1:%d/v1"             ║
║                                                          ║
║  Anthropic SDK:                                          ║
║    export ANTHROPIC_BASE_URL=http://127.0.0.1:%d/v1      ║
║                                                          ║
║  cURL:                                                   ║
║    curl http://127.0.0.1:%d/v1/chat/completions \       ║
║      -H "Content-Type: application/json" \              ║
║      -d '{"model":"gpt-4","messages":[...]}'            ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
`, cfg.Port, cfg.Port, cfg.Port, cfg.Port)
}
