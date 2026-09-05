package engine

// JSON schema builders for structured output. Each structured output type has
// a corresponding schema passed in the `format` parameter (Layer 1 of the
// structured-output strategy). Hand-written literals: exact control over the
// contract, no reflection magic.

// Phase0Schema matches the Phase 0 output defined in prompts/shisho.md.
func Phase0Schema() map[string]any {
	blocks := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"what_is_wanted":          blockEnum(),
			"how_it_should_be_done":   blockEnum(),
			"what_is_not_wanted":      blockEnum(),
			"how_success_is_measured": blockEnum(),
		},
		"required": []string{"what_is_wanted", "how_it_should_be_done", "what_is_not_wanted", "how_success_is_measured"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"project_name":         stringField(),
			"domain":               enumField([]string{"game", "saas", "tool", "software", "hybrid"}),
			"tags":                 arrayField(stringField()),
			"classification_notes": stringField(),
			"diagnostic":           blocks,
			"flags": arrayField(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"severity": enumField([]string{"low", "medium", "high", "critical"}),
					"title":    stringField(),
					"body":     stringField(),
				},
				"required": []string{"severity", "title", "body"},
			}),
			"agents": arrayField(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           stringField(),
					"display_name": stringField(),
					"wave":         enumField([]int{1, 2, 3}),
					"enabled":      map[string]any{"type": "boolean"},
					"rationale":    stringField(),
				},
				"required": []string{"id", "display_name", "wave", "enabled", "rationale"},
			}),
		},
		"required": []string{"project_name", "domain", "tags", "classification_notes", "diagnostic", "flags", "agents"},
	}
}

// AgentOutputSchema matches the specialist response envelope defined in
// prompts/agent_template.md. The "thinking" field is a placeholder the model
// never writes into — the engine populates it from the extracted think block.
func AgentOutputSchema() map[string]any {
	finding := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":     enumField([]string{"opportunity", "risk", "question", "constraint"}),
			"severity": enumField([]string{"low", "medium", "high", "critical"}),
			"title":    stringField(),
			"body":     stringField(),
		},
		"required": []string{"type", "severity", "title", "body"},
	}
	psdContribution := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"section": stringField(),
			"content": stringField(),
		},
		"required": []string{"section", "content"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_id":     stringField(),
			"display_name": stringField(),
			"wave":         map[string]any{"type": "integer"},
			"status":       enumField([]string{"done", "partial", "error"}),
			"output": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary":           stringField(),
					"findings":          arrayField(finding),
					"recommendation":    stringField(),
					"open_questions":    arrayField(stringField()),
					"psd_contributions": arrayField(psdContribution),
				},
				"required": []string{"summary", "findings", "recommendation", "open_questions", "psd_contributions"},
			},
			"thinking": stringField(),
		},
		"required": []string{"agent_id", "display_name", "wave", "status", "output", "thinking"},
	}
}

func stringField() map[string]any {
	return map[string]any{"type": "string"}
}

func enumField[T any](values []T) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func arrayField(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

func blockEnum() map[string]any {
	return enumField([]string{"present", "partial", "missing"})
}
