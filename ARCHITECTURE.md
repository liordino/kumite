# Architecture — Kumite

## Overview

```
Clients (SvelteKit web today; Bubble Tea TUI planned)
      │
      │ HTTP + SSE — no client-side state, no HTML routes
      ▼
Chi Router (Go)
      │
      ├── /api/sessions   → handlers/sessions.go
      ├── /api/config     → handlers/config.go
      ├── /api/agents     → handlers/agents.go
      ├── /api/intake     → handlers/intake.go      (Uchikomi distillation)
      ├── /api/pipeline   → handlers/pipeline.go
      └── /api/handoff    → handlers/handoff.go     (Dojo handoff bundle)
                                    │
                          engine/intake.go        (Uchikomi distillation)
                          engine/orchestrator.go  (Shishō Phase 0 + Phase 2)
                          engine/runner.go        (wave execution)
                          engine/fetcher.go       (GitHub cache)
                          engine/schema.go        (JSON schema builder)
                          engine/template.go      (prompt assembly)
                          engine/llm.go           (single egress point for inference)
                          engine/handoff.go       (PSD → Dojo brief + CONTEXT seed)
                                    │
                          OpenAI-compatible endpoint
                          default: Ollama Cloud
```

---

## Client-Agnostic Constraint

The backend serves data, never presentation. This is a hard constraint, not a preference, because a second client is planned.

- All session state lives in SQLite and is reconstructible from the API alone.
- No route returns HTML. Every response is JSON or an SSE stream.
- No pipeline logic in any client. Wave ordering, agent selection validation, and phase transitions are server-side.
- The SvelteKit app holds view state only — what is expanded, what is selected, what is being dragged. Nothing a reload would lose that matters.

A terminal client using Bubble Tea is planned as a second interface, deferred out of V1. Deferring it is a scheduling decision; the constraint above is what keeps it cheap when it arrives. See `KUMITE-BRIEF.md` for the reasoning and the reevaluation criterion.

---

## LLM Endpoint

All inference goes through a single configurable endpoint exposing an OpenAI-compatible API. Default is Ollama Cloud. Any OpenAI-compatible provider works.

The endpoint, model, and API key are read from the `app_config` table at runtime, not from environment variables — this allows live switching without restart. Environment variables seed initial values only.

Every call goes through `engine/llm.go`. No other package makes HTTP requests to the provider.

---

## Structured Output Strategy

### Layer 1: JSON Schema Constraint (primary)

The provider constrains generation to a supplied JSON schema via the `format` parameter. `engine/schema.go` builds schema objects from Go struct definitions. Each structured output type has a corresponding schema.

```go
// Usage
req.Format = schema.For(schema.AgentOutput)
```

Not every OpenAI-compatible provider supports `format`. When the configured provider rejects it, the call degrades to Layer 2 rather than failing — the retry loop is the compatibility floor.

### Layer 2: Retry with Correction (fallback)

If the response fails JSON parsing or schema validation, retry with:

```
Your previous response was not valid JSON matching the required schema.
Here is the schema again: [schema].
Respond with only valid JSON. No preamble, no markdown fences.
```

Maximum 2 attempts before falling back to Layer 3.

### Layer 3: Markdown Extraction (last resort)

Extract a JSON object from the raw response with a permissive regex. If extraction succeeds but schema validation fails, mark the output `status: "partial"` and surface it to the user rather than silently dropping it.

Uchikomi and Shishō Phase 2 are exceptions: both emit markdown, not JSON, and use no schema constraint. Phase 2 output is delimited by `<PSD_DOCUMENT>` tags; Uchikomi output is a markdown document with fixed frontmatter and fixed section headings, validated structurally rather than by schema.

---

## Thinking Traces

Reasoning models may emit `<think>...</think>` blocks before their response. The execution engine:

