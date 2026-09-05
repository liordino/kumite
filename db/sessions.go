package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"kumite/models"
)

// ErrNotFound is returned by lookups that match no row.
var ErrNotFound = errors.New("not found")

// SessionFilters mirrors the query parameters on GET /api/sessions.
type SessionFilters struct {
	Domain    string
	Phase     string
	Tag       string // exact match, single tag
	HasPsd    *bool
	Distilled *bool
}

// CreateSession inserts a new session. The submitted material goes to
// raw_input; raw_source stays NULL — its absence is the record that intake
// was skipped (or has not run yet).
func (d *DB) CreateSession(projectName, rawInput string) (models.Session, error) {
	if strings.TrimSpace(projectName) == "" {
		projectName = "Untitled Project"
	}
	now := NowUTC()
	id := NewID()
	_, err := d.Exec(
		`INSERT INTO sessions (id, project_name, created_at, updated_at, phase, raw_input)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, projectName, now, now, models.PhaseIntake, rawInput,
	)
	if err != nil {
		return models.Session{}, fmt.Errorf("create session: %w", err)
	}
	return d.GetSession(id)
}

// GetSession loads one full session, deserializing the JSON columns.
// raw_source is "" when the column is NULL (intake skipped).
func (d *DB) GetSession(id string) (models.Session, error) {
	row := d.QueryRow(
		`SELECT id, project_name, domain, tags, created_at, updated_at, phase,
		        raw_source, raw_input, pipeline_plan, agent_outputs, psd, finding_summary
		 FROM sessions WHERE id = ?`, id)
	var s models.Session
	var domain, tags, agentOutputs, findingSummary string
	var rawSource, planRaw, psd sql.NullString
	err := row.Scan(&s.ID, &s.ProjectName, &domain, &tags, &s.CreatedAt, &s.UpdatedAt, &s.Phase,
		&rawSource, &s.RawInput, &planRaw, &agentOutputs, &psd, &findingSummary)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Session{}, ErrNotFound
	}
	if err != nil {
		return models.Session{}, fmt.Errorf("get session: %w", err)
	}
	s.Domain = domain
	tagsList, err := parseStringArray(tags)
	if err != nil {
		return models.Session{}, fmt.Errorf("get session %s: %w", id, err)
	}
	s.Tags = tagsList
	s.RawSource = rawSource.String // NULL -> ""
	if planRaw.Valid && planRaw.String != "" {
		var p models.PipelinePlan
		if err := json.Unmarshal([]byte(planRaw.String), &p); err != nil {
			return models.Session{}, fmt.Errorf("get session plan %s: %w", id, err)
		}
		s.PipelinePlan = &p
	}
	outputs, err := parseOutputs(agentOutputs)
	if err != nil {
		return models.Session{}, fmt.Errorf("get session %s: %w", id, err)
	}
	s.AgentOutputs = outputs
	s.Psd = psd.String
	finding, err := parseFindingSummary(findingSummary)
	if err != nil {
		return models.Session{}, fmt.Errorf("get session %s: %w", id, err)
	}
	s.FindingSummary = finding
	return s, nil
}

// ListSessions returns metadata for sessions matching the filters, newest
// update first.
func (d *DB) ListSessions(f SessionFilters) ([]models.SessionListItem, error) {
	where := []string{}
	args := []any{}
	if f.Domain != "" {
		where = append(where, "domain = ?")
		args = append(args, f.Domain)
	}
	if f.Phase != "" {
		where = append(where, "phase = ?")
		args = append(args, f.Phase)
	}
	if f.Tag != "" {
		// tags is a JSON string[]; exact single-tag match via LIKE on the
		// quoted element keeps this SQL-only per DATA_MODEL.md.
		where = append(where, "tags LIKE ?")
		args = append(args, `%"`+f.Tag+`"%`)
	}
	if f.HasPsd != nil {
		if *f.HasPsd {
			where = append(where, "psd IS NOT NULL")
		} else {
			where = append(where, "psd IS NULL")
		}
	}
	if f.Distilled != nil {
		if *f.Distilled {
			where = append(where, "raw_source IS NOT NULL")
		} else {
			where = append(where, "raw_source IS NULL")
		}
	}
	q := `SELECT id, project_name, domain, tags, created_at, updated_at, phase,
	             raw_source, pipeline_plan, agent_outputs, psd, finding_summary
	      FROM sessions`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY updated_at DESC"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var out []models.SessionListItem
	for rows.Next() {
		var it models.SessionListItem
		var domain, tags, agentOutputs, findingSummary string
		var rawSource, planRaw, psd sql.NullString
		if err := rows.Scan(&it.ID, &it.ProjectName, &domain, &tags, &it.CreatedAt, &it.UpdatedAt,
			&it.Phase, &rawSource, &planRaw, &agentOutputs, &psd, &findingSummary); err != nil {
			return nil, fmt.Errorf("list sessions scan: %w", err)
		}
		it.Domain = domain
		it.Tags = lenientStringArray(tags)
		it.Distilled = rawSource.Valid
		it.AgentCount = planAgentCount(planRaw)
		it.HasPsd = psd.Valid && psd.String != ""
		it.FindingSummary = lenientFindingSummary(findingSummary)
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions rows: %w", err)
	}
	return out, nil
}

// SessionPatch is a partial update. raw_source is intentionally absent: it is
// set only by the intake path, because its NULL state is a record of which
// path the session took.
type SessionPatch struct {
	ProjectName  *string
	Phase        *models.SessionPhase
	RawInput     *string
	PipelinePlan *models.PipelinePlan  // also syncs domain/tags columns
	AgentOutputs *[]models.AgentOutput // also refreshes finding_summary
	Psd          *string
	Tags         *[]string
	Domain       *string
}

// UpdateSession applies the patch and touches updated_at.
func (d *DB) UpdateSession(id string, p SessionPatch) error {
	sets := []string{}
	args := []any{}
	if p.ProjectName != nil {
		sets = append(sets, "project_name = ?")
		args = append(args, *p.ProjectName)
	}
	if p.Phase != nil {
		sets = append(sets, "phase = ?")
		args = append(args, string(*p.Phase))
	}
	if p.RawInput != nil {
		sets = append(sets, "raw_input = ?")
		args = append(args, *p.RawInput)
	}
	if p.PipelinePlan != nil {
		raw, err := json.Marshal(p.PipelinePlan)
		if err != nil {
			return fmt.Errorf("update session plan: %w", err)
		}
		sets = append(sets, "pipeline_plan = ?", "domain = ?", "tags = ?")
		args = append(args, string(raw), p.PipelinePlan.Context.Domain, marshalStringArray(p.PipelinePlan.Context.Tags))
	}
	if p.AgentOutputs != nil {
		raw, err := json.Marshal(p.AgentOutputs)
		if err != nil {
			return fmt.Errorf("update session outputs: %w", err)
		}
		sets = append(sets, "agent_outputs = ?", "finding_summary = ?")
		args = append(args, string(raw), marshalFindingSummary(models.ComputeFindingSummary(*p.AgentOutputs)))
	}
	if p.Psd != nil {
		sets = append(sets, "psd = ?")
		args = append(args, *p.Psd)
	}
	if p.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, marshalStringArray(*p.Tags))
	}
	if p.Domain != nil {
		sets = append(sets, "domain = ?")
		args = append(args, *p.Domain)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, NowUTC(), id)
	_, err := d.Exec("UPDATE sessions SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	return nil
}

// DeleteSession removes the session row. agent_outputs, plan and PSD live in
// this row, so deleting it removes all associated data.
func (d *DB) DeleteSession(id string) error {
	res, err := d.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetRawSource writes raw_source. This is the ONLY writer of the column, used
// by the intake path: its NULL state is the record of which path the session
// took (skipped vs distilled), so nothing else may touch it.
func (d *DB) SetRawSource(id, raw string) error {
	res, err := d.Exec(`UPDATE sessions SET raw_source = ?, updated_at = ? WHERE id = ?`, raw, NowUTC(), id)
	if err != nil {
		return fmt.Errorf("set raw source: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ClearRawSource resets raw_source to NULL — the revert path. NULL is the
// record of a skipped or undone distillation; it must never degrade to an
// empty string.
func (d *DB) ClearRawSource(id string) error {
	res, err := d.Exec(`UPDATE sessions SET raw_source = NULL, updated_at = ? WHERE id = ?`, NowUTC(), id)
	if err != nil {
		return fmt.Errorf("clear raw source: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetSessionPhase writes the phase and touches updated_at. Called before
// every LLM call so state is recoverable.
func (d *DB) SetSessionPhase(id string, phase models.SessionPhase) error {
	_, err := d.Exec(`UPDATE sessions SET phase = ?, updated_at = ? WHERE id = ?`, string(phase), NowUTC(), id)
	if err != nil {
		return fmt.Errorf("set session phase: %w", err)
	}
	return nil
}

// SavePlan writes the pipeline plan and syncs the domain/tags columns that
// Shishō assigns in Phase 0 for fast SQL filtering.
func (d *DB) SavePlan(id string, plan *models.PipelinePlan) error {
	return d.UpdateSession(id, SessionPatch{PipelinePlan: plan})
}

// SavePSD writes the PSD markdown and recomputes the finding-severity summary
// from the stored outputs.
func (d *DB) SavePSD(id string, psd string) error {
	s, err := d.GetSession(id)
	if err != nil {
		return err
	}
	return d.UpdateSession(id, SessionPatch{Psd: &psd, AgentOutputs: &s.AgentOutputs})
}

// SetNodeStatus updates one node's status inside the persisted plan. Used to
// mark a node `running` before its LLM call, so an interruption leaves the
// node recoverable.
func (d *DB) SetNodeStatus(sessionID, nodeID string, status models.AgentStatus) error {
	s, err := d.GetSession(sessionID)
	if err != nil {
		return err
	}
	if s.PipelinePlan == nil {
		return fmt.Errorf("set node status %s/%s: no pipeline plan", sessionID, nodeID)
	}
	node, _ := s.PipelinePlan.FindNode(nodeID)
	if node == nil {
		return fmt.Errorf("set node status: node %s not found in session %s", nodeID, sessionID)
	}
	node.Status = status
	return d.SavePlan(sessionID, s.PipelinePlan)
}

// AppendAgentOutput persists one agent result together with its node status
// in the SAME transaction, at the moment it completes. Never buffered to
// flush at the end of a run — a run that dies with five finished specialists
// must leave five finished specialists in the database.
func (d *DB) AppendAgentOutput(sessionID string, out models.AgentOutput, nodeID string, nodeStatus models.AgentStatus) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("append agent output begin: %w", err)
	}
	defer tx.Rollback() // no-op after commit

	var planRaw sql.NullString
	var outputsRaw string
	err = tx.QueryRow(
		`SELECT pipeline_plan, agent_outputs FROM sessions WHERE id = ?`, sessionID,
	).Scan(&planRaw, &outputsRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("append agent output select: %w", err)
	}

	var plan models.PipelinePlan
	if planRaw.Valid && planRaw.String != "" {
		if err := json.Unmarshal([]byte(planRaw.String), &plan); err != nil {
			return fmt.Errorf("append agent output plan %s: %w", sessionID, err)
		}
	}
	node, _ := plan.FindNode(nodeID)
	if node == nil {
		return fmt.Errorf("append agent output: node %s not found in session %s", nodeID, sessionID)
	}
	node.Status = nodeStatus

	outputs, err := parseOutputs(outputsRaw)
	if err != nil {
		return fmt.Errorf("append agent output parse outputs %s: %w", sessionID, err)
	}
	outputs = append(outputs, out)

	planJSON, err := json.Marshal(&plan)
	if err != nil {
		return fmt.Errorf("append agent output marshal plan: %w", err)
	}
	outputsJSON, err := json.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("append agent output marshal outputs: %w", err)
	}
	summaryJSON := marshalFindingSummary(models.ComputeFindingSummary(outputs))

	if _, err := tx.Exec(
		`UPDATE sessions SET pipeline_plan = ?, agent_outputs = ?, finding_summary = ?, updated_at = ? WHERE id = ?`,
		string(planJSON), string(outputsJSON), summaryJSON, NowUTC(), sessionID,
	); err != nil {
		return fmt.Errorf("append agent output update: %w", err)
	}
	return tx.Commit()
}
