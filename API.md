# API Reference — Kumite

Base URL: `http://localhost:3001`

All request and response bodies are JSON unless noted. No route returns HTML. Every route is designed to be consumed by any client — the SvelteKit app today, a planned TUI later — with no client-specific behavior.

---

## Sessions

### GET /api/sessions
List all sessions (metadata only, no full content).

**Query params:**
- `domain` — filter by domain string
- `phase` — filter by phase
- `tag` — filter by tag (exact match, single tag)
- `has_psd` — `true` | `false`
- `distilled` — `true` | `false`, filters on whether intake ran

**Response:** `SessionListItem[]`

---

### GET /api/sessions/:id
Get full session including both input versions, pipeline plan, agent outputs, and PSD.

**Response:** `Session`

**Errors:**
- `404` — session not found

---

### POST /api/sessions
Create a new session. The submitted material goes to `raw_input`; `raw_source` stays null until intake runs.

**Body:**
```json
{
  "project_name": "My Game",
  "raw_input": "A platformer about a robot..."
}
```

**Response:** `{ "id": "uuid" }`

---

### PATCH /api/sessions/:id
Partial update. Permitted fields: `project_name`, `phase`, `raw_input`, `pipeline_plan`, `agent_outputs`, `psd`, `tags`, `domain`.

`raw_source` is not writable through this route. It is set only by the intake endpoint, because its null state is a record of which path the session took.

Complex fields (`pipeline_plan`, `agent_outputs`) are accepted as objects and serialized server-side.

**Response:** `{ "ok": true }`

---

### DELETE /api/sessions/:id
Delete session and all associated data.

**Response:** `{ "ok": true }`

---

## Intake (Uchikomi)

### GET /api/intake/explain
Return the static explanation of both intake paths, so every client presents the same reasoning without duplicating copy. Content is static; no session required.

**Response:**
```json
{
  "distill": {
    "label": "Distill first",
    "when": "Your input is a transcript of conversations with AI models, or several files of loose brainstorming.",
    "why": "Most text in a transcript was written by a model, not by you. Distillation separates what you decided from what a model suggested, so the panel analyzes your project rather than the model's proposals.",
    "cost": "One extra model call. Your original is preserved."
  },
  "skip": {
    "label": "Use as-is",
    "when": "Your input is a brief you wrote yourself, or an existing PSD.",
    "why": "Distillation compresses and rephrases. When you already wrote the brief, that loses your own wording for no gain.",
    "cost": "None."
  }
}
```

---

### POST /api/intake/:id
Run Uchikomi on the session. Copies `raw_input` to `raw_source`, distills, and replaces `raw_input` with the resulting brief. Session must be in `intake` phase.

Regular HTTP POST, not SSE. Sets phase to `distilling` while running, back to `intake` on completion or failure — intake does not advance the session, it only transforms the input.

**Response:**
```json
{
  "raw_input": "distilled brief markdown",
  "raw_source": "original material"
}
```

**Errors:**
- `404` — session not found
- `409` — session not in `intake` phase, or `raw_source` already set (intake already ran)
- `422` — `raw_input` is empty
- `502` — LLM endpoint unreachable
- `500` — output failed structural validation after retries

On any error the session returns to `intake` with `raw_input` untouched and `raw_source` null. The user may retry or proceed without distillation.

---

### DELETE /api/intake/:id
Revert a distillation. Restores `raw_input` from `raw_source` and sets `raw_source` to null. Only valid before Phase 0 has run.

**Response:** `{ "ok": true }`

**Errors:**
- `404` — session not found, or no distillation to revert
- `409` — session has already advanced past `intake`

---

## Pipeline

### POST /api/pipeline/phase0/:id
Run Shishō Phase 0 on the session's `raw_input`. Updates session phase to `pipeline_review` and writes `pipeline_plan`.

Regular HTTP POST, not SSE. Phase 0 is fast enough not to require streaming.

**Response:**
```json
{
  "pipeline_plan": { "...PipelinePlan": true }
}
```

**Errors:**
- `404` — session not found
- `422` — `raw_input` is empty
- `502` — LLM endpoint unreachable
- `500` — Phase 0 output failed schema validation after retries

---

### PATCH /api/pipeline/:id/plan
Replace the pipeline plan with user modifications after pipeline-builder review. Session stays in `pipeline_review` until the run is triggered.

Server-side validation: the Fixed wave cannot be emptied or reordered, and every `agent_id` must resolve to a roster entry or a custom agent. Clients do not enforce this.

**Body:** `PipelinePlan`

**Response:** `{ "ok": true }`

