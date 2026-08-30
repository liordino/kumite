# HANDOFF

Status after the human-supervised submission session (run #2 closed).
Triage time target: ~5 minutes.

## Where the project stands

- **Engine complete per MVP.md**, build clean, **23/23 tests green**
  (`dotnet test`). New this session: `--auto` unattended gate-bypass
  mode (`06b2c1b`), with integration tests proving zero stdin reads in
  auto mode and blocking gates without it.
- **Run #2 (official) executed and committed**: run id
  `20260830-221525-0390216`, baseline + 7-gated debate on
  `ideas/andante.md`, human decision at every gate. Engine commit
  `7ea42ab`; baseline artifact `wiki/baseline-result-run2.md`
  (`dfd6cdf`). Run #2 executed **supervised**; run #1
  (`20260829-134912-d3bb30c`) executed **unattended** and is relabeled
  (prose only) as a failure-mode exhibit in RESULTS.md.
- Docs repaired: REPRODUCING.md (`--idea ideas/andante.md`, "Two
  runs", `--auto` one-liner), findings.md (key constraint resolved,
  `--auto` sanctioned), RESULTS.md (two labeled sections), CHANGELOG
  appendix wave 3.

## What remains for submission (human)

1. **Score the Run #2 table in RESULTS.md** (baseline column: score
   `wiki/baseline-result-run2.md`; verdict column: the run-2
   verdict/round files; full baseline text was printed to the console
   in-session). Agent filled no score cells — co-sign is yours.
2. **Tag on explicit go**: `git tag run2-official && git push origin
   run2-official` (main push pending too: `git push`).
3. **Video/demo**: record the supervised run narrative per the
   challenge format (run #1 vs run #2 is the story).
4. **Post-tag only**: fix the `ideas/andante.md` typo ("feeds a a
   small") as a separate commit — it is quoted verbatim in run-2
   trajectories and must not change before tagging.

## Disclosures / sharp edges introduced this session

- `.markdownlint.json` + `.markdownlintignore` — out-of-scope infra
  shields stopping the harness markdownlint autofixer from rewriting
  immutable wiki/trajectory artifacts on every git touch. Removable
  after submission if undesired; zero product-code impact.
- `kumite baseline` always writes `wiki/baseline-result.md` — it
  clobbered run #1's committed baseline in the working tree during
  preflight. Run-1 file restored byte-identical; run-2 baseline kept
  as `wiki/baseline-result-run2.md`. Run baselines with care.
- Stray aborted attempt `20260830-221349-8f82944` (no gates reached)
  rode along in commit `7ea42ab`; immutable history, disclosed in
  CHANGELOG wave 3.
- Branch `run1-autonomous` and all run-1/2 wiki + trajectory files are
  immutable history — never rewrite; relabel in RESULTS.md prose only.
- Trajectories use relative paths — run `kumite` from the repo root.
- Ollama Cloud: keys pasted WITHOUT the `oak-` label; model 404s →
  pick live models via `curl https://ollama.com/api/tags`.

## Files to touch first (triage order)

1. `RESULTS.md` — fill Run #2 scores.
2. `wiki/baseline-result-run2.md` + `wiki/verdict-20260830-221525-0390216.md`
   — the two artifacts being scored.
3. `trajectories/20260830-221525-0390216/` — raw evidence.
4. `CHANGELOG.md` appendix wave 3 — session narrative.
5. `src/Kumite.Cli/Gate.cs` / `Program.cs` — the only code delta this
   session (`--auto`).
