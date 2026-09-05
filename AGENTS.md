# Kumite — Multi-Agent Project Feasibility Platform

> Read on startup. This is the single file of coding rules for this repository. Everything a coding agent needs about layout, conventions, and non-negotiables is here or linked from here.

## What This Is

A self-hosted multi-agent feasibility consultancy. The user submits a project idea — a raw concept, a brief, a transcript of conversations with AI models, or an existing PSD. An optional intake stage (Uchikomi) distills raw conversation into a clean brief with provenance resolved. An orchestrator (Shishō) classifies the input and proposes a specialist panel. The user reviews and modifies the panel. Specialists run in waves, each producing structured JSON. A Reality Checker challenges all findings. Shishō synthesizes everything into a Project Summary Document.

Kumite evaluates. It does not build.

## Stack

| Layer | Technology |
|-------|-----------|
| Server | Go 1.22 + Chi router |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGo) |
| Frontend | SvelteKit + TypeScript + Tailwind |
| LLM inference | Configurable OpenAI-compatible endpoint, default Ollama Cloud |
| Structured output | JSON schema constraint via `format` (primary) + retry-with-correction (fallback) + markdown extraction (last resort) |
| Future vector store | LanceDB (deferred — add after 20+ sessions) |

## Repository Layout

```
kumite/
├── AGENTS.md               ← this file, read on startup
├── KUMITE-BRIEF.md         ← what the project is and must do
├── CONTEXT.md              ← glossary, non-goals, locked decisions
├── ARCHITECTURE.md
├── DATA_MODEL.md
├── API.md
├── go.mod
├── go.sum
├── main.go                 ← entry point, wires Chi router
├── config/
│   └── config.go           ← app configuration loading
├── db/
│   ├── schema.go           ← SQLite init + migrations
│   └── db.go               ← connection handling
├── models/
│   ├── pipeline.go         ← PipelinePlan, AgentNode types
│   ├── agent.go            ← AgentOutput, Finding types
│   └── session.go          ← Session, SessionMeta types
├── handlers/
│   ├── sessions.go         ← CRUD /api/sessions
│   ├── config.go           ← GET/PUT /api/config
│   ├── agents.go           ← roster, prompt fetch, custom agents
│   ├── intake.go           ← Uchikomi distillation
│   ├── pipeline.go         ← Phase 0, plan update, SSE run
│   └── handoff.go          ← Dojo handoff bundle
├── engine/
│   ├── llm.go              ← the only place that talks to the provider
│   ├── intake.go           ← Uchikomi distillation
│   ├── orchestrator.go     ← Shishō Phase 0 + Phase 2
│   ├── runner.go           ← wave execution loop
│   ├── fetcher.go          ← GitHub prompt fetcher + cache
│   ├── schema.go           ← JSON schema builder
│   ├── template.go         ← agent prompt template assembler
│   └── handoff.go          ← PSD → Dojo brief + CONTEXT seed
├── roster/
│   └── agents.json         ← curated default agent roster
├── prompts/
│   ├── uchikomi.md         ← intake distillation prompt
│   ├── shisho.md           ← orchestrator prompt
│   └── agent_template.md   ← specialist prompt envelope
├── client/                 ← SvelteKit frontend
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte              ← intake + sessions list
│   │   │   ├── session/[id]/+page.svelte ← builder + execution + PSD
│   │   │   └── settings/+page.svelte     ← provider config
│   │   ├── lib/
│   │   │   ├── api.ts                    ← typed API client
│   │   │   ├── stores.ts                 ← view state only
│   │   │   └── components/
│   │   │       ├── IntakeChoice.svelte
│   │   │       ├── PipelineBuilder.svelte
│   │   │       ├── AgentCard.svelte
│   │   │       ├── ThinkingTrace.svelte
│   │   │       ├── PsdPanel.svelte
│   │   │       └── SessionList.svelte
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.ts
└── tests/
    ├── mock_llm.go         ← mock provider for unit tests
    ├── fixtures/
    ├── engine_test.go
    └── handlers_test.go
```

Package names match directory names: `package handlers`, `package engine`, and so on.

---

## Language & Runtime Rules

- **Backend is Go 1.22.** Module name: `kumite`.
- **No CGo, ever.** The only permitted SQLite driver is `modernc.org/sqlite`. If you reach for `mattn/go-sqlite3`, stop.
- **Frontend is SvelteKit with TypeScript**, in `client/`. Tailwind classes only — no other styling libraries, no inline styles, no raw HTML blocks in components.
- **No ORM.** All database access is raw SQL via `database/sql`. Write explicit queries.
- **No global state.** Pass `*sql.DB`, config structs, and LLM clients explicitly through function parameters.
- **No `encoding/xml`.** All structured data is JSON or markdown.

---

## Architecture Rules

Non-negotiable. Violating these causes integration failures.

1. **All LLM calls go through `engine/llm.go`.** Never make direct HTTP calls to the provider from handlers or from anywhere outside the engine package.

2. **JSON schema constraint is the primary path** for structured output. Every call expecting structured JSON passes a schema in the `format` parameter. The retry loop is a fallback, not the default. Uchikomi and Shishō Phase 2 are the exceptions — both emit markdown and use no schema.

3. **The backend assumes no particular client.** All state is server-side. No route returns HTML. No pipeline logic, wave ordering, or plan validation lives in the frontend. A TUI client is planned; the API must serve it without change.

