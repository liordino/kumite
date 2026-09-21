# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] — 2026-09-21

First published release. Everything listed is new since project inception —
there is no previous tag to diff against.

### Added

- Feasibility panel pipeline: Shishō Phase 0 classifies input and proposes a
  specialist panel organized into dependency waves; the user edits the panel
  in a visual builder; specialists run in strict wave order and produce
  structured JSON findings; Shishō Phase 2 synthesizes a Project Summary
  Document (PSD) with section-level attribution.
- Uchikomi intake stage (optional, user-selected): distills raw AI-conversation
  transcripts into a clean brief, classifying every statement as
  author-stated, author-confirmed, or model-proposed. Original material is
  preserved as `raw_source`; its absence records that intake was skipped.
- Fixed Reality Checker wave that challenges all specialist findings last.
- Durable, resumable runs: every agent result is persisted in the same
  transaction that marks its node done; orphaned runs are reconciled to
  `interrupted` on startup; resume is driven by node status, never a cursor.
- SSE execution monitor with live per-agent state, thinking traces collapsed
  by default, and a pause-on-specialist-fail run mode.
- Dojo handoff bundle: a `BRIEF.md` + `CONTEXT.md` seed derived from the PSD,
  ready for a downstream build cycle.
- Agent roster sourced from the agency-agents GitHub repo, fetched and cached
  at runtime, with a curated 13-agent default roster and an advanced
  full-repo mode.
- Structured output hardening: JSON schema constraint as primary path,
  retry-with-correction as fallback, markdown extraction as last resort.
- Sessions: full CRUD, persisted history, metadata and filtering.
- Frontend: SvelteKit + TypeScript + Tailwind — intake, pipeline builder,
  live execution view, PSD scoresheet (light editorial theme), settings.
- Provider-agnostic LLM egress (`engine/llm.go` is the single egress point),
  configurable at runtime from the settings panel; default Ollama Cloud.
- Mock LLM provider for tests; full suite runs with `LLM_MOCK=true`.
- Diagnostic pprof listener on 127.0.0.1:3002.
- Release tooling: `bin/release <major|minor|patch>` — semver release
  entrypoint with loud preconditions (integration line, clean tree, tag
  collisions, tree-sealed proof) and a printed fact sheet; publishing stays
  the human's act. `scripts/dojo-check.sh` compiles, vets, tests, and seals
  the proof to the exact tree; a failed run deletes the proof and records the
  real exit status. `scripts/release-asserts.sh` seeds every violation
  (10/10 proven).

### Fixed

- Wave arrays and agent findings are always arrays, never null — builder
  crashes on the Field Ledger and AgentCard null-slice crashes.
- Fail loudly on non-JSON 2xx API responses instead of silently mishandling.

[0.2.0]: https://github.com/liordino/kumite/releases/tag/v0.2.0