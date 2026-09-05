# CONTEXT

> Seed for Dojo's domain contract. randori extends this during the design grill but should treat the Decisions section as already-locked unless a genuine conflict surfaces. Three sections only: Glossary · Non-Goals · Decisions.

## Glossary

- **Kumite** — the project. A self-hosted multi-agent feasibility consultancy. Japanese for "sparring": an idea enters the floor, specialists test it, what survives is stronger.
- **Uchikomi** — the optional intake stage (打ち込み, the entry drill practiced before sparring). Distills raw conversation transcripts into a clean brief with provenance resolved. Never evaluates.
- **Shishō** — the Orchestrator agent (師匠, "master"). Runs Phase 0 (classify + propose pipeline) and Phase 2 (synthesize PSD). Never does specialist work.
- **Provenance tag** — Uchikomi's classification of every extracted statement: `[A]` author-stated, `[C]` author-confirmed (a model proposed it and the author explicitly adopted it), `[M]` model-proposed (raised by a model, never engaged with by the author).
- **Specialist** — a domain expert agent that analyzes the project from one perspective and emits structured findings. Sourced from the agency-agents repo.
- **Panel** — the full set of specialists selected for a given project run.
- **Wave** — a group of specialists with the same dependency level. Wave 1 = independent (needs only input). Wave 2 = reactive (needs Wave 1 outputs). Fixed = always last (Reality Checker). Expresses dependency, not concurrency.
- **Reality Checker** — the fixed final specialist. Challenges all prior findings before synthesis. Always runs last, never in Wave 1 or 2.
- **PSD (Project Summary Document)** — the feasibility verdict Kumite produces. Markdown, ten fixed sections, with section-level attribution to specialists.
- **Minimum Viable Slice** — PSD section 4. The smallest change that runs end to end and produces one real unit of the intended value. Not a roadmap; one slice.
- **Pipeline Plan** — the JSON object Shishō emits in Phase 0 describing which specialists run, in which wave, and why. Rendered into the pipeline builder for user modification.
- **Pipeline Builder** — the interface where the user reviews and modifies the proposed panel before execution: toggle agents, move between waves, add custom agents.
- **Finding** — a typed unit of specialist output: `type` (opportunity/risk/question/constraint), `severity` (low/medium/high/critical), title, body.
- **PSD Contribution** — a markdown fragment a specialist nominates for a specific PSD section. Shishō assembles these in synthesis.
- **4-Block Diagnostic** — Shishō's Phase 0 check: is the input clear on (1) what is wanted, (2) how it should be done, (3) what is NOT wanted, (4) how success is measured. Missing blocks are flagged, never blocking.
- **Thinking trace** — the `<think>...</think>` content a reasoning model emits before its answer. Preserved per agent, collapsed in UI, stripped before forwarding downstream.
- **Roster** — the set of available specialists. Curated default (13 agents) plus advanced mode (full agency-agents repo).
- **Custom agent** — a user-defined specialist: name, role summary, wave preference, system prompt. Stored locally, selectable like built-in agents.
- **Dojo handoff bundle** — optional Kumite output: a `BRIEF.md` + `CONTEXT.md` seed derived from the PSD, formatted for Dojo's `/hajime`. The bridge from feasibility verdict to build cycle.
- **agency-agents** — the upstream GitHub repo (msitarzewski, MIT) providing specialist prompts and the Shishō base persona.
- **Ollama Cloud** — the default inference provider, exposed as an OpenAI-compatible endpoint. Any OpenAI-compatible provider is swappable via settings.

## Non-Goals

