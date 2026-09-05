# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

SvelteKit + TypeScript, Tailwind only. Decided in the repository's fixed constraints (CONTEXT.md "Decisions", AGENTS.md "Stack") — not delegated, not an open question.

## Users

A solo developer reading dense panel output, often across long sessions, judging findings rather than being persuaded.

## Product Purpose

Kumite is a self-hosted multi-agent project feasibility consultancy. The user submits a project idea; an optional intake stage (Uchikomi) distills raw material with provenance resolved; an orchestrator (Shishō) classifies the input and proposes a specialist panel; the panel runs in waves and produces structured findings; synthesis produces a Project Summary Document — a feasibility verdict with section-level attribution to the specialist that surfaced each insight. Kumite evaluates; it does not build. Success for V1: a submitted brief produces a complete, well-attributed PSD through a full panel run, with sessions persisting and resuming intact.

## Positioning

Traceability: every significant insight in the verdict carries the specialist that surfaced it, so the user can trace any claim back to its source. The artifact is the PSD, not chat.

## Operating Context

Surface mode: operate. Kumite is an internal tool where design serves the task, not attention. Not a landing page. Runs on the user's own machine against a configurable OpenAI-compatible provider; used in long, focused sessions between build cycles.

## Capabilities and Constraints

- Tailwind classes only — no other styling libraries, no inline styles.
- The backend is client-agnostic: no session state in the frontend, no route returns HTML, no pipeline logic in the client. A planned TUI client must consume the same API unchanged.
- Dense panel output: typed specialist findings, thinking traces collapsed by default, PSD sections with attribution.
- Sessions persist server-side and reload with full history; runs survive their client and resume from the first incomplete agent.

## Brand Commitments

- Voice: sober, precise, no marketing language.
- Anti-references (binding): hero eyebrows, gradient CTAs, italic serif display, nested cards, decorative iconography.

## Evidence on Hand

None yet — greenfield. The V1 acceptance criteria are defined in KUMITE-BRIEF.md ("Success Metric (V1)"). Do not fabricate testimonials, benchmarks, or users beyond that document.

## Product Principles

Derived from the confirmed brief (labeled as derived; not user-interviewed):

1. Evaluation, not execution — the tool decides whether a project is worth building; it never builds it.
2. Traceability over persuasion — every claim in the verdict carries its source; the user judges, the tool informs.
3. Compression must be honest — when input is distilled, the original is preserved and both versions stay inspectable.
4. Sober density — design serves a reader judging findings across long sessions; clarity comes from structure, not decoration.
