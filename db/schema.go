package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"kumite/models"
)

// Migrate creates all tables and indexes if missing and seeds app_config
// defaults. Idempotent — safe to run on every boot.
func (d *DB) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id              TEXT PRIMARY KEY,
			project_name    TEXT NOT NULL DEFAULT 'Untitled Project',
			domain          TEXT NOT NULL DEFAULT '',
			tags            TEXT NOT NULL DEFAULT '[]',
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL,
			phase           TEXT NOT NULL DEFAULT 'intake',
			raw_source      TEXT,
			raw_input       TEXT NOT NULL DEFAULT '',
			pipeline_plan   TEXT,
			agent_outputs   TEXT NOT NULL DEFAULT '[]',
			psd             TEXT,
			finding_summary TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_domain   ON sessions(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_phase    ON sessions(phase)`,
		`CREATE TABLE IF NOT EXISTS agent_cache (
			agent_id    TEXT PRIMARY KEY,
			source_url  TEXT NOT NULL,
			content     TEXT NOT NULL,
			fetched_at  TEXT NOT NULL,
			etag        TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS custom_agents (
			id               TEXT PRIMARY KEY,
			display_name     TEXT NOT NULL,
			role_summary     TEXT NOT NULL,
			wave_preference  INTEGER NOT NULL DEFAULT 1,
			system_prompt    TEXT NOT NULL,
			created_at       TEXT NOT NULL,
			updated_at       TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS app_config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		// Seeded defaults (Ollama Cloud). Runtime values are changed through the
		// settings panel; INSERT OR IGNORE keeps user edits across boots.
		`INSERT OR IGNORE INTO app_config VALUES ('llm_endpoint', 'https://ollama.com/v1')`,
		`INSERT OR IGNORE INTO app_config VALUES ('llm_model',    'qwen3:32b')`,
		`INSERT OR IGNORE INTO app_config VALUES ('llm_api_key',  '')`,
	}
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			return fmt.Errorf("db migrate: %w (stmt: %.80s)", err, s)
		}
	}
	return nil
}

// ReconcileOrphanedRuns implements startup reconciliation: the server is a
// single process, so any session in `running` or `synthesis` at boot is
// orphaned by definition. It moves those sessions to `interrupted` and resets
// any AgentNode left at `running` back to `pending`. Nothing is deleted —
// partial agent_outputs stay exactly as they are.
func (d *DB) ReconcileOrphanedRuns() error {
	rows, err := d.Query(`SELECT id, pipeline_plan FROM sessions WHERE phase IN ('running','synthesis')`)
	if err != nil {
		return fmt.Errorf("reconcile select: %w", err)
	}
	defer rows.Close()

	type orphan struct {
		id   string
		plan *models.PipelinePlan
	}
	var orphans []orphan
	for rows.Next() {
		var id string
		var planRaw sql.NullString
		if err := rows.Scan(&id, &planRaw); err != nil {
			return fmt.Errorf("reconcile scan: %w", err)
		}
		o := orphan{id: id}
		if planRaw.Valid && planRaw.String != "" {
			var p models.PipelinePlan
			if err := json.Unmarshal([]byte(planRaw.String), &p); err != nil {
				return fmt.Errorf("reconcile plan %s: %w", id, err)
			}
			o.plan = &p
		}
		orphans = append(orphans, o)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reconcile rows: %w", err)
	}
	if len(orphans) == 0 {
		return nil
	}

	// One bulk phase flip covers every orphaned session…
	if _, err := d.Exec(
		`UPDATE sessions SET phase = ?, updated_at = ? WHERE phase IN ('running','synthesis')`,
		models.PhaseInterrupted, NowUTC(),
	); err != nil {
		return fmt.Errorf("reconcile bulk update: %w", err)
	}
	// …and only sessions with a persisted plan need a per-row follow-up,
	// because each plan JSON is rewritten individually to reset running nodes.
	for _, o := range orphans {
		if o.plan == nil {
			continue
		}
		resetRunningNodes(o.plan)
		raw, err := json.Marshal(o.plan)
		if err != nil {
			return fmt.Errorf("reconcile marshal %s: %w", o.id, err)
		}
		if _, err := d.Exec(
			`UPDATE sessions SET pipeline_plan = ? WHERE id = ?`,
			string(raw), o.id,
		); err != nil {
			return fmt.Errorf("reconcile plan reset %s: %w", o.id, err)
		}
	}
	return nil
}

// resetRunningNodes moves every node left at `running` back to `pending`.
// A node at `running` is one whose output was never persisted, so re-running
// it on resume is safe and never duplicates an output.
func resetRunningNodes(p *models.PipelinePlan) {
	waves := []*[]models.AgentNode{&p.Pipeline.Wave1, &p.Pipeline.Wave2, &p.Pipeline.Fixed}
	for _, w := range waves {
		for i := range *w {
			if (*w)[i].Status == models.StatusRunning {
				(*w)[i].Status = models.StatusPending
			}
		}
	}
}
