package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config is the process-level configuration loaded from environment
// variables. Env vars seed initial values only — runtime LLM settings live in
// the app_config table and change through the settings panel without restart.
type Config struct {
	Port               int
	DBPath             string
	AgentCacheTTLHours int
	GitHubRawBase      string
	LLMMock            bool
	// Seed values for app_config (applied once, INSERT OR IGNORE).
	LLMEndpoint string
	LLMModel    string
	LLMAPIKey   string
}

// Load reads configuration from the environment with defaults matching
// AGENTS.md.
func Load() (Config, error) {
	c := Config{
		Port:               3001,
		DBPath:             "./kumite.db",
		AgentCacheTTLHours: 24,
		GitHubRawBase:      "https://raw.githubusercontent.com/msitarzewski/agency-agents/main",
		LLMEndpoint:        "https://ollama.com/v1",
		LLMModel:           "qwen3:32b",
	}

	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("config PORT: %w", err)
		}
		c.Port = p
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = v
	}
	if v := os.Getenv("AGENT_CACHE_TTL_HOURS"); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("config AGENT_CACHE_TTL_HOURS: %w", err)
		}
		c.AgentCacheTTLHours = h
	}
	if v := os.Getenv("GITHUB_RAW_BASE"); v != "" {
		c.GitHubRawBase = v
	}
	if v := os.Getenv("LLM_MOCK"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("config LLM_MOCK: %w", err)
		}
		c.LLMMock = b
	}
	if v := os.Getenv("LLM_ENDPOINT"); v != "" {
		c.LLMEndpoint = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		c.LLMModel = v
	}
	c.LLMAPIKey = os.Getenv("LLM_API_KEY")

	return c, nil
}
