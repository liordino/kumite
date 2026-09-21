# Kumite

A self-hosted multi-agent feasibility consultancy. You bring a project idea — a one-line concept, a written brief, a transcript of your conversations with AI models, or an existing plan document. Kumite puts it in front of a panel of AI specialists, lets them challenge each other, and hands you a verdict: should this be built, by what reasoning, and what are the risks?

It evaluates. It does not build.

## How it works

1. **Intake (Uchikomi, optional).** Most project material is a transcript of you talking to a model, which means most of it was written by a model. Uchikomi classifies every statement as author-stated, author-confirmed, or model-proposed, and distills the transcript into a brief where only your decisions are presented as the project. The original is preserved; the choice to distill is always yours.
2. **Phase 0 (Shishō).** The orchestrator classifies the input, raises flags, and proposes a panel of specialist agents organized into dependency waves. You edit the panel before anything runs: toggle agents, move them between waves, add your own.
3. **The panel runs.** Wave 1 analyzes independently. Wave 2 reacts to Wave 1's findings. A fixed Reality Checker challenges everything last. Each specialist produces structured findings with severity and a recommendation, persisted as it completes. An interrupted run resumes from the first incomplete agent; nothing finished is ever lost.
4. **Synthesis (Phase 2).** Shishō merges everything into a Project Summary Document: ten fixed sections, every significant insight attributed to the specialist that surfaced it.
5. **Handoff (optional).** If the verdict says build, Kumite can produce a `BRIEF.md` plus a `CONTEXT.md` seed, ready to feed a build cycle.

## Running it

Requirements: Go 1.22+, Node (for the frontend), and an OpenAI-compatible inference endpoint. Default is Ollama Cloud; any OpenAI-compatible provider works and can be swapped at runtime from the settings panel, without a restart.

```bash
# Backend
go run main.go

# Frontend (separate terminal)
cd client && npm install && npm run dev
```

The backend serves the API on port 3001; the frontend dev server proxies to it. Configuration lives in the `app_config` table and is edited through the settings panel; environment variables only seed the first run. See `ARCHITECTURE.md` for the full picture.

## Tests

```bash
go test ./...
LLM_MOCK=true go test ./...   # no real provider needed
```

## Layout

The repository docs are the spec: `KUMITE-BRIEF.md` (what it must and must not do), `ARCHITECTURE.md` (waves, SSE, structured output), `DATA_MODEL.md` (types and schemas), `API.md` (routes), `AGENTS.md` (coding rules for agents working in this repo). The LLM prompt artifacts live in `prompts/`.

## Design notes

- Backend is Go with SQLite (no CGo), no ORM; all LLM calls go through a single engine module.
- Structured output comes from JSON schema constraint first, with retry and markdown extraction as fallbacks.
- All state is server-side. The web client is one consumer of the API; a terminal client is planned and must work without backend changes.
- Pipeline execution streams over SSE. Clients subscribe, they never poll.

## Status

V1 is functional: the full pipeline runs end to end, sessions persist and resume, and the PSD is attributed section by section. See `CHANGELOG.md` for what shipped and when.