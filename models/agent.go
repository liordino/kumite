package models

type FindingType string
type Severity string

const (
	FindingOpportunity FindingType = "opportunity"
	FindingRisk        FindingType = "risk"
	FindingQuestion    FindingType = "question"
	FindingConstraint  FindingType = "constraint"
)

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Finding struct {
	Type     FindingType `json:"type"`
	Severity Severity    `json:"severity"`
	Title    string      `json:"title"`
	Body     string      `json:"body"`
}

type PsdContribution struct {
	Section string `json:"section"` // exact PSD section heading, see PSD Sections below
	Content string `json:"content"`
}

type AgentOutputData struct {
	Summary          string            `json:"summary"`
	Findings         []Finding         `json:"findings"`
	Recommendation   string            `json:"recommendation"`
	OpenQuestions    []string          `json:"open_questions"`
	PsdContributions []PsdContribution `json:"psd_contributions"`
}

type AgentOutput struct {
	AgentID     string          `json:"agent_id"`
	DisplayName string          `json:"display_name"`
	Wave        int             `json:"wave"`
	Status      string          `json:"status"` // "done" | "error" | "partial"
	Output      AgentOutputData `json:"output"`
	Thinking    string          `json:"thinking"`
	Error       string          `json:"error,omitempty"`
}

// RosterEntry defines a built-in agent from agents.json
type RosterEntry struct {
	AgentID     string `json:"agent_id"`
	DisplayName string `json:"display_name"`
	Division    string `json:"division"`
	SourceURL   string `json:"source_url"`
	DefaultWave int    `json:"default_wave"`
	Conditional bool   `json:"conditional"`
	Condition   string `json:"condition,omitempty"` // e.g. "project_type:game"
	Fixed       bool   `json:"fixed"`
}

// CustomAgent is a user-defined specialist stored in the custom_agents table.
type CustomAgent struct {
	ID             string `json:"id"`
	DisplayName    string `json:"display_name"`
	RoleSummary    string `json:"role_summary"`
	WavePreference int    `json:"wave_preference"` // 1 or 2
	SystemPrompt   string `json:"system_prompt"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
