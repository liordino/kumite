# HANDOFF

Status after autonomous build session. Triage time target: ~5 minutes.

## Where the project stands

- **Waves 0, 0.1, 1 are DONE and committed** (see `CHANGELOG.md`
  appendix for the wave-by-wave log). Build clean, 21/21 tests green
  (`dotnet test`).
- **The engine is complete per MVP.md**: board parser, config/.env,
  LLM client, prompt builder, trajectory logger (full request + raw
  response per call), gates with `[a]pprove/[r]erun/[e]dit` + $EDITOR,
  wiki artifacts, git sink with kill switch, round-1 parallel /
  round-2 sequential / verdict, `--no-round2` kill switch,
  `kumite init|run|baseline` CLI wiring.
- Proven end-to-end against a fake OpenAI-compatible endpoint in
  `tests/Kumite.Tests/EngineIntegrationTests.cs` (7 gate steps per
  run: edit-approval, rerun attempt files, wiki paths, trajectory
  contents).

## What is left (one human input, then ~15 min)

1. **Paste your key into `.env`** (already scaffolded, gitignored):
   create one at <https://ollama.com/settings/keys>, replace
   `KUMITE_API_KEY=oak-PASTE_YOUR_KEY_HERE`. Endpoint and board models
   are already set for Ollama Cloud (`https://ollama.com/v1`; gpt-oss:20b
   / gpt-oss:120b / deepseek-v4-flash:0731 / qwen3.5:397b).
2. Run the measured-improvement experiment (exact commands in
   `REPRODUCING.md`), approve/edit at the 7 gates.
3. Commit `trajectories/{run-id}/`, `wiki/*`, `baseline-result.md`;
   fill the scoring table in `RESULTS.md`.

## Files to touch first

- `MVP.md` — authoritative scope. Everything else interprets it.
- `findings.md` — scope decisions/deviations (net9.0, gate ordering).
- `src/Kumite.Cli/Engine.cs` — the state machine; start reading here.
- `boards/software_squad.yaml` — models preset for Ollama Cloud;
  prompts already agent-drafted.

## Known sharp edges

- YamlDotNet maps empty flow sequences (`[]`) to null, not empty list;
  guarded in `BoardParser.Parse` with regression tests.
- Ollama Cloud retires models over time; if a board model 404s, pick a
  live one via `curl https://ollama.com/api/tags`.
- Windows console: OS-thrown exceptions print localized text (pt-BR);
  runtime environment quirk only.
- Trajectories use relative paths (`trajectories/`, `wiki/`) — run from
  the repo root.
