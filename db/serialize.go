package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"kumite/models"
)

// JSON column serialization lives inside the db package per DATA_MODEL.md —
// never in handlers.

// parseStringArray decodes a tags JSON column.
func parseStringArray(s string) ([]string, error) {
	if s == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse tags: %w", err)
	}
	return out, nil
}

func marshalStringArray(a []string) string {
	if a == nil {
		a = []string{}
	}
	b, err := json.Marshal(a)
	if err != nil {
		// Marshaling a []string cannot fail in practice; the fallback keeps
		// the column valid JSON regardless.
		return "[]"
	}
	return string(b)
}

// parseOutputs decodes an agent_outputs JSON column. Strict: a corrupted
// outputs column must surface as an error, not silently read as empty — the
// durability guarantee depends on what was actually persisted.
func parseOutputs(s string) ([]models.AgentOutput, error) {
	if s == "" {
		return []models.AgentOutput{}, nil
	}
	var out []models.AgentOutput
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse agent outputs: %w", err)
	}
	return out, nil
}

// parseFindingSummary decodes a finding_summary JSON column.
func parseFindingSummary(s string) (models.FindingSummary, error) {
	var fs models.FindingSummary
	if s == "" {
		return fs, nil
	}
	if err := json.Unmarshal([]byte(s), &fs); err != nil {
		return fs, fmt.Errorf("parse finding summary: %w", err)
	}
	return fs, nil
}

func marshalFindingSummary(fs models.FindingSummary) string {
	b, err := json.Marshal(fs)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// planAgentCount returns the number of nodes in a stored plan for list
// metadata. Lenient by design: list rendering must not fail on a plan that
// fails to parse; the strict read path is GetSession.
func planAgentCount(planRaw sql.NullString) int {
	if !planRaw.Valid || planRaw.String == "" {
		return 0
	}
	var p models.PipelinePlan
	if err := json.Unmarshal([]byte(planRaw.String), &p); err != nil {
		return 0
	}
	return len(p.AllNodes())
}

// lenientStringArray / lenientFindingSummary are the list-metadata variants:
// same shapes, corruption-tolerant defaults.
func lenientStringArray(s string) []string {
	out, err := parseStringArray(s)
	if err != nil {
		return []string{}
	}
	return out
}

func lenientFindingSummary(s string) models.FindingSummary {
	fs, err := parseFindingSummary(s)
	if err != nil {
		return models.FindingSummary{}
	}
	return fs
}
