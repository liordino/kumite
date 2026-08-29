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
dotnet run --project src/Kumite.Cli -- baseline --idea Andante
dotnet run --project src/Kumite.Cli -- run --board software_squad --idea Andante
```

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
