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

## What is left (blocked on one human input, then ~15 min)

1. **`.env` with a real key** — no `KUMITE_API_KEY` exists in this
   environment, so the real-model loop has never executed. Copy
   `.env.example` → `.env`, set `KUMITE_BASE_URL` + `KUMITE_API_KEY`.
2. **Fill board models** — `boards/software_squad.yaml` still has
   `SET_ME` / `SET_DIFFERENT_MODEL` / `SET_BEST_MODEL` placeholders
   (deliberate human review point; reality_check must differ from
   architect). Review persona prompts at the same time.
3. Run the measured-improvement experiment (exact commands in
   `REPRODUCING.md`), approve/edit at the gates.
4. Commit `trajectories/{run-id}/`, `wiki/*`, `baseline-result.md`;
   fill the scoring table in `RESULTS.md`.

## Files to touch first

- `MVP.md` — authoritative scope. Everything else interprets it.
- `findings.md` — scope decisions/deviations (net9.0, gate ordering).
- `src/Kumite.Cli/Engine.cs` — the state machine; start reading here.
- `boards/software_squad.yaml` — prompts drafted, models pending.

## Known sharp edges

- YamlDotNet maps empty flow sequences to null (fixed + regression-
  guarded, but remember for new YAML fields).
- Windows console: OS-thrown exceptions print localized text (pt-BR);
  runtime environment quirk only.
- Trajectories use relative paths (`trajectories/`, `wiki/`) — run from
  the repo root.
