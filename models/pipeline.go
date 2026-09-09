package models

type Wave int
type AgentStatus string
type InputType string
type ProjectType string
type Maturity string
type BlockStatus string

const (
	Wave1 Wave = 1
	Wave2 Wave = 2
	Fixed Wave = 3
)

const (
	StatusPending AgentStatus = "pending"
	StatusRunning AgentStatus = "running"
	StatusDone    AgentStatus = "done"
	StatusSkipped AgentStatus = "skipped"
	StatusError   AgentStatus = "error"
)

const (
	InputRawIdea    InputType = "raw_idea"
	InputBrief      InputType = "brief"
	InputTranscript InputType = "transcript"
	InputPSD        InputType = "psd"
)

const (
	TypeSoftware ProjectType = "software"
	TypeGame     ProjectType = "game"
	TypeSaaS     ProjectType = "saas"
	TypeTool     ProjectType = "tool"
	TypeHybrid   ProjectType = "hybrid"
)

const (
	MaturityRaw     Maturity = "raw"
	MaturityPartial Maturity = "partial"
	MaturitySpec    Maturity = "spec"
)

const (
	BlockPresent BlockStatus = "present"
	BlockPartial BlockStatus = "partial"
	BlockMissing BlockStatus = "missing"
)

type AgentNode struct {
	ID          string      `json:"id"`
	AgentID     string      `json:"agent_id"`
	DisplayName string      `json:"display_name"`
	RoleSummary string      `json:"role_summary"`
	SourceURL   string      `json:"source_url"`
	Wave        Wave        `json:"wave"`
	Status      AgentStatus `json:"status"`
	Enabled     bool        `json:"enabled"`
	Rationale   string      `json:"rationale"`
}

type FourBlock struct {
	WhatIsWanted         BlockStatus `json:"what_is_wanted"`
	HowItShouldBeDone    BlockStatus `json:"how_it_should_be_done"`
	WhatIsNotWanted      BlockStatus `json:"what_is_not_wanted"`
	HowSuccessIsMeasured BlockStatus `json:"how_success_is_measured"`
}

type PipelineContext struct {
	ProblemStatement string    `json:"problem_statement"`
	Maturity         Maturity  `json:"maturity"`
	FourBlock        FourBlock `json:"four_block"`
	Flags            []string  `json:"flags"`
	Domain           string    `json:"domain"`
	Tags             []string  `json:"tags"`
	Distilled        bool      `json:"distilled"` // input came through Uchikomi
}

type PipelineWaves struct {
	Wave1 []AgentNode `json:"wave1"`
	Wave2 []AgentNode `json:"wave2"`
	Fixed []AgentNode `json:"fixed"`
}

type PipelinePlan struct {
	SessionID   string          `json:"session_id"`
	ProjectName string          `json:"project_name"`
	InputType   InputType       `json:"input_type"`
	ProjectType ProjectType     `json:"project_type"`
	Pipeline    PipelineWaves   `json:"pipeline"`
	Context     PipelineContext `json:"context"`
	// Runner policy (client-set in the builder, not a Shishō output field):
	// pause the run when a specialist fails, instead of continuing past it.
	// Persists on the plan so a resume keeps the same policy.
	PauseOnFail bool `json:"pause_on_fail"`
}

// AllNodes returns the plan's nodes in execution order: Wave 1, Wave 2, Fixed.
func (p *PipelinePlan) AllNodes() []AgentNode {
	out := make([]AgentNode, 0, len(p.Pipeline.Wave1)+len(p.Pipeline.Wave2)+len(p.Pipeline.Fixed))
	out = append(out, p.Pipeline.Wave1...)
	out = append(out, p.Pipeline.Wave2...)
	out = append(out, p.Pipeline.Fixed...)
	return out
}

// FindNode returns the node with the given node ID and its index within its wave slice.
func (p *PipelinePlan) FindNode(nodeID string) (*AgentNode, int) {
	waves := []*[]AgentNode{&p.Pipeline.Wave1, &p.Pipeline.Wave2, &p.Pipeline.Fixed}
	for _, w := range waves {
		for i := range *w {
			if (*w)[i].ID == nodeID {
				return &(*w)[i], i
			}
		}
	}
	return nil, -1
}
