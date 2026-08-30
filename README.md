# Kumite

[![License: MIT](LICENSE)](https://opensource.org/licenses/MIT)
![Status](https://img.shields.io/badge/status-Hackathon%20MVP-orange)

**Kumite** is a CLI that stress-tests an idea through a multi-persona
LLM debate — with human approval gates, LLMWiki markdown state, git
versioning, and full trajectory logging. Human-in-the-loop is the
product, not an option.

## Quickstart

```bash
git clone <this-repo> && cd kumite
dotnet restore
cp .env.example .env   # fill KUMITE_API_KEY (+ KUMITE_BASE_URL)
dotnet run --project src/Kumite.Cli -- init
dotnet run --project src/Kumite.Cli -- run --board software_squad --idea "Your idea here"
```

You will be stopped at every step to **[a]pprove / [r]erun / [e]dit**
the agent output in your `$EDITOR`. Nothing reaches the wiki without
your sign-off.

## How it works

```
IDEA → ROUND 1 (personas critique in parallel) → GATE
     → ROUND 2 (personas respond to each other, sequentially) → GATE
     → VERDICT (chief synthesizes the full log) → GATE
     → git add wiki/ + commit
```

One board ships with the MVP: `boards/software_squad.yaml`
(product_owner, architect, reality_check, plus a verdict-only chief).

Every LLM call is logged verbatim — request + raw response — under
`trajectories/{run-id}/`. This is a deliverable, not a debug aid.

Also: `kumite baseline --idea ...` runs a single flat prompt (no
debate) so you can score the measured-improvement table.

## Docs

- [MVP.md](MVP.md) — authoritative scope
- [Kumite.md](Kumite.md) — full design (background, beyond MVP)
- [docs/](docs/) — architecture notes
- [REPRODUCING.md](REPRODUCING.md) — how to reproduce the experiment
- [CHANGELOG.md](CHANGELOG.md) — kata-wave development log
- [trajectories/README.md](trajectories/README.md) — LLM interaction evidence index

## License

[MIT](LICENSE)