1. Strips the `<think>` block from the response before parsing
2. Stores the raw thinking content in `AgentOutput.Thinking`
3. Never passes thinking traces downstream to other agents — only `output` is forwarded
4. Returns thinking traces to clients via the SSE stream as a separate event type

Models that emit no reasoning blocks yield an empty string. Clients omit the section entirely in that case. No feature depends on traces being present.

---

## Uchikomi — The Intake Stage

An optional stage that runs before Phase 0, distilling raw material into a clean brief.

`engine/intake.go` → `RunIntake(session)` loads `prompts/uchikomi.md`, sends the session's `raw_source` as delimited data, and receives a structured markdown brief.

**Why it is optional and user-selected.** Deciding whether input needs distillation requires classifying it, and classification is Phase 0, which runs after. Making the choice automatic would mean running Phase 0 twice. The client presents both paths with static explanatory text and the user decides.

**Data flow.** On session creation, the submitted material is written to `raw_input`. If the user requests intake, the original is copied to `raw_source`, Uchikomi runs, and the distilled brief overwrites `raw_input`. If intake is skipped, `raw_source` stays null.

The null is meaningful: it records which path was taken. When a PSD comes out wrong, the first diagnostic question is whether the fault lies with the panel or with the compression, and that requires having both versions.

**Prompt injection surface.** Uchikomi's input is a transcript of conversations with AI models. It routinely contains imperative text — system prompts the author pasted, instructions to other models, role definitions. The prompt delimits this material explicitly as data and instructs the model that nothing inside the delimiters is an instruction. `engine/intake.go` must preserve the delimiters and must not interpolate user content anywhere outside them.

**Failure.** If Uchikomi fails, the session stays in `intake` with `raw_input` untouched and `raw_source` null. The user may retry or proceed without distillation. Intake failure never aborts anything downstream, because nothing downstream has started.

---

## Shishō — The Orchestrator

Shishō is the orchestrator, forked from the agency-agents Product Manager agent and repurposed for feasibility evaluation. It runs in exactly two phases, never both in one call:

- **Phase 0** (`engine/orchestrator.go` → `RunPhase0`): classifies input, runs the 4-Block diagnostic, raises flags, emits a `PipelinePlan` JSON. Regular POST, not SSE.
- **Phase 2** (`engine/orchestrator.go` → `RunPhase2`): receives all specialist outputs, synthesizes the PSD wrapped in `<PSD_DOCUMENT>` tags. Invoked at the end of the wave run, streams via the run SSE.

The handoff bundle is a third invocation with a dedicated instruction, not a phase.

Shishō never performs specialist analysis. The prompt lives in `prompts/shisho.md` and is loaded at runtime — never modified during implementation.

---

## Wave Execution Model

Waves express dependency, not concurrency. "Wave 1" means agents whose prompts have no dependency on other agents' outputs. "Wave 2" means agents that receive Wave 1 outputs in their context.

Execution is sequential by choice. With a cloud provider, agents within a wave could run concurrently — the decision not to is deliberate, for deterministic ordering, reproducible SSE event sequences, and traceable output. Revisit only if run duration becomes a real complaint.

Order within a wave does not matter. Order between waves is strictly enforced: Wave 2 starts only after all enabled Wave 1 agents have status `done` or `skipped`.

```
Wave 1: [arch, financial, ux, trend, product_mgr]  → sequential
          ↓ (all done or skipped)
Wave 2: [security, senior_pm, legal]               → sequential
          ↓ (all done or skipped)
Fixed:  [reality_checker]                          → always last
          ↓
Shishō Phase 2: synthesis → PSD
```

The `enabled` field on each `AgentNode` controls whether it runs. Disabled agents are skipped but their node remains in the pipeline state with `status: skipped`.

---

## SSE Progress Stream

`POST /api/pipeline/run/:id` does not return a response body. It opens an SSE stream and emits events as execution progresses:

