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

## Status: COMPLETE — all MVP deliverables shipped

Waves 0, 0.1, 1, 2 are done and committed. The measured-improvement
experiment ran live on Ollama Cloud (run `20260829-134912-d3bb30c`):
baseline **7/20** vs debate **19/20**, scored in `RESULTS.md` with
evidence links. All MVP.md deliverables checked:

- src + CHANGELOG — done
- REPRODUCING.md — done (incl. the Ollama `oak-` key gotcha)
- trajectories/ from a real full run — committed (7 calls, full
  request + verbatim response each)
- baseline + kumite result tables — RESULTS.md

Human follow-ups (optional): review agent-assessed scores in
RESULTS.md; sharpen persona prompts/models at a future gate (edit the
board; no rebuild needed).

## Files to touch first

- `MVP.md` — authoritative scope. Everything else interprets it.
- `findings.md` — scope decisions/deviations (net9.0, gate ordering).
- `src/Kumite.Cli/Engine.cs` — the state machine; start reading here.
- `boards/software_squad.yaml` — models preset for Ollama Cloud;
  prompts already agent-drafted.
- `RESULTS.md` — the published experiment table.

## Known sharp edges

- YamlDotNet maps empty flow sequences (`[]`) to null, not empty list;
  guarded in `BoardParser.Parse` with regression tests.
- Ollama Cloud retires models over time; if a board model 404s, pick a
  live one via `curl https://ollama.com/api/tags`.
- Ollama keys: paste WITHOUT the `oak-` label (401 otherwise) —
  documented in REPRODUCING.md.
- Windows console: OS-thrown exceptions print localized text (pt-BR);
  runtime environment quirk only.
- Trajectories use relative paths (`trajectories/`, `wiki/`) — run from
  the repo root.
