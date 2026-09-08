package engine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kumite/db"
	"kumite/models"
)

// runner.go executes the pipeline: strictly sequential, strictly wave-ordered,
// persisting before every LLM call and at every completion. No goroutines for
// wave execution — waves express dependency, not concurrency.

// RunDeps carries everything the runner needs. No global state.
type RunDeps struct {
	DB       *db.DB
	LLM      *Client
	HTTP     *http.Client // GitHub fetches
	CacheTTL time.Duration
	Roster   []models.RosterEntry

	// LLMOverride, when set (LLM_MOCK=true), replaces the app_config-derived
	// provider settings for every call in the run.
	LLMOverride *RuntimeLLMConfig
}

// llmConfig resolves the provider settings per call: the mock override when
// active, otherwise the runtime values from app_config — so a provider change
// through the settings panel takes effect on the next LLM call without restart.
func (deps RunDeps) llmConfig() (RuntimeLLMConfig, error) {
	if deps.LLMOverride != nil {
		return *deps.LLMOverride, nil
	}
	rt, err := deps.DB.LoadRuntimeLLM()
	if err != nil {
		return RuntimeLLMConfig{}, err
	}
	return RuntimeLLMConfig{Endpoint: rt.Endpoint, Model: rt.Model, APIKey: rt.APIKey}, nil
}

// EmitFunc receives pipeline lifecycle events (the SSE payloads). The runner
// is agnostic to the transport — handlers own the stream.
type EmitFunc func(event string, payload any)

// RunPipeline walks the persisted plan in wave order, skipping nodes already
// done or skipped, executing the rest. It is used by both the run and the
// resume endpoints; resume is driven purely by node status. On entry the
// session's plan is the record; on exit every completed specialist is in the
// database.
func RunPipeline(ctx context.Context, deps RunDeps, session models.Session, emit EmitFunc) error {
	if session.PipelinePlan == nil {
		return fmt.Errorf("run pipeline %s: no pipeline plan", session.ID)
	}
	template, err := LoadTemplate()
	if err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}

	plan := session.PipelinePlan
	emit("pipeline_start", map[string]any{
		"session_id":   session.ID,
		"total_agents": RemainingAgents(plan),
	})

	// Persist before every LLM call — the phase flip included.
	if err := deps.DB.SetSessionPhase(session.ID, models.PhaseRunning); err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}

	waves := []struct {
		wave  models.Wave
		nodes []models.AgentNode
	}{
		{models.Wave1, plan.Pipeline.Wave1},
		{models.Wave2, plan.Pipeline.Wave2},
		{models.Fixed, plan.Pipeline.Fixed},
	}

	for _, w := range waves {
		for i := range w.nodes {
			node := &w.nodes[i]
			if !node.Enabled {
				// Disabled agents are skipped but their node stays in the
				// pipeline state with status skipped.
				if node.Status != models.StatusSkipped {
					node.Status = models.StatusSkipped
					if err := deps.DB.SetNodeStatus(session.ID, node.ID, models.StatusSkipped); err != nil {
						return fmt.Errorf("run pipeline %s: %w", session.ID, err)
					}
				}
				continue
			}
			if node.Status == models.StatusDone || node.Status == models.StatusSkipped {
				continue // resume: driven by node status, no stored cursor
			}

			if err := runAgent(ctx, deps, template, session.ID, node, emit); err != nil {
				return fmt.Errorf("run pipeline %s: %w", session.ID, err)
			}
		}
		emit("wave_complete", map[string]any{"wave": int(w.wave)})
	}

	// Synthesis. Phase 2 failure lands the session in interrupted — every
	// specialist output is intact and synthesis is retriable alone.
	if err := deps.DB.SetSessionPhase(session.ID, models.PhaseSynthesis); err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}
	emit("synthesis_start", map[string]any{})

	sess, err := deps.DB.GetSession(session.ID)
	if err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}
	runCfg, err := deps.llmConfig()
	if err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}
	psd, err := RunPhase2(ctx, deps.LLM, runCfg, sess, func(delta string) {
		emit("psd_chunk", map[string]any{"chunk": delta})
	})
	if err != nil {
		// Phase 2 failure lands the session in interrupted — every specialist
		// output is intact and synthesis is retriable alone. The caller emits
		// pipeline_error; the runner owns persistence, the caller owns transport.
		if phaseErr := deps.DB.SetSessionPhase(session.ID, models.PhaseInterrupted); phaseErr != nil {
			return fmt.Errorf("run pipeline %s: %w (original: %v)", session.ID, phaseErr, err)
		}
		return err
	}

	if err := deps.DB.SavePSD(session.ID, psd); err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}
	if err := deps.DB.SetSessionPhase(session.ID, models.PhaseComplete); err != nil {
		return fmt.Errorf("run pipeline %s: %w", session.ID, err)
	}
	emit("psd_done", map[string]any{"psd": psd})
	emit("pipeline_complete", map[string]any{"session_id": session.ID})
	return nil
}