- Kumite does not build, execute, or write the evaluated project's code. Evaluation only.
- Kumite does not replace Dojo. It feeds Dojo via the handoff bundle.
- Kumite is not a general-purpose agent orchestration framework.
- No parallel execution. Waves express dependency; sequential execution is a deliberate choice for deterministic ordering and traceable output, not a provider limitation.
- Uchikomi does not evaluate, score, prioritize, or recommend. It distills and separates provenance. Judgment belongs to the panel.
- Uchikomi does not decide whether it should run. The user chooses; the system explains both paths with static text.
- The Orchestrator does not perform specialist analysis. It classifies, routes, and synthesizes only.
- PSD synthesis does not fabricate content for empty sections.
- No ORM. No CGo. No XML parsing. No cloud-only dependencies.
- Clients do not poll for pipeline progress — SSE only.
- No session state in the frontend. No route returns HTML. No pipeline logic in the client.
- No editing of `prompts/uchikomi.md`, `prompts/shisho.md`, or `prompts/agent_template.md` during implementation — they are prompt-engineering artifacts.
- No CLI or TUI in V1. Planned as a second client; the backend must not foreclose it.
- The vector store (LanceDB) is not in V1 — deferred until 20+ sessions exist.

## Decisions

- **Language/stack:** Backend Go 1.22 + Chi router. Frontend SvelteKit + TypeScript + Tailwind. SQLite via `modernc.org/sqlite` (pure Go, no CGo).
- **Client-agnostic backend:** All state is server-side. The SvelteKit app is a client, not a half of the application. A second client (planned Bubble Tea TUI) must be able to consume the same routes with no backend change. This constrains implementation now even though the TUI is out of V1.
- **TUI deferred with a reevaluation criterion:** Revisit after roughly ten real sessions, measuring how often the user actually modifies the proposed panel. Frequent modification favors the web builder as primary; rare modification favors the TUI. When built, the Uchikomi choice is exposed as `--intake` / `--no-intake`, with the trade-off explanation in `--help` rather than in the flag.
- **Inference:** Configurable OpenAI-compatible endpoint. Default Ollama Cloud. Any OpenAI-compatible provider swappable via the settings panel, stored in `app_config`, switchable at runtime without restart.
- **Structured output:** JSON schema constraint via the `format` parameter (primary) → retry-with-correction fallback (max 2 attempts) → markdown extraction last resort. Failed-but-extracted outputs stored with `status: "partial"`, never dropped. Replaces the earlier GBNF approach, which was local-llama.cpp-specific.
- **Intake stage (Uchikomi):** Optional, user-selected, runs before Phase 0. Emits a structured brief with provenance tags. Original material preserved in `raw_source`; distilled brief becomes `raw_input`. When intake is skipped, `raw_source` is null — its absence is the record of which path was taken. The choice cannot be automatic, because classifying the input is Phase 0's job and Phase 0 runs after.
- **Provenance bar:** `[C]` requires that the author *used* the idea — referenced it later, specified it further, or made a decision depending on it. Reaction alone ("interesting", "makes sense", silence) is `[M]`. When uncertain between two tags, the weaker one wins.
- **Orchestrator phases:** Shishō runs exactly two phases. Phase 0 emits a Pipeline Plan JSON and stops. Phase 2 emits a PSD (wrapped in `<PSD_DOCUMENT>` tags) and stops. No mixing. The handoff bundle is a third, separate invocation.
- **PSD format:** Ten fixed sections. Section numbers and headings never change; empty sections are marked empty, never deleted. Section 9 (Undecided Candidates) is populated from Uchikomi's `[M]` list and is empty when intake was skipped.
- **Wave model:** Strictly sequential, strictly ordered. All enabled Wave 1 agents finish before any Wave 2 agent starts. Reality Checker (fixed) runs after all Wave 2. Disabled agents are skipped, their node retained with `status: skipped`.
- **Thinking traces:** Extracted into `AgentOutput.thinking`, stripped from content before JSON parse, returned to clients via SSE as a distinct event, never forwarded to downstream agents. Degrade gracefully to empty string if the model emits no reasoning blocks; the UI omits the section in that case.
- **Agent sourcing:** Specialist prompts fetched from agency-agents raw GitHub URLs, cached in SQLite with 24h TTL and ETag conditional requests. Stale cache served if GitHub unreachable. Curated 13-agent default roster in `roster/agents.json`; advanced mode lists the full repo via the GitHub tree API.
- **Curated roster & waves:** Wave 1 always — Software Architect, Financial Analyst, UX Researcher, Trend Researcher, Product Manager. Wave 1 conditional — Data Engineer (data-heavy), Game Designer (game), Narrative Designer (game+story). Wave 2 always — Senior Project Manager, Security Engineer, Legal Compliance Checker. Wave 2 conditional — Tool Evaluator. Fixed — Reality Checker.
- **Shishō origin:** Forked from the agency-agents Product Manager agent. ~30% base persona, ~70% new orchestration/synthesis behavior. Attribution blockquote at the top of `prompts/shisho.md`. MIT.
- **Persistence:** SQLite. Sessions persisted before every LLM call. Session stores original source, distilled or raw input, pipeline plan, agent outputs (typed), PSD, domain, tags, finding-severity summary.
- **Pipeline execution transport:** SSE. `POST /api/pipeline/run/:id` opens an event stream emitting pipeline/agent/wave/synthesis/PSD lifecycle events. Clients subscribe and reconnect on drop.
- **Phase 0 and intake are regular POSTs.** Only the wave run is SSE.
- **User modification scope:** Toggle agents on/off, move between Wave 1 and Wave 2, add custom agents, reorder within a wave. Read-only: the Fixed wave (Reality Checker) and Shishō's classification context block.
- **Failure policy:** Specialist agent failures do not abort the pipeline (stored as `status: error` or `partial`, run continues). Only Shishō Phase 0 or Phase 2 failure aborts. Uchikomi failure leaves the session in `intake` with the original input intact and the user free to proceed without distillation.
- **Run durability:** A panel run outlives its client. Each agent result is persisted with its node status in the same transaction as it completes — never buffered to flush at the end. Interruption is treated as expected, not exceptional.
- **Run recovery:** Sessions found in `running` or `synthesis` at server startup are orphaned by definition (single process) and are reconciled to `interrupted`, with any node left at `running` reset to `pending`. `POST /api/pipeline/resume/:id` continues from there, skipping nodes already `done` or `skipped`. Resume is driven by node status; there is no stored cursor. `interrupted` is a recoverable state distinct from `error`, which is reserved for states with no recovery path. A Phase 2 failure is interrupted, not error.
- **Live attach:** `GET /api/pipeline/stream/:id` lets a client subscribe to a run already in flight, because the run survives the client closing and polling is forbidden. Events are not replayed — the persisted session is the record, the stream is the live tail.
- **Session metadata:** Domain and tags assigned by Shishō in Phase 0, persisted to dedicated columns for SQL filtering. Finding-severity summary computed after the panel completes.
- **Custom agents:** Stored in `custom_agents` table. Shishō sees them in its roster reference and may select them. User can force-add any agent regardless of Shishō's choice.
- **Output modes:** PSD always produced. Dojo handoff bundle produced on request, after the PSD exists, as a later epic.
- **Naming:** Project is Kumite. Orchestrator is Shishō (ASCII `shisho` in code and filenames). Intake stage is Uchikomi (`uchikomi`). DB file `kumite.db`. Go module `kumite`. No personal names anywhere in the system.
- **Vector store:** LanceDB, deferred post-V1. Embed agent summaries + PSD sections, not raw chat. Embedding via `nomic-embed-text`.
- **Testing:** `LLM_MOCK=true` swaps the LLM endpoint for a local mock server with fixture responses. In-memory SQLite for tests. Every engine function needs a happy-path and an error-path test. Mock infrastructure is built before the execution engine.

## Open Decision

- **Owner of the Minimum Viable Slice.** No roster agent currently has PSD section 4 as an explicit mandate. Resolve by assigning it to an existing agent through its pipeline-plan rationale, adding a custom agent, or accepting that the section is often empty in V1. Depends on what the upstream agency-agents prompts actually deliver.
