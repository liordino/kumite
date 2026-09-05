package engine

import (
	"fmt"
	"sort"
	"strings"

	"kumite/models"
	"kumite/prompts"
)

// template.go assembles the specialist prompt envelope. The envelope text
// lives in prompts/agent_template.md — a prompt-engineering artifact loaded
// at runtime, never modified during implementation.

// Input markers around the project material. The marker integrity rule:
// {RAW_INPUT} is the only content placed between them, and if the input
// itself contains either marker it is escaped before substitution.
const (
	InputBeginMarker = "<<<PROJECT_INPUT_BEGIN>>>"
	InputEndMarker   = "<<<PROJECT_INPUT_END>>>"
)

// LoadTemplate extracts the fenced template block from the artifact.
func LoadTemplate() (string, error) {
	text, err := prompts.AgentTemplate()
	if err != nil {
		return "", fmt.Errorf("load template: %w", err)
	}
	start := strings.Index(text, "## Template")
	if start < 0 {
		return "", fmt.Errorf("load template: no '## Template' section in the template artifact")
	}
	fenceStart := strings.Index(text[start:], "```")
	if fenceStart < 0 {
		return "", fmt.Errorf("load template: no fenced block under '## Template'")
	}
	from := start + fenceStart + len("```")
	// Skip a language tag on the opening fence line, if any.
	if nl := strings.IndexByte(text[from:], '\n'); nl >= 0 && !strings.HasPrefix(strings.TrimSpace(text[from:from+nl]), "{") {
		from += nl + 1
	}
	fenceEnd := strings.Index(text[from:], "```")
	if fenceEnd < 0 {
		return "", fmt.Errorf("load template: unterminated fenced block")
	}
	return strings.TrimSpace(text[from : from+fenceEnd]), nil
}

// AssemblePrompt builds the full system prompt for one specialist call,
// following the substitution order from the artifact's Runtime Assembly
// Notes:
//
//  1. {AGENT_SYSTEM_PROMPT}  (fetched roster prompt, or a custom agent's
//     locally stored system prompt — resolved by the caller)
//  2. project-context tokens from models.PipelineContext
//  3. {DISTILLED_NOTE}
//  4. {RAW_INPUT} strictly between the input markers
//  5. {WAVE2_CONTEXT} omitted entirely for Wave 1 agents
//  6. {AGENT_ID}, {DISPLAY_NAME}, {WAVE}
func AssemblePrompt(templateText, systemPrompt string, node models.AgentNode, plan models.PipelinePlan, priorOutputs []models.AgentOutput, rawInput string) (string, error) {
	if node.Wave != models.Wave1 && node.Wave != models.Wave2 && node.Wave != models.Fixed {
		return "", fmt.Errorf("assemble prompt %s: invalid wave %d", node.ID, node.Wave)
	}

	prompt := templateText
	prompt = strings.ReplaceAll(prompt, "{AGENT_SYSTEM_PROMPT}", systemPrompt)

	ctx := plan.Context
	prompt = strings.ReplaceAll(prompt, "{PROJECT_NAME}", plan.ProjectName)
	prompt = strings.ReplaceAll(prompt, "{INPUT_TYPE}", string(plan.InputType))
	prompt = strings.ReplaceAll(prompt, "{PROJECT_TYPE}", string(plan.ProjectType))
	prompt = strings.ReplaceAll(prompt, "{MATURITY}", string(ctx.Maturity))
	prompt = strings.ReplaceAll(prompt, "{DISTILLED_NOTE}", distilledNote(ctx.Distilled))
	prompt = strings.ReplaceAll(prompt, "{PROBLEM_STATEMENT}", ctx.ProblemStatement)
	prompt = strings.ReplaceAll(prompt, "{FOUR_BLOCK_WANTED}", string(ctx.FourBlock.WhatIsWanted))
	prompt = strings.ReplaceAll(prompt, "{FOUR_BLOCK_HOW}", string(ctx.FourBlock.HowItShouldBeDone))
	prompt = strings.ReplaceAll(prompt, "{FOUR_BLOCK_NOT_WANTED}", string(ctx.FourBlock.WhatIsNotWanted))
	prompt = strings.ReplaceAll(prompt, "{FOUR_BLOCK_SUCCESS}", string(ctx.FourBlock.HowSuccessIsMeasured))
	prompt = strings.ReplaceAll(prompt, "{FLAGS}", renderFlags(ctx.Flags))

	// {RAW_INPUT}: escaped if it contains a marker, placed strictly between
	// the markers. Nothing else is ever placed there.
	safeInput := escapeMarkers(rawInput)
	prompt = strings.ReplaceAll(prompt,
		InputBeginMarker+"\n{RAW_INPUT}\n"+InputEndMarker,
		InputBeginMarker+"\n"+safeInput+"\n"+InputEndMarker)
	prompt = strings.ReplaceAll(prompt, "{RAW_INPUT}", safeInput)

	if node.Wave == models.Wave1 {
		prompt = strings.ReplaceAll(prompt, "\n{WAVE2_CONTEXT}\n", "")
		prompt = strings.ReplaceAll(prompt, "{WAVE2_CONTEXT}", "")
	} else {
		prompt = strings.ReplaceAll(prompt, "{WAVE2_CONTEXT}", renderWave2Context(node, priorOutputs))
	}

	prompt = strings.ReplaceAll(prompt, "{AGENT_ID}", node.AgentID)
	prompt = strings.ReplaceAll(prompt, "{DISPLAY_NAME}", node.DisplayName)
	prompt = strings.ReplaceAll(prompt, "{WAVE}", waveNumber(node.Wave))
	return prompt, nil
}

