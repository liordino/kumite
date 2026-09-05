# Data Model — Kumite

## SQLite Tables

### sessions

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    project_name    TEXT NOT NULL DEFAULT 'Untitled Project',
    domain          TEXT NOT NULL DEFAULT '',        -- "game"|"saas"|"tool"|"software"|"hybrid"
    tags            TEXT NOT NULL DEFAULT '[]',      -- JSON string[]
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    phase           TEXT NOT NULL DEFAULT 'intake',  -- see SessionPhase
    raw_source      TEXT,                            -- original material | NULL if intake skipped
    raw_input       TEXT NOT NULL DEFAULT '',        -- what Shishō consumes
    pipeline_plan   TEXT,                            -- JSON PipelinePlan | NULL
    agent_outputs   TEXT NOT NULL DEFAULT '[]',      -- JSON AgentOutput[]
    psd             TEXT,                            -- markdown string | NULL
    finding_summary TEXT NOT NULL DEFAULT '{}'       -- JSON {"critical":0,"high":0,...}
);

CREATE INDEX idx_sessions_updated ON sessions(updated_at DESC);
CREATE INDEX idx_sessions_domain   ON sessions(domain);
CREATE INDEX idx_sessions_phase    ON sessions(phase);
```

**On `raw_source` and `raw_input`.** `raw_input` is always what Shishō consumes. On session creation the submitted material goes there directly. If the user runs Uchikomi, the original is copied to `raw_source` and the distilled brief replaces `raw_input`.

`raw_source` is NULL exactly when intake was skipped. That null is load-bearing — it is the record of which path the session took. When a PSD comes out wrong, distinguishing a panel failure from a compression failure requires both versions. Do not backfill it with a copy of `raw_input` for tidiness.

### agent_cache

```sql
CREATE TABLE IF NOT EXISTS agent_cache (
    agent_id    TEXT PRIMARY KEY,
    source_url  TEXT NOT NULL,
    content     TEXT NOT NULL,
    fetched_at  TEXT NOT NULL,
    etag        TEXT
);
```

### custom_agents

```sql
CREATE TABLE IF NOT EXISTS custom_agents (
    id               TEXT PRIMARY KEY,
    display_name     TEXT NOT NULL,
    role_summary     TEXT NOT NULL,
    wave_preference  INTEGER NOT NULL DEFAULT 1,   -- 1 or 2
    system_prompt    TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);
```

### app_config

```sql
CREATE TABLE IF NOT EXISTS app_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Seeded defaults (Ollama Cloud). Override any of these via the settings panel.
INSERT OR IGNORE INTO app_config VALUES ('llm_endpoint', 'https://ollama.com/v1');
INSERT OR IGNORE INTO app_config VALUES ('llm_model',    'qwen3:32b');
INSERT OR IGNORE INTO app_config VALUES ('llm_api_key',  '');
```

The `pipeline_plan`, `agent_outputs`, `tags`, and `finding_summary` columns are JSON strings. Serialize and deserialize inside `db/` helpers, never in handlers.

> Endpoint and model defaults assume Ollama Cloud. Confirm the exact endpoint path and model slug against Ollama Cloud's current docs when wiring `engine/llm.go` — substitute the real values. The settings panel lets the user change all three at runtime.

---

## Go Types

### models/pipeline.go

```go
package models

type Wave        int
type AgentStatus string
type InputType   string
type ProjectType string
type Maturity    string
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
}
```

`Distilled` is set by the engine from `raw_source != NULL`, not by the model. It is passed to specialists in their prompt context so they know whether they are reading the author's own words or a compression of them.

### models/agent.go

```go
package models

type FindingType string
type Severity    string

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
```

### models/session.go

```go
package models

type SessionPhase string

