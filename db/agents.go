package db

import (
	"database/sql"
	"errors"
	"fmt"

	"kumite/models"
)

// AgentCacheRow is one row of agent_cache: a fetched specialist prompt with
// its ETag for conditional revalidation.
type AgentCacheRow struct {
	AgentID   string
	SourceURL string
	Content   string
	FetchedAt string
	ETag      string
}

// GetAgentCache returns the cached prompt for agentID, or ErrNotFound.
func (d *DB) GetAgentCache(agentID string) (AgentCacheRow, error) {
	row := d.QueryRow(
		`SELECT agent_id, source_url, content, fetched_at, COALESCE(etag,'') FROM agent_cache WHERE agent_id = ?`,
		agentID)
	var r AgentCacheRow
	var etag sql.NullString
	err := row.Scan(&r.AgentID, &r.SourceURL, &r.Content, &r.FetchedAt, &etag)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentCacheRow{}, ErrNotFound
	}
	if err != nil {
		return AgentCacheRow{}, fmt.Errorf("get agent cache: %w", err)
	}
	r.ETag = etag.String
	return r, nil
}

// UpsertAgentCache stores fetched content and its ETag.
func (d *DB) UpsertAgentCache(agentID, sourceURL, content, etag string) error {
	_, err := d.Exec(
		`INSERT INTO agent_cache (agent_id, source_url, content, fetched_at, etag)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(agent_id) DO UPDATE SET
		   source_url = excluded.source_url,
		   content = excluded.content,
		   fetched_at = excluded.fetched_at,
		   etag = excluded.etag`,
		agentID, sourceURL, content, NowUTC(), etag,
	)
	if err != nil {
		return fmt.Errorf("upsert agent cache: %w", err)
	}
	return nil
}

// TouchAgentCache resets the TTL (fetched_at) without re-downloading — used
// when a conditional GET answers 304.
func (d *DB) TouchAgentCache(agentID string) error {
	_, err := d.Exec(`UPDATE agent_cache SET fetched_at = ? WHERE agent_id = ?`, NowUTC(), agentID)
	if err != nil {
		return fmt.Errorf("touch agent cache: %w", err)
	}
	return nil
}

// CreateCustomAgent inserts a user-defined specialist.
func (d *DB) CreateCustomAgent(a models.CustomAgent) (models.CustomAgent, error) {
	if a.ID == "" {
		a.ID = NewID()
	}
	now := NowUTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	_, err := d.Exec(
		`INSERT INTO custom_agents (id, display_name, role_summary, wave_preference, system_prompt, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.DisplayName, a.RoleSummary, a.WavePreference, a.SystemPrompt, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return models.CustomAgent{}, fmt.Errorf("create custom agent: %w", err)
	}
	return a, nil
}

// ListCustomAgents returns all user-defined agents, creation order.
func (d *DB) ListCustomAgents() ([]models.CustomAgent, error) {
	rows, err := d.Query(
		`SELECT id, display_name, role_summary, wave_preference, system_prompt, created_at, updated_at
		 FROM custom_agents ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list custom agents: %w", err)
	}
	defer rows.Close()

	var out []models.CustomAgent
	for rows.Next() {
		var a models.CustomAgent
		if err := rows.Scan(&a.ID, &a.DisplayName, &a.RoleSummary, &a.WavePreference, &a.SystemPrompt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list custom agents scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetCustomAgent returns one custom agent, or ErrNotFound.
func (d *DB) GetCustomAgent(id string) (models.CustomAgent, error) {
	row := d.QueryRow(
		`SELECT id, display_name, role_summary, wave_preference, system_prompt, created_at, updated_at
		 FROM custom_agents WHERE id = ?`, id)
	var a models.CustomAgent
	err := row.Scan(&a.ID, &a.DisplayName, &a.RoleSummary, &a.WavePreference, &a.SystemPrompt, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CustomAgent{}, ErrNotFound
	}
	if err != nil {
		return models.CustomAgent{}, fmt.Errorf("get custom agent: %w", err)
	}
	return a, nil
}

// UpdateCustomAgent replaces a custom agent definition.
func (d *DB) UpdateCustomAgent(a models.CustomAgent) error {
	res, err := d.Exec(
		`UPDATE custom_agents SET display_name = ?, role_summary = ?, wave_preference = ?, system_prompt = ?, updated_at = ?
		 WHERE id = ?`,
		a.DisplayName, a.RoleSummary, a.WavePreference, a.SystemPrompt, NowUTC(), a.ID,
	)
	if err != nil {
		return fmt.Errorf("update custom agent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCustomAgent removes a custom agent.
func (d *DB) DeleteCustomAgent(id string) error {
	res, err := d.Exec(`DELETE FROM custom_agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete custom agent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