func distilledNote(distilled bool) string {
	if distilled {
		return "This input was distilled from raw conversation transcripts. Statements are tagged [A] author-stated or [C] author-confirmed. Treat both as the author's position. The wording is a compression, so judge substance rather than phrasing."
	}
	return "This is the author's original material, unmodified."
}

func renderFlags(flags []string) string {
	if len(flags) == 0 {
		return "None flagged."
	}
	return "- " + strings.Join(flags, "\n- ")
}

func waveNumber(w models.Wave) string {
	if w == models.Fixed {
		return "3"
	}
	return fmt.Sprintf("%d", int(w))
}

// escapeMarkers defangs marker strings that appear inside the project input
// so the assembled prompt can never contain a second, ambiguous marker pair.
func escapeMarkers(input string) string {
	input = strings.ReplaceAll(input, InputBeginMarker, "< < <PROJECT_INPUT_BEGIN> > >")
	input = strings.ReplaceAll(input, InputEndMarker, "< < <PROJECT_INPUT_END> > >")
	return input
}

// renderWave2Context builds the PRIOR SPECIALIST FINDINGS block for Wave 2
// and Fixed agents:
//   - only agents with status "done" from prior waves
//   - ordered by wave, then by position in the pipeline plan
//   - capped at the top 3 findings per agent, sorted critical-first
//   - thinking content is never included
//   - if the block would exceed the token budget, finding bodies truncate to
//     the first sentence
func renderWave2Context(node models.AgentNode, priorOutputs []models.AgentOutput) string {
	// Include only outputs from waves strictly before this node's wave.
	prior := make([]models.AgentOutput, 0, len(priorOutputs))
	for _, o := range priorOutputs {
		if o.Status != "done" {
			continue
		}
		if models.Wave(o.Wave) < node.Wave {
			prior = append(prior, o)
		}
	}
	if len(prior) == 0 {
		return ""
	}
	// Plan position order: Wave 1 nodes were appended in plan order, so the
	// output slice order IS wave-then-position (the runner appends in walk
	// order). Stable sort keeps that order across equal waves.
	sort.SliceStable(prior, func(i, j int) bool {
		return prior[i].Wave < prior[j].Wave
	})

	var b strings.Builder
	b.WriteString("## PRIOR SPECIALIST FINDINGS\n\n")
	b.WriteString("The following agents have already analyzed this project. Their findings are provided for\n")
	b.WriteString("context. React to them where relevant. Challenge them if your domain expertise warrants it.\n")
	b.WriteString("Do not simply agree — add friction where you see gaps.\n\n")
	for _, o := range prior {
		fmt.Fprintf(&b, "### %s\n", o.DisplayName)
		fmt.Fprintf(&b, "**Summary:** %s\n", o.Output.Summary)
		b.WriteString("**Key findings:**\n")
		findings := append([]models.Finding(nil), o.Output.Findings...)
		sort.SliceStable(findings, func(i, j int) bool {
			return SeverityRank(findings[i].Severity) < SeverityRank(findings[j].Severity)
		})
		if len(findings) > 3 {
			findings = findings[:3]
		}
		for _, f := range findings {
			body := f.Body
			if estimateTokens(b.String())+estimateTokens(body) > wave2TokenBudget {
				body = firstSentence(body)
			}
			fmt.Fprintf(&b, "- [%s] [%s] %s: %s\n", strings.ToUpper(string(f.Severity)), f.Type, f.Title, body)
		}
		fmt.Fprintf(&b, "**Recommendation:** %s\n\n", o.Output.Recommendation)
	}
	return b.String()
}

const wave2TokenBudget = 8000

// estimateTokens approximates tokens as ~4 characters — deliberately rough;
// it only drives the finding-body truncation rule.
func estimateTokens(s string) int { return len(s) / 4 }

func firstSentence(s string) string {
	for _, sep := range []string{". ", "! ", "? "} {
		if i := strings.Index(s, sep); i >= 0 {
			return s[:i+len(sep)-1]
		}
	}
	return s
}
