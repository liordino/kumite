# Reproducing Kumite

## Setup

```bash
git clone <this-repo> && cd kumite
dotnet restore
cp .env.example .env
# Edit .env: set KUMITE_API_KEY (and KUMITE_BASE_URL if not OpenRouter).
dotnet run --project src/Kumite.Cli -- init
```

## Run

```bash
# Measured-improvement experiment:
dotnet run --project src/Kumite.Cli -- baseline --idea ideas/andante.md
dotnet run --project src/Kumite.Cli -- run --board software_squad --idea ideas/andante.md
```

## Two runs

- **Run #1** (`20260829-134912-d3bb30c`) — title-only stress test:
  the idea input was the bare word "Andante". Kept as a failure-mode
  exhibit (the debate had to invent the product from a name; see the
  run-1 section in RESULTS.md and its trajectories).
- **Run #2** — the official measured-improvement run: a real,
  fully-specified idea file (`ideas/andante.md`), executed fully
  supervised (human decision at every gate). Scores in RESULTS.md.

`kumite run --auto` is available as an unattended smoke test /
demonstration mode (auto-approves every gate, identical artifacts and
git commits) — **not used for the published measurement**: run #2 was
executed supervised, run #1 unattended.

At every gate: `[a]pprove` writes wiki files and continues, `[r]erun`
re-runs the current step (old trajectory kept), `[e]dit` opens the
draft in `$EDITOR`.

Ollama Cloud gotcha: API keys from ollama.com/settings/keys are
accepted WITHOUT the `oak-` label shown on the page — paste the bare
token, or every call returns 401 Unauthorized.

## Expected artifacts

```text
wiki/idea-{run-id}.md
wiki/round-1-{run-id}.md
wiki/round-2-{run-id}.md
wiki/verdict-{run-id}.md
trajectories/{run-id}/{round}/{persona}.md   # full request + raw response
baseline-result.md                            # from kumite baseline
RESULTS.md                                    # 1–5 scoring table
```

`trajectories/` from at least one full run is committed to this repo.
