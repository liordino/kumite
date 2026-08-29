# RESUME

If a prior session was interrupted, start here.

## Fast path

1. `dotnet test` — expect **21/21 green, 0 warnings**. If not, the last
   commit regressed something; `git log --oneline` to find it.
2. Read `CHANGELOG.md` (top section + appendix) — 2 minutes, covers
   everything shipped.
3. `git log --oneline` — every commit is atomic and conventional.

## Status: all MVP waves DONE

Waves 0, 0.1, 1 **and 2** are complete and committed. The measured-
improvement experiment ran live on Ollama Cloud (run
`20260829-134912-d3bb30c`): baseline 7/20 vs debate 19/20 — see
[`RESULTS.md`](RESULTS.md). All MVP.md deliverables exist in this repo.

## Remaining (human, optional)

- [ ] Review the agent-assessed scores in `RESULTS.md` (rationale and
      evidence links inline; override freely).
- [ ] Optional re-runs: same commands, new run-id — old trajectories
      are kept, so runs are comparable.
- [ ] If a model 404s (Ollama retires cloud models), pick a live one:
      `curl https://ollama.com/api/tags`, edit `boards/software_squad.yaml`.

## If tests fail

- Board parse failures: check YAML comments/indentation of
  `boards/software_squad.yaml` and the null-flow-sequence guard in
  `BoardParser.Parse` (empty `[]` → null, not empty list).
- Integration tests need a free loopback port (random 20000–65535);
  collisions are unlikely but possible.
- If a live-model check is ever needed: `kumite baseline` is a single
  cheap call — the connectivity smoke test.
