# Evidence map — LLM interactions

## Product runtime (the deliverable's own agents)

- Run #2 (official, SUPERVISED — human at every gate):
  trajectories/20260830-221525-0390216/ — 7 LLM calls, full verbatim
  request + raw response each (round-1 x3, round-2 x3, chief).
  Models: gpt-oss:20b, gpt-oss:120b, deepseek-v4-flash:0731
  (deliberately different family for the adversary), qwen3.5:397b
  (chief). Human gate decisions at each step; wiki/ artifacts
  committed by the engine itself (7ea42ab).
- Run #1 (title-only stress test, UNATTENDED):
  trajectories/20260829-134912-d3bb30c/ — same logging contract;
  preserved as the failure-mode exhibit (see RESULTS.md).
- Stray aborted attempt 20260830-221349-8f82944 committed as-is —
  disclosed, not rewritten (CHANGELOG wave 3).

## Development sessions (how this product was built)

- Per-wave log with commit hashes: CHANGELOG.md appendix
  (wave 0/0.1/1 = autonomous overnight build with divergence halts;
  wave 2 = doc repair + --auto TDD + run #1; wave 3 = --auto
  productization, run #2, incidents; wave 4 = scoring + evidence map).
- Human checkpoints: every wave approval-gated (HANDOFF.md,
  findings.md, RESUME.md, learning-log).
- The solution video captures live human gates during the supervised
  run.

## Integrity verification

- Run artifacts committed by the engine itself (GitSink) — run #2
  artifact commit 7ea42ab; run-1 baseline preserved byte-identical
  after the clobber incident.
- Branch run1-autonomous = exact post-build state before run #1.
- Tag run2-official = reviewed, scored state of run #2.
- Byte-level integrity of trajectories/ and wiki/ verified at Wave C.