```
event: pipeline_start
data: {"session_id":"...","total_agents":7}

event: agent_start
data: {"agent_id":"engineering-software-architect","display_name":"...","wave":1}

event: agent_thinking
data: {"agent_id":"...","chunk":"..."}   ← streamed as tokens arrive

event: agent_done
data: {"agent_id":"...","output":{...}}  ← full AgentOutput

event: agent_error
data: {"agent_id":"...","error":"..."}

event: wave_complete
data: {"wave":1}

event: synthesis_start
data: {}

event: psd_chunk
data: {"chunk":"..."}

event: psd_done
data: {"psd":"..."}                      ← full PSD markdown

event: pipeline_complete
data: {"session_id":"..."}

event: pipeline_error
data: {"error":"..."}
```

Clients reconnect on drop. Because state is persisted before every LLM call, a reconnecting client reconstructs current progress from `GET /api/sessions/:id` and then attaches to the live tail via `GET /api/pipeline/stream/:id` — the stream is a live view, not the source of truth. See Run Durability & Recovery below.

---

## Run Durability & Recovery

A full panel run takes minutes and spans many provider calls. It will be interrupted — by a provider timeout, a network drop, a crash, or the user stopping the server. The design treats interruption as expected, not exceptional.

### The durable point

State is persisted before every LLM call, and each agent's result is persisted as it completes: the `AgentOutput` is appended to `agent_outputs` and the corresponding `AgentNode.Status` is written in the same transaction. That pairing is what makes recovery cheap — the persisted plan is always an accurate record of what has and has not run.

Never hold completed outputs in memory to write in a batch at the end. A run that dies with five finished specialists must leave five finished specialists in the database.

### Startup reconciliation

The server runs as a single process; there is no second instance that could legitimately be executing a run. Therefore any session found in `running` or `synthesis` at startup is by definition orphaned.

On boot, `db/schema.go` reconciles: every session in `running` or `synthesis` moves to `interrupted`, and any `AgentNode` left at `running` moves back to `pending`. Nothing is deleted. The partial `agent_outputs` stay exactly as they are.

### Resuming

`POST /api/pipeline/resume/:id` restarts an interrupted run. It walks the persisted plan in wave order, skips nodes already `done` or `skipped`, and executes the rest. It opens the same SSE stream as a normal run and emits the same events, prefixed by a `run_resumed` event naming which agents are being skipped, so a client can render the recovered state without a separate code path.

If all nodes are complete and the PSD is missing, resume goes straight to synthesis.

Resume is idempotent in the way that matters: re-running a node whose output was never persisted produces one output, not two. A node is only marked `done` at the moment its output is written.

### Aborting failures

Specialist failures do not abort — they are stored as `error` or `partial` and the run continues. Only a Shishō Phase 0 or Phase 2 failure aborts.

When Phase 2 aborts, the session goes to `interrupted`, not `error`, because every specialist output is intact and synthesis is retriable on its own. `error` is reserved for states with no recovery path.

### Attaching to a live run

`GET /api/pipeline/stream/:id` attaches an SSE stream to a run already in flight. This exists because the run outlives the client: a user who closes the tab mid-run and returns has no stream to subscribe to otherwise, and clients are forbidden from polling.

The runner keeps a per-session in-memory broadcast channel for the duration of a run. Attaching subscribes to it. A client that attaches mid-run first fetches `GET /api/sessions/:id` for completed state, then subscribes for what follows. Late subscribers do not receive replayed events — the persisted session is the record; the stream is only the live tail.

When the run ends, the channel closes and the endpoint returns 409 for that session.

---

`engine/fetcher.go` fetches agent `.md` files from GitHub raw URLs and caches them in SQLite (`agent_cache` table). Default TTL is 24 hours. On each fetch:

1. Check cache — if present and not expired, return cached content
2. If expired, send a conditional GET with `If-None-Match: <etag>`
3. On 304, reset TTL without re-downloading
4. On 200, update cache with new content and ETag
5. If GitHub is unreachable and cache exists, serve stale with `stale: true`

