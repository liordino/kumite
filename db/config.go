package db

import (
	"fmt"
)

// RuntimeLLM is the inference configuration resolved from app_config at call
// time — environment variables seed it once; the settings panel changes it
// without restart.
type RuntimeLLM struct {
	Endpoint string
	Model    string
	APIKey   string
}

// LoadRuntimeLLM reads the current LLM settings from app_config.
func (d *DB) LoadRuntimeLLM() (RuntimeLLM, error) {
	rows, err := d.Query(`SELECT key, value FROM app_config WHERE key IN ('llm_endpoint','llm_model','llm_api_key')`)
	if err != nil {
		return RuntimeLLM{}, fmt.Errorf("load runtime llm: %w", err)
	}
	defer rows.Close()

	var cfg RuntimeLLM
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return RuntimeLLM{}, fmt.Errorf("load runtime llm scan: %w", err)
		}
		switch k {
		case "llm_endpoint":
			cfg.Endpoint = v
		case "llm_model":
			cfg.Model = v
		case "llm_api_key":
			cfg.APIKey = v
		}
	}
	if err := rows.Err(); err != nil {
		return RuntimeLLM{}, fmt.Errorf("load runtime llm rows: %w", err)
	}
	return cfg, nil
}

// AllConfig returns every config key-value pair.
func (d *DB) AllConfig() (map[string]string, error) {
	rows, err := d.Query(`SELECT key, value FROM app_config ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("all config: %w", err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("all config scan: %w", err)
		}
		out[k] = v
	}
	return out, rows.Err()
}

// UpsertConfig writes config values. Takes effect on the next LLM call.
func (d *DB) UpsertConfig(kv map[string]string) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("upsert config begin: %w", err)
	}
	defer tx.Rollback()
	for k, v := range kv {
		if _, err := tx.Exec(
			`INSERT INTO app_config (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return fmt.Errorf("upsert config %s: %w", k, err)
		}
	}
	return tx.Commit()
}

// SeedLLMDefaults seeds the initial LLM settings from environment variables.
// INSERT OR IGNORE semantics: env seeds once, user edits survive restarts.
func (d *DB) SeedLLMDefaults(endpoint, model, apiKey string) error {
	return d.UpsertConfigOnIgnore(map[string]string{
		"llm_endpoint": endpoint,
		"llm_model":    model,
		"llm_api_key":  apiKey,
	})
}

// UpsertConfigOnIgnore inserts only keys that do not exist yet.
func (d *DB) UpsertConfigOnIgnore(kv map[string]string) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("seed config begin: %w", err)
	}
	defer tx.Rollback()
	for k, v := range kv {
		if _, err := tx.Exec(
			`INSERT INTO app_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO NOTHING`,
			k, v,
		); err != nil {
			return fmt.Errorf("seed config %s: %w", k, err)
		}
	}
	return tx.Commit()
}
