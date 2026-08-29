# Findings & Scope Notes

Anything that exceeds MVP.md scope is recorded here as a note — NOT
implemented. Items are ordered newest-first inside each section.

## Deviations from MVP.md (deliberate, noted per spec)

- **net9.0 target instead of net8.0** — the build machine has only the
  .NET 9 SDK/runtime; SDK 9 targets net9.0 naturally. No MVP feature
  depends on 8-vs-9. Revisit if a contributor wants strict net8.0
  (TargetFramework swap only; no API used is 9-specific).
- **Parallel-round gating order**: all round-1 LLM calls fire
  concurrently (Task.WhenAll), but the human gates are then walked
  sequentially in board order. Interleaved interactive gates from
  concurrent tasks would garble the console. Trajectories keep per-call
  files; spec's "parallel" (round-1 parallelism of LLM calls) holds.

## Notes (possible future scope — NOT implemented)

- `--dry-run` flag with a scripted fake endpoint would let CI exercise
  the full loop; today the fake endpoint lives only in the xUnit
  integration tests (EngineIntegrationTests). Fine for MVP.
- Board model placeholders (`SET_ME`, `SET_DIFFERENT_MODEL`,
  `SET_BEST_MODEL`) are intentional human review points, per the board
  header comment. They fail at first LLM call with a clear message from
  the provider — acceptable. A preflight validator could be a wave later.
- Multi-board support (`boards/` directory listing command) — MVP says
  one board only.
- Round-2 skip is exposed as `--no-round2` per the MVP kill switch; a
  board-level flag instead would be more declarative.
- `kumite init` writes a placeholder board only when boards/ has no
  yaml — avoids clobbering the shipped software_squad board.
- The measured-improvement scoring table is a human task (RESULTS.md);
  no code automates scoring (aligned with "do NOT call an LLM to
  compute anything code can compute" and YAGNI).

## Environment constraints encountered

- No `OPENROUTER_API_KEY` present in this environment → the full
  real-model run (trajectories/ deliverable) must be executed by a
  human with a `.env` key. Engine loop fully proven via fake-endpoint
  integration tests (7/7 gate steps: see EngineIntegrationTests).
- Windows locale → runtime exception text prints in Portuguese; our
  own messages are English (conventional-commits/repo rule holds for
  committed files).
