# Kumite — Hackathon MVP Spec (Agentic Workflows Challenge)

> Scope subset of the full Kumite design (see Kumite.md). Ships in ~2 days.
> Stack: C# (.NET 8 console). Rewrite in Rust/Go later — this is an experiment.

## Goal
A CLI that stress-tests an idea through a multi-persona debate with human
approval gates, LLMWiki markdown state, git versioning, and full trajectory
logging. Human-in-the-loop is the product, not an option.

## Stack rules
- .NET 8 console, single project, xUnit tests
- YamlDotNet (board templates), System.Text.Json + HttpClient (API)
- OpenAI-compatible endpoint: OPENROUTER_API_KEY + KUMITE_BASE_URL from .env
- Git via shelling out to the `git` CLI. No git library.
- Minimum dependencies otherwise. YAGNI everywhere.

## The product (state machine)
IDEA → ROUND1 (personas critique, parallel, Task.WhenAll)
     → GATE  [a]pprove / [r]erun / [e]dit (open $EDITOR)
     → ROUND2 (personas respond to each other, sequential)
     → GATE
     → VERDICT (Chief synthesizes the full log)
     → GATE
     → git add wiki/ + commit

## Board template
boards/software_squad.yaml defines personas + rounds. One board ships in MVP.
- personas: product_owner, architect, reality_check, chief (verdict-only)
- Round 1: first three in PARALLEL (Task.WhenAll)
- Round 2: same three SEQUENTIALLY, each sees the idea + all round-1
  outputs + prior round-2 responses
- Verdict: chief sees everything, writes the final artifact
- system prompts are drafted by the agent into the YAML, reviewed at the gate

## Files each run produces (paths are part of the spec)
wiki/idea-{run-id}.md          the original idea, verbatim (written once)
wiki/round-1-{run-id}.md       updated per approved gate
wiki/round-2-{run-id}.md
wiki/verdict-{run-id}.md       final markdown artifact
trajectories/{run-id}/{round}/{persona}.md
                               FULL request + raw response, every call,
                               no exceptions. This is a deliverable.

## CLI
kumite init                                  # scaffold wiki/, boards/, .env.example, .gitignore
kumite run --board software_squad --idea "<text-or-file-path>"
kumite baseline --idea "<text-or-file-path>" # single LLM prompt, no debate

Gate behavior:
- [a]pprove → write wiki file(s), log trajectories, continue
- [r]erun  → re-run current step (new trajectory entry, old kept)
- [e]dit   → open the draft in $EDITOR, continue with edited content

## Measured improvement experiment
1. kumite baseline --idea Andante  → save as baseline-result.md
2. kumite run --board software_squad --idea Andante → verdict-{run}.md
3. Score both 1–5 on: flaws identified · spec completeness ·
   actionability · perspective diversity. One table, published.

## Deliberate non-goals (deadline kill switches)
- NO web/Tauri UI, no MemPalace, no autoresearch loops, no caveman
- DO NOT call an LLM to compute anything that code can compute
- If Round 2 proves fiddly → ship ROUND1 + VERDICT only (still a
  complete agent loop), note it in the changelog
- If git integration fights back → wiki only + print suggested
  `git add/commit` commands at each gate
- One board only.

## Deliverables this repo must end with
- src + CHANGELOG.md (kata-wave log, LLM-translated into the challenge
  format; raw wave log kept as appendix)
- REPRODUCING.md (clone → dotnet run -- init/checkout → set .env → the
  exact commands → expected artifacts)
- trajectories/ from at least one full run (committed)
- baseline + kumite result tables (committed)
