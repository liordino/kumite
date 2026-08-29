# RESUME

If a prior session was interrupted, start here.

## Fast path

1. `dotnet test` — expect **21/21 green, 0 warnings**. If not, the last
   commit regressed something; `git log --oneline` to find it.
2. Read `CHANGELOG.md` (top section + appendix) — 2 minutes, covers
   everything shipped.
3. `git log --oneline` — every commit is atomic and conventional.

## Pending work (in order)

- [ ] Human: fill `.env` (copy `.env.example`; set `KUMITE_BASE_URL`,
      `KUMITE_API_KEY`) and board models in `boards/software_squad.yaml`
      (`SET_ME` / `SET_DIFFERENT_MODEL` / `SET_BEST_MODEL`).
- [ ] `dotnet run --project src/Kumite.Cli -- baseline --idea Andante`
      → review → commit baseline result.
- [ ] `dotnet run --project src/Kumite.Cli -- run --board software_squad --idea Andante`
      → approve/edit at each of the 7 gates → artifacts in `wiki/`,
      trajectories in `trajectories/{run-id}/`.
- [ ] Score both in `RESULTS.md` (table headers already there), commit.
- [ ] Tick off the two pending deliverables in `CHANGELOG.md` wave 2.

## If tests fail

- Board parse failures: check YAML comments/indentation of
  `boards/software_squad.yaml` and the null-flow-sequence guard in
  `BoardParser.Parse` (empty `[]` → null, not empty list).
- Integration tests need a free loopback port (random 20000–65535);
  collisions are unlikely but possible.
