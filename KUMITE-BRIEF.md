# Kumite — Project Brief

> This is the feasibility brief that `/hajime`'s randori grills against to derive `TASKS.md`. It describes what Kumite is, what it must do, what it must not do, and how success is measured. Read alongside `ARCHITECTURE.md`, `DATA_MODEL.md`, `API.md`, `prompts/uchikomi.md`, `prompts/shisho.md`, and `prompts/agent_template.md` — all of which are fixed constraints, not open questions.

## What Kumite Is

Kumite is a self-hosted multi-agent project feasibility consultancy. The user submits a project idea — a one-line concept, a detailed brief, a raw transcript of conversations with AI models, an uploaded document, or an existing Project Summary Document (PSD).

The material optionally passes through **Uchikomi**, an intake stage that distills raw conversation into a clean brief with provenance resolved. Then **Shishō**, the orchestrator, classifies the input, proposes a panel of specialist agents organized into dependency waves, and lets the user modify that panel. The specialists run sequentially, each producing structured findings. A Reality Checker challenges all findings last. Shishō then synthesizes everything into a PSD — a feasibility verdict with section-level attribution to the specialists who surfaced each insight.

The tool exists to answer one question before any code is written: **should this be built, by what reasoning, and what are the risks?** It is an evaluation system, not an execution system. It does not build the project — it decides whether the project is worth building and hands off a starting point if so.

## The Core Loop

1. User submits project input (text, file, transcript, or existing PSD).
2. **Optionally**, Uchikomi distills the input into a clean brief. The user chooses; Kumite explains when each path applies but does not decide.
3. Shishō runs Phase 0: classifies input, runs the 4-Block diagnostic, raises flags, and proposes a pipeline plan (which specialists, which wave, why).
4. User reviews the pipeline in a visual builder — toggles agents, drags between waves, adds custom agents — then confirms.
5. The execution engine runs the panel in waves: Wave 1 (independent analysis), Wave 2 (reactive analysis using Wave 1 outputs), then the fixed Reality Checker.
6. Each specialist produces structured JSON output (summary, typed findings with severity, recommendation, open questions, PSD section contributions).
7. Shishō runs Phase 2: synthesizes all specialist outputs into a PSD with attribution.
8. Optionally, the user requests a Dojo handoff bundle — a `BRIEF.md` + `CONTEXT.md` seed derived from the PSD, ready to feed into Dojo's `/hajime` to build the project.

## The Uchikomi Stage

Uchikomi (打ち込み — the repetitive entry drill practiced before sparring) exists because most raw material arriving at Kumite is a transcript of the user talking to an AI model. In that material, most of the text was written by a model, not the author. A feature a model proposed and the author never engaged with is not a requirement — it is a proposal sitting in a transcript. Without a stage that separates the two, the panel analyzes a project the author never described.

Uchikomi classifies every extracted statement as author-stated, author-confirmed, or model-proposed, and emits a structured brief where only the first two are presented as the project. It does not evaluate — no risk assessment, no feasibility judgment, no scoring, no build order. Anything it concluded would anchor thirteen specialists to a verdict a single model produced alone.

The stage is **optional and user-selected**. It cannot be automatic, because deciding whether input needs distillation is classification, and classification is Shishō's Phase 0, which runs after. The UI presents both paths with a static explanation of when each applies: raw conversation transcripts benefit from distillation; briefs the author wrote directly do not, and passing them through loses the author's own phrasing.

When Uchikomi runs, the original material is preserved in `raw_source` and the distilled brief becomes `raw_input`. When it is skipped, `raw_source` stays null — its absence is the record of which path was taken. This matters for diagnosis: when a PSD comes out wrong, the first question is whether the fault was the panel or the compression, and that is unanswerable without the original.

## The Main User

A solo developer (the author) bringing personal project ideas to reality. Builds across software, SaaS, tools, and games. Runs inference against Ollama Cloud, or any OpenAI-compatible provider, at normal interactive speed. Values rigor, skepticism, and structured output. Wants to trace every insight in the verdict back to the specialist that raised it.

