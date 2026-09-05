package models

type SessionPhase string

const (
	PhaseIntake         SessionPhase = "intake"          // input submitted, intake choice pending
	PhaseDistilling     SessionPhase = "distilling"      // Uchikomi running
	PhasePipelineReview SessionPhase = "pipeline_review" // Phase 0 done, awaiting confirmation
	PhaseRunning        SessionPhase = "running"
	PhaseSynthesis      SessionPhase = "synthesis"
	PhaseInterrupted    SessionPhase = "interrupted" // run cut mid-flight, resumable
	PhaseComplete       SessionPhase = "complete"
	PhaseError          SessionPhase = "error"
)

type FindingSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

type Session struct {
	ID             string         `json:"id"`
	ProjectName    string         `json:"project_name"`
	Domain         string         `json:"domain"`
	Tags           []string       `json:"tags"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	Phase          SessionPhase   `json:"phase"`
	RawSource      string         `json:"raw_source"` // "" when intake was skipped
	RawInput       string         `json:"raw_input"`
	PipelinePlan   *PipelinePlan  `json:"pipeline_plan"`
	AgentOutputs   []AgentOutput  `json:"agent_outputs"`
	Psd            string         `json:"psd"`
	FindingSummary FindingSummary `json:"finding_summary"`
}

type SessionListItem struct {
	ID             string         `json:"id"`
	ProjectName    string         `json:"project_name"`
	Domain         string         `json:"domain"`
	Tags           []string       `json:"tags"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	Phase          SessionPhase   `json:"phase"`
	Distilled      bool           `json:"distilled"`
	AgentCount     int            `json:"agent_count"`
	HasPsd         bool           `json:"has_psd"`
	FindingSummary FindingSummary `json:"finding_summary"`
}

// ComputeFindingSummary counts findings by severity across all agent outputs.
func ComputeFindingSummary(outputs []AgentOutput) FindingSummary {
	sum := FindingSummary{}
	for _, o := range outputs {
		for _, f := range o.Output.Findings {
			switch f.Severity {
			case SeverityCritical:
				sum.Critical++
			case SeverityHigh:
				sum.High++
			case SeverityMedium:
				sum.Medium++
			case SeverityLow:
				sum.Low++
			}
		}
	}
	return sum
}