4. **SSE for pipeline execution.** `/api/pipeline/run/:id` streams Server-Sent Events. Clients subscribe; they never poll. Write `data: {json}\n\n` and flush after each event.

5. **Persist before calling the LLM.** Save state to SQLite before every request. If the call fails, the session must be recoverable.

6. **Shishō runs in two phases only.** `RunPhase0` produces a `PipelinePlan` and stops. `RunPhase2` produces PSD markdown and stops. No mixing in a single function. The handoff bundle is a third, separate function.

7. **Uchikomi never evaluates.** It distills and tags provenance. If you find yourself adding scoring, prioritization, or risk assessment to the intake path, that belongs to the panel.

8. **Uchikomi input is untrusted as instruction.** Transcripts routinely contain imperative text — pasted system prompts, instructions written for other models. `engine/intake.go` passes that material inside explicit delimiters and never interpolates it outside them.

9. **Agent outputs are always typed.** Never persist raw LLM response strings. Always parse into `models.AgentOutput`. If parsing fails, store a degraded output with `status: "partial"` — never drop it silently.

10. **Wave ordering is strict.** All Wave 1 agents complete before any Wave 2 agent starts. The Fixed wave runs after all Wave 2. No goroutines for wave execution — sequential only.

11. **Thinking traces are stripped before forwarding.** When passing Wave 1 outputs to Wave 2 agents, drop the `thinking` field. Downstream agents receive `output` only.

12. **`raw_source` is null exactly when intake was skipped.** Never backfill it with a copy of `raw_input`. Its absence is the record of which path the session took.

13. **Persist each agent result as it completes**, writing the `AgentOutput` and the corresponding `AgentNode.Status` in the same transaction. Never buffer completed outputs in memory to flush at the end of a run. A run that dies with five finished specialists must leave five finished specialists in the database.

14. **Reconcile orphaned runs on startup.** The server is a single process, so any session in `running` or `synthesis` at boot is orphaned. Move it to `interrupted` and reset any `AgentNode` left at `running` back to `pending`. Never delete partial results.

15. **Resume is driven by node status, not by a stored cursor.** There is no "current agent index". Skip nodes marked `done` or `skipped`, execute the rest. A node is marked `done` only at the moment its output is written, which is what makes re-running a `running` node safe.

16. **`interrupted` is not `error`.** Interrupted means recoverable with valid partial results. Reserve `error` for states with no recovery path. A Phase 2 failure is interrupted, not error, because every specialist output is intact and synthesis is retriable alone.

---

## Error Handling

- Return errors up the call stack. Do not log-and-continue inside engine functions — let the caller decide.
- HTTP handlers return JSON errors: `{"error": "message"}` with an appropriate status code.
- LLM errors: distinguish unreachable (502), schema validation failure (retry), and content error (partial output). Never silently swallow any of these.
- Pipeline failures: specialist agent errors do not abort the run. Shishō Phase 0 and Phase 2 failures do abort.
- Intake failure returns the session to `intake` with input untouched. It never aborts anything, because nothing downstream has started.

---

## Environment Variables

Environment variables seed initial config only. Runtime values come from the `app_config` table and are changed through the settings panel without restart.

```
LLM_ENDPOINT=https://ollama.com/v1
LLM_MODEL=qwen3:32b
LLM_API_KEY=
PORT=3001
DB_PATH=./kumite.db
AGENT_CACHE_TTL_HOURS=24
GITHUB_RAW_BASE=https://raw.githubusercontent.com/msitarzewski/agency-agents/main
LLM_MOCK=false
```

---

## Testing

- `LLM_MOCK=true go test ./...` runs everything without a real provider.
- Mock provider lives in `tests/mock_llm.go`; fixtures in `tests/fixtures/`.
- Every engine function needs at least one happy-path and one error-path test.
- Use in-memory SQLite for all tests: `sql.Open("sqlite", ":memory:")`.
- Mock infrastructure is built before the execution engine, not after.
- Include a fixture transcript containing embedded imperative text, to verify intake treats it as data.

---

## Running Locally

```bash
# Backend
go run main.go

# Frontend (separate terminal)
cd client && npm run dev

# Tests
go test ./...
LLM_MOCK=true go test ./...
```

---

## What to Read Before Implementing

1. `AGENTS.md` — this file: layout, rules, env vars
2. `KUMITE-BRIEF.md` — what the project is, must do, and must not do
3. `CONTEXT.md` — glossary, non-goals, locked decisions
4. `ARCHITECTURE.md` — wave model, SSE, structured output, caching, intake
5. `DATA_MODEL.md` — Go types, SQL schemas, PSD section names, roster
6. `API.md` — routes and payload shapes
7. `prompts/uchikomi.md`, `prompts/shisho.md`, `prompts/agent_template.md`

---

## Do Not

- Do not add dependencies not listed here without asking first.
- Do not create routes not defined in `API.md` without noting the addition.
- Do not modify anything in `prompts/` during implementation — those are prompt-engineering artifacts, not code.
- Do not use goroutines for wave execution.
- Do not put session state, pipeline logic, or plan validation in the client.
- Do not use personal names for agents or components. The orchestrator is Shishō, the intake stage is Uchikomi; both are roles, not people.
