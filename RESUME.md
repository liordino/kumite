# RESUME

If a prior session was interrupted, start here.

## Fast path

1. `dotnet test` — expect **21/21 green, 0 warnings**. If not, the last
   commit regressed something; `git log --oneline` to find it.
2. Read `CHANGELOG.md` (top section + appendix) — 2 minutes, covers
   everything shipped.
3. `git log --oneline` — every commit is atomic and conventional.

## Pending work (in order)

- [ ] Human: paste Ollama Cloud key into `.env` (already scaffolded at
      `KUMITE_BASE_URL=https://ollama.com/v1`; get key at
      https://ollama.com/settings/keys). Board models already filled.
  - [ ] If a model 404s (Ollama retires cloud models), pick a live one:
        `curl https://ollama.com/api/tags`.
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