## Success Metric (V1)

V1 works when: a project brief submitted to Kumite produces a complete, well-attributed PSD through a full panel run; the optional Uchikomi stage produces a distilled brief that the user judges faithful to what they actually decided; the user can modify the pipeline before execution; sessions persist and reload with full history intact; a run interrupted mid-panel resumes from where it stopped without losing completed specialist output; and the inference provider is swappable at runtime without restart. The Dojo handoff bundle is a fast-follow, not a V1 gate.

## What Must Be True (Hard Constraints)

- **Self-hosted, provider-agnostic.** The application runs on the user's own machine. Inference targets a configurable OpenAI-compatible endpoint, default Ollama Cloud. Any OpenAI-compatible provider works via settings, switchable at runtime without restart.
- **Backend: Go 1.22 + Chi router. No CGo.** SQLite via `modernc.org/sqlite` only.
- **Frontend: SvelteKit + TypeScript, Tailwind only.**
- **The backend assumes no particular client.** All state lives server-side. No session state in the frontend, no route that returns HTML, no pipeline logic in the client. A TUI is planned (see below) and must be able to consume the same API without backend changes.
- **Structured output via JSON schema constraint** (the `format` parameter) as the primary path, retry-with-correction as fallback, markdown extraction as last resort.
- **Thinking traces** (`<think>` blocks, when the model emits them) preserved per agent, collapsed by default in the UI, stripped before forwarding to downstream agents. Degrades gracefully to empty when absent.
- **Specialist agents sourced from the agency-agents GitHub repo**, fetched at runtime and cached. A curated default roster of 13 agents, with an advanced mode exposing the full repo.
- **Uchikomi and Shishō are prompt artifacts, not code.** Loaded at runtime from `prompts/`, never modified during implementation.
- **Shishō runs in exactly two phases:** Phase 0 (classify + propose pipeline) and Phase 2 (synthesize PSD). Never both in one call. It never performs specialist analysis.
- **Wave execution is strictly sequential and ordered.** Wave 1 completes before Wave 2 starts. Reality Checker is always last.
- **All agent outputs are typed** and parsed into structs before persistence — never stored as raw strings.
- **Sessions persist before LLM calls, not after** — every state is recoverable.
- **Runs are durable and resumable.** A panel run outlives the client that started it. Each agent result is persisted as it completes. A run cut mid-flight leaves valid partial results and can be resumed from the first incomplete agent — never restarted from zero.
- **Pipeline execution streams via SSE.** Clients subscribe; they never poll.

## What This Is NOT (Explicit Exclusions)

- Kumite does NOT execute, build, or write the project's code. It evaluates feasibility only.
- Kumite does NOT replace Dojo. It produces input for Dojo via the handoff bundle.
- Kumite is NOT a generic agent framework. It is a single-purpose feasibility panel.
- Kumite does NOT run agents in parallel. Waves express dependency; execution is sequential by choice, for deterministic ordering and traceable output — not because the provider forces it.
- Uchikomi does NOT evaluate. It distills and separates. Judgment belongs to the panel.
- Kumite does NOT use an ORM, CGo, XML, or any cloud-only dependency.
- The Orchestrator does NOT do specialist work — it classifies, routes, and synthesizes only.
- The PSD synthesis does NOT invent content — sections with no specialist input are marked as such, not padded.
- V1 does NOT ship a CLI or TUI. See below.

## Planned, Not in V1: TUI Client

A terminal client using Bubble Tea is planned as a second interface. It is deliberately out of V1, but the backend must not foreclose it.

The reasoning: Kumite is a developer tool invoked between Dojo sessions, and switching to a browser to paste a brief is friction a TUI would remove. SSE renders naturally in a terminal. The open question is the pipeline builder — drag-and-drop between waves and toggles across thirteen agents becomes keyboard navigation with visual state, which is the most expensive part of a TUI and the least similar to the web version.