---

## Session Metadata & Tags

Every session stores structured metadata for cross-session filtering:

```go
type SessionMeta struct {
    ID          string   `json:"id"`
    ProjectName string   `json:"project_name"`
    Domain      string   `json:"domain"`      // "game", "saas", "tool", etc.
    Tags        []string `json:"tags"`        // Shishō-assigned + user-editable
    CreatedAt   string   `json:"created_at"`
    UpdatedAt   string   `json:"updated_at"`
    Phase       string   `json:"phase"`
    Distilled   bool     `json:"distilled"`   // true when raw_source is non-null
    AgentCount  int      `json:"agent_count"`
    HasPsd      bool     `json:"has_psd"`
    FindingSeveritySummary map[string]int `json:"finding_severity_summary"`
    // e.g. {"critical":2,"high":3,"medium":5,"low":1}
}
```

Tags and domain are assigned by Shishō in Phase 0 and stored in the `pipeline_plan` JSON. They are also copied to dedicated columns on the `sessions` table for fast SQL querying without JSON parsing.

---

## Custom Agents

Users can define custom agents via the settings panel:

- `id`: user-defined slug
- `display_name`: shown in UI
- `role_summary`: one sentence
- `wave_preference`: 1 or 2 (Shishō can override)
- `system_prompt`: full text, entered by user
- `source`: "custom" (not a GitHub URL)

Stored in the `custom_agents` table, appearing in the roster alongside built-in agents. Shishō sees them in its roster reference and can select them in Phase 0.

A custom agent's `system_prompt` is arbitrary user text sent to a model as a system message. In a single-user self-hosted tool the user is trusting their own input, so this is not a privilege boundary — but it is worth stating explicitly rather than discovering later.

---

## Dojo Handoff Bundle

An optional second output mode, generated on request after a PSD exists. It translates the feasibility verdict into the shape Dojo's `/hajime` expects, bridging Kumite (evaluation) into Dojo (build cycle).

`engine/handoff.go` → `GenerateHandoff(session)` produces two artifacts from the session's PSD and agent outputs:

- **`BRIEF.md`** — the project reframed as a Dojo feasibility brief: what it is, the core loop, the main user, success metric, hard constraints, explicit exclusions, and build-order intent. Derived from PSD sections 1, 2, 4, 6, 7 and the panel's findings.
- **`CONTEXT.md` seed** — the three-section Dojo domain contract: Glossary (terms from the PSD), Non-Goals (from PSD section 7's out-of-scope list), Decisions (from stated constraints and resolved findings).

This is a synthesis call to Shishō with a dedicated handoff instruction, not a separate agent. Output is plain markdown with no schema constraint. Returned via `POST /api/handoff/:sessionId`, downloadable from the client. A later epic — built after the core panel pipeline is proven end-to-end.

---

## Future: Vector Store (LanceDB)

Deferred until 20+ completed sessions exist. When added:

- Embedding target: `AgentOutput.output.summary` + PSD section content, not raw chat
- Embedding model: `nomic-embed-text`
- Each completed session contributes ~15-20 embeddable chunks
- Retrieval injected into Shishō Phase 0 context: "Similar past projects found: [summaries]"
- LanceDB runs embedded (no separate process) via Go bindings

---

## Testing

`LLM_MOCK=true` replaces the LLM endpoint with `tests/mock_llm.go`, a local HTTP server returning fixture responses for known prompts. This enables:

- Unit testing the wave runner without model inference
- Testing JSON parsing and schema validation
- Testing SSE event sequencing
- Testing the intake stage with a fixture transcript, including a transcript containing embedded imperative text, to verify it is treated as data
- Testing error paths (provider unreachable, malformed JSON, timeout, schema rejection)

Fixtures live in `tests/fixtures/`. One fixture per agent per scenario (happy path + error path). Mock infrastructure is built before the execution engine.