// runAgent executes one specialist: mark running (persisted BEFORE the LLM
// call), fetch + assemble the prompt, stream the call, persist the result
// together with its node status in the same transaction. Specialist failures
// never abort the run.
func runAgent(ctx context.Context, deps RunDeps, template string, sessionID string, node *models.AgentNode, emit EmitFunc) error {
	cfg, err := deps.llmConfig()
	if err != nil {
		return err
	}
	if err := deps.DB.SetNodeStatus(sessionID, node.ID, models.StatusRunning); err != nil {
		return err
	}
	emit("agent_start", map[string]any{
		"agent_id":     node.AgentID,
		"display_name": node.DisplayName,
		"wave":         int(node.Wave),
	})

	fail := func(err error) error {
		errored := models.AgentOutput{
			AgentID:     node.AgentID,
			DisplayName: node.DisplayName,
			Wave:        int(node.Wave),
			Status:      "error",
			Error:       err.Error(),
			Output: models.AgentOutputData{
				Findings:      []models.Finding{},
				OpenQuestions: []string{},
			},
		}
		if appendErr := deps.DB.AppendAgentOutput(sessionID, errored, node.ID, models.StatusError); appendErr != nil {
			return fmt.Errorf("persist agent error %s: %w (original: %v)", node.AgentID, appendErr, err)
		}
		emit("agent_error", map[string]any{"agent_id": node.AgentID, "error": err.Error()})
		return nil
	}

	systemPrompt, _, _, err := FetchAgentPrompt(ctx, deps.DB, deps.HTTP, node.AgentID, node.SourceURL, deps.CacheTTL)
	if err != nil {
		return fail(err)
	}

	// Prior outputs for the Wave 2 context are read fresh from the database —
	// the persisted state is the record; nothing accumulates in memory.
	sess, err := deps.DB.GetSession(sessionID)
	if err != nil {
		return fail(err)
	}
	prompt, err := AssemblePrompt(template, systemPrompt, *node, *sess.PipelinePlan, sess.AgentOutputs, sess.RawInput)
	if err != nil {
		return fail(err)
	}

	var out models.AgentOutput
	res, err := deps.LLM.JSON(ctx, cfg, ChatRequest{
		System: prompt,
		Stream: true,
		Format: AgentOutputSchema(),
	}, &out, Hooks{
		OnThinking: func(chunk string) {
			emit("agent_thinking", map[string]any{"agent_id": node.AgentID, "chunk": chunk})
		},
	})
	if err != nil {
		return fail(err)
	}

	// The envelope's thinking field is a placeholder the model never writes
	// into — the engine populates it from the extracted block.
	out.AgentID = node.AgentID
	out.DisplayName = node.DisplayName
	out.Wave = int(node.Wave)
	out.Thinking = res.Thinking
	out.Status = "done"
	if res.Partial {
		out.Status = "partial"
	}
	if out.Output.Findings == nil {
		out.Output.Findings = []models.Finding{}
	}
	if out.Output.OpenQuestions == nil {
		out.Output.OpenQuestions = []string{}
	}
	if out.Output.PsdContributions == nil {
		out.Output.PsdContributions = []models.PsdContribution{}
	}

	status := models.StatusDone
	if err := deps.DB.AppendAgentOutput(sessionID, out, node.ID, status); err != nil {
		return fmt.Errorf("persist agent %s: %w", node.AgentID, err)
	}
	emit("agent_done", map[string]any{"agent_id": node.AgentID, "output": out})
	return nil
}

// RemainingAgents counts enabled nodes not yet done or skipped — the total
// reported in pipeline_start (and the `remaining` count for resume).
func RemainingAgents(plan *models.PipelinePlan) int {
	n := 0
	for _, node := range plan.AllNodes() {
		if node.Enabled && node.Status != models.StatusDone && node.Status != models.StatusSkipped {
			n++
		}
	}
	return n
}

// SkippedAgentIDs lists enabled=false node ids — what resume reports it is
// skipping.
func SkippedAgentIDs(plan *models.PipelinePlan) []string {
	var out []string
	for _, node := range plan.AllNodes() {
		if !node.Enabled || node.Status == models.StatusDone || node.Status == models.StatusSkipped {
			out = append(out, node.AgentID)
		}
	}
	return out
}

// ensureHTTPClient returns a client for GitHub fetches with a sane default.
func ensureHTTPClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: 30 * time.Second}
}