That question is answerable only with usage data: after roughly ten real sessions, how often does the user actually modify the proposed panel? If the plan is usually accepted as-is, the builder is not the center of the tool and the TUI becomes the better interface. If it is modified every time, the web version stays primary and the TUI is for dispatch and monitoring.

Because the backend is client-agnostic by constraint, the TUI is a client, not a rewrite. Both interfaces consume the same routes and carry the same information — only the presentation differs.

When the TUI exists, the Uchikomi choice appears twice: as an interactive prompt with the same static explanation the web version shows, and as `--intake` / `--no-intake` flags for non-interactive invocation. Flags carry no room to explain trade-offs, so the reasoning for each path lives in `--help`.

## Output Modes

Kumite produces two artifacts:

1. **The PSD** — always. The feasibility verdict in ten fixed sections: overview, problem and value, feasibility challenge, minimum viable slice, features by dependency, versioning and switches, constraints and scope boundaries, assumptions and gaps, undecided candidates, and questions for the next session. Every significant insight attributed to its source specialist. Section numbers and headings are fixed; an empty section is marked empty, never deleted.

2. **The Dojo handoff bundle** — optional, generated on request after the PSD exists. A `BRIEF.md` (the project reframed as a Dojo feasibility brief) plus a `CONTEXT.md` seed (Glossary · Non-Goals · Decisions) derived from the PSD and panel findings. This is the bridge from Kumite's verdict into Dojo's `/hajime` build cycle. It is a later epic — built after the core panel pipeline works.

## Build Order Intent

The core feasibility panel — input, Phase 0, pipeline builder, wave execution, Phase 2 synthesis, PSD output, session persistence — is the spine and must work end-to-end first. Run durability belongs to the spine, not to polish: the persist-per-agent discipline shapes how the runner is written, and retrofitting it later means rewriting the execution loop. Startup reconciliation and the resume endpoint can follow once the loop is correct. Agent roster and caching, structured output hardening, and the SSE execution monitor are part of that spine.

The Uchikomi stage is a spine-adjacent addition: it can be built after the panel runs end-to-end, because a session that skips intake exercises the whole pipeline without it. Building it early risks debugging two new things at once.

Session metadata and filtering, the advanced full-repo roster, and the Dojo handoff bundle are extensions that come after the spine is proven. Polish and error recovery close it out.

## Reference Documents (Fixed Constraints)

- `AGENTS.md` — repository layout, coding rules, environment variables. Read first.
- `ARCHITECTURE.md` — LLM endpoint, structured output strategy, thinking traces, wave model, SSE protocol, agent cache, intake stage, session metadata, future vector store, custom agents, testing approach.
- `DATA_MODEL.md` — all Go types, SQLite schemas, `roster/agents.json`.
- `API.md` — all HTTP routes with request/response shapes.
- `prompts/uchikomi.md` — the intake distillation prompt. Do not modify during implementation.
- `prompts/shisho.md` — the Shishō orchestrator system prompt (Phase 0 + Phase 2 + handoff). Do not modify during implementation.
- `prompts/agent_template.md` — the specialist agent prompt envelope with runtime assembly notes. Do not modify during implementation.
- `roster/agents.json` — the curated 13-agent default roster (defined inside `DATA_MODEL.md`).

randori grills the design against these. The schemas and routes are decided. What randori resolves is implementation sequencing, wave boundaries, and the open questions that surface when these documents meet real code.

## Known Open Question

**No agent in the roster owns the Minimum Viable Slice.** PSD section 4 asks for the smallest change that runs end to end and produces one real unit of value. Neither Uchikomi (which does not evaluate) nor any of the thirteen specialists has that as an explicit mandate. Software Architect and Senior Project Manager come closest but will not produce it reliably.

Three possible resolutions: assign the mandate to an existing agent via its rationale in the pipeline plan, add a custom agent for it, or accept that section 4 is frequently empty in V1. This must be decided before the first real run, and it depends on what the upstream agency-agents prompts actually deliver — which is knowledge the author has and the specs do not.
