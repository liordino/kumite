# Results — measured improvement experiment

Score each output 1–5 (higher better). One table per run, published
per MVP.md. Runs 1 and 2 used **different idea inputs** (see
REPRODUCING.md §Two runs); the tables are not directly comparable.

## Run #1 (exhibit)

Idea used: **Andante** (title-only submission — intentionally under-
specified, per MVP.md §measured improvement experiment).

Run #1 was executed **unattended** (gates driven by the autonomous
build session) and is kept as a **failure-mode exhibit**: how the
board behaves when the "idea" is only a product name. **Agent-assessed
scores** — retained as originally recorded, not the official
measurement.

| Dimension             | Baseline (1–5) | Kumite verdict (1–5) |
| --------------------- | -------------- | -------------------- |
| Flaws identified      | 3              | 5                    |
| Spec completeness     | 2              | 4                    |
| Actionability         | 1              | 5                    |
| Perspective diversity | 1              | 5                    |
| **Total**             | **7 / 20**     | **19 / 20**          |

> Scores below are **agent-assessed** from the committed artifacts;
> human review and overrides welcome — the evidence links are inline.

### Scoring rationale (run #1, agent-assessed)

#### Baseline — `wiki/baseline-result.md` (single qwen3.5:397b prompt)

- **Flaws 3** — correctly identified the meta-flaw ("a product name is
  not a specification"), but the analysis terminated there; no flaws
  *within* any interpretation, because none was attempted.
- **Spec completeness 2** — refused to elaborate: "unactionable,
  unassessable". Accurate, terminal, no reconstruction offered.
- **Actionability 1** — no next actions at all; verdict was return-for-
  revision with nothing to return *to*.
- **Perspective diversity 1** — one generic evaluator voice, no
  opposing views, no self-critique.

#### Kumite — `wiki/verdict-20260829-134912-d3bb30c.md` (4-model debate)

- **Flaws 5** — cited, person-specific flaws *inside* the concrete
  interpretation: fabricated user segments ("phantoms"), Likert-survey
  metrics that are self-referential vs. behavioral deltas, value
  outsourced to uncontrolled BPM APIs, and "semantic suicide"
  (Andante = *slow* for a discovery app). Evidence trail:
  `trajectories/20260829-134912-d3bb30c/round-{1,2}/reality_check.md`.
- **Spec completeness 4** — round-2 cross-examination forced a real
  product spec: BPM extraction → tempo-vs-genre scatter plot →
  filter by BPM range → instant preview playback of top-10, plus
  Spotify-API licensing questions (−1: the chief itself flags the
  round-1→round-2 pivot, SaaS→music, as a stability risk).
- **Actionability 5** — chief's exit list is concrete and ordered:
  10 target-user interviews → 100-track BPM accuracy sample → replace
  Likert with task-time deltas → API terms check → signed 20-user
  pilot before any engineering.
- **Perspective diversity 5** — three independent lenses (four
  distinct models across two providers: gpt-oss ×2, deepseek, qwen)
  - adversarial cross-examination + synthesis that states *who was
  more convincing and why* ("reality_check convincingly argued
  segments are phantoms"; "product_owner was more convincing on
  defining a concrete, buildable slice in Round 2").

### Links to artifacts (run #1)

- Baseline: [`wiki/baseline-result.md`](wiki/baseline-result.md)
- Full debate: [`wiki/verdict-20260829-134912-d3bb30c.md`](wiki/verdict-20260829-134912-d3bb30c.md) · [`wiki/round-1-…md`](wiki/round-1-20260829-134912-d3bb30c.md) · [`wiki/round-2-…md`](wiki/round-2-20260829-134912-d3bb30c.md)
- Raw evidence (full request + verbatim response per call):
  [`trajectories/20260829-134912-d3bb30c/`](trajectories/20260829-134912-d3bb30c/)

## Run #2 (official)

Idea used: [`ideas/andante.md`](ideas/andante.md) — full problem/user/
core-loop spec. Executed **fully supervised**: a human decision at
every gate. **This is the official measured-improvement table; scores
are filled in by the human reviewer, not the agent.**

| Dimension             | Baseline (1–5)   | Kumite verdict (1–5) |
| --------------------- | ---------------- | -------------------- |
| Flaws identified      | 4                | 5                    |
| Spec completeness     | 3                | 4                    |
| Actionability         | 4                | 5                    |
| Perspective diversity | 2                | 5                    |
| **Total**             | **13 / 20**      | **19 / 20**          |

> Run #2 scores assigned by the human reviewer after reading both
> artifacts in full (post-tag co-sign; see HANDOFF.md). Agent filled
> no score cells.

### Run #2 artifact links

- Baseline (single call, same idea file): [`wiki/baseline-result-run2.md`](wiki/baseline-result-run2.md)
- Full debate: [`wiki/verdict-20260830-221525-0390216.md`](wiki/verdict-20260830-221525-0390216.md) · [`wiki/round-1-…md`](wiki/round-1-20260830-221525-0390216.md) · [`wiki/round-2-…md`](wiki/round-2-20260830-221525-0390216.md)
- Raw evidence (full request + verbatim response per call):
  [`trajectories/20260830-221525-0390216/`](trajectories/20260830-221525-0390216/)

### Run #2 comparison note (facts, not scores)

- Baseline and debate consumed the **same idea file**
  (`ideas/andante.md`); there is no input-fidelity difference.
- The baseline is a **single qwen3.5:397b call — the same model the
  board uses as Chief** — so any delta between the two outputs is
  attributable to the workflow (multi-persona rounds + gates), not to
  a stronger model.
- Run #2 was executed **supervised** (human decision at all 7 gates);
  run #1 was executed **unattended**.

### Run #2 scoring rationale

**Kumite (19/20):** Flaws 5 — six substantive findings across two rounds,
each cited from the idea text or the emerging proposal: "the batch is a
fiction" (RSS cadence), "cold start is the product," the attention/
retention contradiction, the unmeasurable value hypothesis ("with what
instrument?" — no telemetry in a local-first design), and the
friction-contradiction in the Architect's mitigation. Completeness 4 —
verdict artifact with survived/risks/attributed disagreement and five
ordered actions; cold-start curation remains honestly unresolved.
Actionability 5 — ordered, gated ("before coding the client"), incl. a
5-user concierge test. Diversity 5 — three lenses across three model
families, cross-examination that caught claims the single voice never
had to defend, with explicit attribution of who was more convincing.

### Baseline rationale (single qwen3.5:397b call, same model as the Chief)

- Flaws 4 — five real flaws (relevance paradox, cadence mismatch,
  local-first+LLM trilemma, done-state FOMO, sustainability). Strong,
  but static: it critiques the idea without ever proposing a design,
  so it cannot be caught asserting unmeasurable claims.
- Completeness 3 — a thorough gap table ("a manifesto, not a
  specification") but produces no artifact.
- Actionability 4 — six concrete actions, incl. a heuristic-only
  batching script; strong but unprioritized against a validation path.
- Diversity 2 — one voice performing three perspectives; no counter-
  examination, no attribution, no external check on its own assertions.

Note: baseline = same model as the Chief (qwen3.5:397b). The delta is
the workflow, not the model. Trade-off disclosed: the debate costs
~7 calls vs. 1 — the improvement is bought with human judgment time,
which is the product's thesis.
