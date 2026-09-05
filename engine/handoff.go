package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"kumite/models"
	"kumite/prompts"
)

// handoff.go produces the Dojo bundle: a third, separate Shishō invocation
// (never mixed with Phase 0 or Phase 2). It translates the feasibility
// verdict into the shape Dojo's /hajime expects — a translation, not a new
// analysis: nothing appears in the bundle that is not already in the PSD or
// the panel outputs.

// HandoffBundle holds the two artifacts POST /api/handoff/:sessionId returns.
type HandoffBundle struct {
	BriefMD   string
	ContextMD string
}

// GenerateHandoff requires a session with a PSD (the verdict is the input).
// Plain markdown — no schema constraint. The engine defines the invocation
// and delimiters; the prompt artifact defines the content contract.
func GenerateHandoff(ctx context.Context, client *Client, cfg RuntimeLLMConfig, session models.Session) (*HandoffBundle, error) {
	if strings.TrimSpace(session.Psd) == "" {
		return nil, fmt.Errorf("handoff: session has no PSD — run the panel first")
	}
	prompt, err := prompts.Shisho()
	if err != nil {
		return nil, fmt.Errorf("handoff load prompt: %w", err)
	}

	// Phase 2's {{AGENT_OUTPUTS}} token would otherwise stay unresolved in
	// the shared prompt file; the handoff call reuses the same injection so
	// the model can draw on panel findings without re-analyzing anything.
	injections := make([]map[string]any, 0, len(session.AgentOutputs))
	for _, o := range session.AgentOutputs {
		injections = append(injections, map[string]any{
			"id":             o.AgentID,
			"display_name":   o.DisplayName,
			"status":         o.Status,
			"summary":        o.Output.Summary,
			"findings":       o.Output.Findings,
			"recommendation": o.Output.Recommendation,
		})
	}
	outputsJSON, err := json.Marshal(injections)
	if err != nil {
		return nil, fmt.Errorf("handoff outputs marshal: %w", err)
	}

	system := strings.ReplaceAll(string(prompt), "{{AGENT_OUTPUTS}}", string(outputsJSON))
	system += "\n\n---\n\nINVOCATION: Handoff — produce the Dojo bundle. Emit exactly two " +
		"documents and nothing else: first BRIEF.md wrapped in <BRIEF_DOCUMENT> tags, " +
		"then the CONTEXT.md seed wrapped in <CONTEXT_DOCUMENT> tags. Nothing outside " +
		"the tags."

	user := "PSD TO TRANSLATE:\n\n" +
		InputBeginMarker + "\n" + escapeMarkers(session.Psd) + "\n" + InputEndMarker

	res, err := client.Complete(ctx, cfg, ChatRequest{
		System:   system,
		Messages: []Message{{Role: "user", Content: user}},
	}, Hooks{})
	if err != nil {
		return nil, fmt.Errorf("handoff: %w", err)
	}

	brief, briefOK := extractDelimited(res.Content, "BRIEF_DOCUMENT")
	contextSeed, ctxOK := extractDelimited(res.Content, "CONTEXT_DOCUMENT")
	if !briefOK || !ctxOK {
		return nil, fmt.Errorf("handoff: response did not contain both <BRIEF_DOCUMENT> and <CONTEXT_DOCUMENT> blocks")
	}
	return &HandoffBundle{BriefMD: brief, ContextMD: contextSeed}, nil
}

// extractDelimited pulls the text between <TAG> and </TAG>.
func extractDelimited(content, tag string) (string, bool) {
	open := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	start := strings.Index(content, open)
	if start < 0 {
		return "", false
	}
	rest := content[start+len(open):]
	end := strings.Index(rest, closeTag)
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(rest[:end]), true
}