const (
    PhaseIntake         SessionPhase = "intake"          // input submitted, intake choice pending
    PhaseDistilling     SessionPhase = "distilling"      // Uchikomi running
    PhasePipelineReview SessionPhase = "pipeline_review" // Phase 0 done, awaiting confirmation
    PhaseRunning        SessionPhase = "running"
    PhaseSynthesis      SessionPhase = "synthesis"
    PhaseInterrupted    SessionPhase = "interrupted"    // run cut mid-flight, resumable
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
```

`PhaseDistilling` is transient — a session is in it only while `POST /api/intake/:id` is in flight. On failure it returns to `PhaseIntake`, not `PhaseError`, because nothing downstream has started and the user can simply proceed without distillation.

`PhaseInterrupted` means a run was cut mid-flight and can be resumed. It is distinct from `PhaseError`: error is terminal and requires user action, interrupted is a recoverable state with valid partial results already persisted. Sessions enter it two ways — startup reconciliation (see `ARCHITECTURE.md`) or an aborting failure during a run.

**Resume is driven by node status, not by a cursor.** Each `AgentNode` in the persisted `pipeline_plan` carries its own `status`. Resuming means: skip every node with `done` or `skipped`, re-run every node with `pending` or `running`, in the usual wave order. A node left at `running` is one whose output was never persisted, so re-running it is safe and never duplicates an output. Do not store a "current agent index" — it would be a second source of truth that can disagree with the node statuses.

When every node is `done` or `skipped` and `psd` is still null, resume skips the panel entirely and re-runs synthesis.

---

## PSD Sections

The PSD has ten fixed sections. `PsdContribution.Section` must match one of these strings exactly. Shishō assembles contributions by section; unmatched section names are dropped with a warning rather than creating a new section.

```
1. Overview
2. Core Problem & Value Proposition
3. Feasibility Challenge
4. Minimum Viable Slice
5. Features by Dependency
6. Versioning & Switches
7. Constraints & Scope Boundaries
8. Assumptions, Gaps & Contradictions
9. Undecided Candidates
10. Questions for the Next Session
```

Section 9 is populated from Uchikomi's model-proposed list and is empty when intake was skipped. Specialists do not contribute to it.

Section 4 currently has no roster agent with an explicit mandate — see the open decision in `CONTEXT.md`.

---

## roster/agents.json

```json
[
  {
    "agent_id": "engineering-software-architect",
    "display_name": "Software Architect",
    "division": "engineering",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/engineering/engineering-software-architect.md",
    "default_wave": 1,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "engineering-security-engineer",
    "display_name": "Security Engineer",
    "division": "engineering",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/engineering/engineering-security-engineer.md",
    "default_wave": 2,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "engineering-data-engineer",
    "display_name": "Data Engineer",
    "division": "engineering",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/engineering/engineering-data-engineer.md",
    "default_wave": 1,
    "conditional": true,
    "condition": "project_type:software,saas,hybrid",
    "fixed": false
  },
  {
    "agent_id": "design-ux-researcher",
    "display_name": "UX Researcher",
    "division": "design",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/design/design-ux-researcher.md",
    "default_wave": 1,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "product-trend-researcher",
    "display_name": "Trend Researcher",
    "division": "product",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/product/product-trend-researcher.md",
    "default_wave": 1,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "product-manager",
    "display_name": "Product Manager",
    "division": "product",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/product/product-manager.md",
    "default_wave": 1,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "project-manager-senior",
    "display_name": "Senior Project Manager",
    "division": "project-management",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/project-management/project-manager-senior.md",
    "default_wave": 2,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "support-legal-compliance-checker",
    "display_name": "Legal Compliance Checker",
    "division": "support",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/support/support-legal-compliance-checker.md",
    "default_wave": 2,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "testing-tool-evaluator",
    "display_name": "Tool Evaluator",
    "division": "testing",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/testing/testing-tool-evaluator.md",
    "default_wave": 2,
    "conditional": true,
    "condition": "has_tooling_decisions",
    "fixed": false
  },
  {
    "agent_id": "finance-financial-analyst",
    "display_name": "Financial Analyst",
    "division": "finance",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/finance/finance-financial-analyst.md",
    "default_wave": 1,
    "conditional": false,
    "fixed": false
  },
  {
    "agent_id": "game-designer",
    "display_name": "Game Designer",
    "division": "game-development",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/game-development/game-designer.md",
    "default_wave": 1,
    "conditional": true,
    "condition": "project_type:game,hybrid",
    "fixed": false
  },
  {
    "agent_id": "narrative-designer",
    "display_name": "Narrative Designer",
    "division": "game-development",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/game-development/narrative-designer.md",
    "default_wave": 1,
    "conditional": true,
    "condition": "project_type:game,hybrid+has_narrative",
    "fixed": false
  },
  {
    "agent_id": "testing-reality-checker",
    "display_name": "Reality Checker",
    "division": "testing",
    "source_url": "https://raw.githubusercontent.com/msitarzewski/agency-agents/main/testing/testing-reality-checker.md",
    "default_wave": 3,
    "conditional": false,
    "fixed": true
  }
]
```

> Verify every `source_url` resolves before the first real run. The upstream repo may reorganize paths; a 404 falls through to the fetcher's stale-cache path, which has nothing cached on first use.
