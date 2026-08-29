# Architecture (MVP)

Single console project (`src/Kumite.Cli`), xUnit tests
(`tests/Kumite.Tests`). Dependencies: YamlDotNet (board templates),
System.Text.Json + HttpClient (OpenAI-compatible API). Git via
shelling out to the `git` CLI. YAGNI everywhere.

## Components

| Component | Responsibility |
|---|---|
| `BoardParser` | Load + validate `boards/*.yaml` into `Board` records |
| `IdeaSource` | Accept idea as literal text or file path |
| `LlmClient` | POST chat-completions to `KUMITE_BASE_URL`; returns raw JSON |
| `PromptBuilder` | Build per-persona user prompts per round (idea + prior outputs) |
| `Engine` | State machine: ROUND1 (parallel) → GATE → ROUND2 (sequential) → GATE → VERDICT → GATE → git commit |
| `Gate` | Console UI: `[a]pprove / [r]erun / [e]dit` ($EDITOR) |
| `TrajectoryLogger` | Write full request + raw response to `trajectories/{run}/{round}/{persona}.md` — every call, no exceptions |
| `Wiki` | Write `wiki/*-{run-id}.md` artifacts |
| `GitSink` | `git add wiki/ trajectories/` + commit after final gate |

Config: `.env` (`KUMITE_BASE_URL`, `KUMITE_API_KEY`), loaded without
extra dependencies.

## Kill switches (from MVP.md)

- Round 2 fiddly → ship ROUND1 + VERDICT only, note in changelog.
- Git fights back → wiki only + print suggested `git add/commit`.
- One board only. No UI. No LLM-for-what-code-can-compute.
