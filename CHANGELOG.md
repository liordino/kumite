# Changelog

Kata-wave development log. Raw (agent-authored) wave log kept in the
Appendix; the top section is the challenge-format summary.

## [Unreleased]

### Added

- Repository bootstrap: solution skeleton (console + xUnit), GitHub
  surface (README, LICENSE, .gitignore, .env.example), docs/ stubs.
- boards/software_squad.yaml with the four persona prompts (agent-
  drafted) and the `verifierdict`-typo canary fixed to `verdict`.
- Engine: ROUND1 parallel (Task.WhenAll) → gates → ROUND2 sequential →
  gates → VERDICT (chief) → gate → git commit; per-call trajectory
  logging under `trajectories/{run-id}/{round}/{persona}.md` including
  rerun `.attemptN` files.
- CLI: `init`, `run --board ... --idea ... [--no-round2]`, `baseline
  --idea ...`.
- Ollama Cloud preset: `.env.example` example (`https://ollama.com/v1`),
  board models filled with live cloud models (gpt-oss:20b / gpt-oss:120b
  / deepseek-v4-flash:0731 / qwen3.5:397b — architect ≠ reality_check
  family). Local `.env` scaffolded, gitignored. HttpClient timeout 5 min
  (thinking models exceed the 100 s default).
- xUnit suite: board parsing, prompt builder, wiki paths, config, idea
  source, plus full-loop integration tests against a fake
  OpenAI-compatible endpoint (21 tests, all green).

### Fixed

- YamlDotNet deserializes empty flow sequences to null → friendly board
  errors instead of NRE; network errors surface as clean `error:` lines.

### Changed

- Round-1 gates walked sequentially in board order after concurrent LLM
  calls (console readability); see findings.md.

## Notes

- Target is net9.0 (only SDK on this machine); MVP.md says net8.0 —
  logged as deliberate deviation in findings.md.
- Per the MVP kill switch, `--no-round2` ships ROUND1 + VERDICT as a
  complete agent loop.

## Appendix — raw wave log

- **Wave 0** — repo + solution bootstrap; dojo-check (build + tests green).
- **Wave 0.1** — persona prompts drafted into software_squad.yaml;
  `verifierdict` canary retired (→ `verdict`). Board parse test guards
  the four prompts (no TODOs, ≤170 words, contract keywords present).
- **Wave 1** — board parser → api client + prompt builder → engine
  (round 1 parallel + trajectory logging) → gates + wiki + git commits →
  round 2 sequential → verdict → CLI wiring → baseline command.
  Integration tests prove the whole loop against a fake endpoint.
- **Wave 2 (open)** — real-model run pending human `.env` key only
  (endpoint + models now pre-configured for Ollama Cloud);
  trajectories/ deliverable + RESULTS.md table follow that run.