**Errors:**
- `409` — plan removes or moves the Fixed wave
- `422` — unresolvable `agent_id`

---

### POST /api/pipeline/run/:id
Execute the pipeline. Opens an SSE stream. Session must be in `pipeline_review` phase.

**Response:** `text/event-stream`

SSE event types:
```
pipeline_start    {"session_id":"...","total_agents":N}
agent_start       {"agent_id":"...","display_name":"...","wave":N}
agent_thinking    {"agent_id":"...","chunk":"..."}
agent_done        {"agent_id":"...","output":{AgentOutput}}
agent_error       {"agent_id":"...","error":"..."}
wave_complete     {"wave":N}
synthesis_start   {}
psd_chunk         {"chunk":"..."}
psd_done          {"psd":"..."}
pipeline_complete {"session_id":"..."}
pipeline_error    {"error":"..."}
```

State is persisted before every LLM call, so a client that drops mid-run reconstructs progress from `GET /api/sessions/:id`. The stream is a live view, not the source of truth.

**Errors (before SSE opens):**
- `404` — session not found
- `409` — session not in `pipeline_review` phase

---

### POST /api/pipeline/resume/:id
Resume an interrupted run. Session must be in `interrupted` phase. Walks the persisted plan in wave order, skipping nodes already `done` or `skipped`, executing the rest. If every node is complete and `psd` is null, goes straight to synthesis.

Opens the same SSE stream as a normal run, preceded by one extra event:

```
run_resumed  {"session_id":"...","skipping":["agent_id",...],"remaining":N}
```

**Errors (before SSE opens):**
- `404` — session not found
- `409` — session not in `interrupted` phase

---

### GET /api/pipeline/stream/:id
Attach to a run already in flight. Returns the same event stream as the run endpoint, from the current moment forward. Events already emitted are not replayed — fetch `GET /api/sessions/:id` first for completed state, then attach for what follows.

This exists because the run outlives the client. A user who closes the tab mid-run and returns has no stream to subscribe to otherwise, and clients never poll.

**Response:** `text/event-stream`

**Errors:**
- `404` — session not found
- `409` — no run in flight for this session

---

## Agents

### GET /api/agents/roster
Return the full built-in roster from `roster/agents.json`.

**Response:** `RosterEntry[]`

---

### GET /api/agents/:agentId
Fetch agent prompt content. Checks cache first, fetches from GitHub if expired.

**Query params:**
- `source_url` — override URL (for custom agents or advanced roster)

**Response:**
```json
{
  "agent_id": "engineering-software-architect",
  "content": "...",
  "cached": true,
  "stale": false
}
```

---

## Custom Agents

### GET /api/agents/custom
List all user-defined custom agents.

**Response:** `CustomAgent[]`

---

### POST /api/agents/custom
Create a custom agent.

**Body:**
```json
{
  "display_name": "Domain Expert",
  "role_summary": "Analyzes X for Y projects.",
  "wave_preference": 1,
  "system_prompt": "You are..."
}
```

**Response:** `{ "id": "slug" }`

---

### PUT /api/agents/custom/:id
Replace custom agent definition. Body same as POST.

**Response:** `{ "ok": true }`

---

### DELETE /api/agents/custom/:id
Delete a custom agent.

**Response:** `{ "ok": true }`

---

## Config

### GET /api/config
Return all config key-value pairs. The API key is returned masked.

**Response:**
```json
{
  "llm_endpoint": "https://ollama.com/v1",
  "llm_model": "qwen3:32b",
  "llm_api_key": "••••••••"
}
```

---

### PUT /api/config
Upsert config values. Accepts partial updates. Takes effect on the next LLM call — no restart.

**Body:**
```json
{
  "llm_endpoint": "https://ollama.com/v1",
  "llm_model": "qwen3:32b",
  "llm_api_key": "..."
}
```

**Response:** `{ "ok": true }`

---

## Handoff (Dojo Bundle)

### POST /api/handoff/:sessionId
Generate the Dojo handoff bundle from a completed session's PSD. Session must have a non-null `psd`. Calls Shishō with a dedicated handoff instruction — plain markdown output, no schema constraint.

**Response:**
```json
{
  "brief_md": "# <Project> — Project Brief\n...",
  "context_md": "# CONTEXT\n## Glossary\n...## Non-Goals\n...## Decisions\n..."
}
```

**Errors:**
- `404` — session not found
- `409` — session has no PSD yet (run the panel first)
- `502` — LLM endpoint unreachable

---

## Health

### GET /health

**Response:**
```json
{
  "ok": true,
  "model": "qwen3:32b",
  "endpoint": "https://ollama.com/v1"
}
```
