# Results — measured improvement experiment

Score each output 1–5 (higher better). One table, published per MVP.md.

Idea used: **Andante** (title-only submission — intentionally under-
specified, per MVP.md §measured improvement experiment).

| Dimension             | Baseline (1–5) | Kumite verdict (1–5) |
| --------------------- | -------------- | -------------------- |
| Flaws identified      | 3              | 5                    |
| Spec completeness     | 2              | 4                    |
| Actionability         | 1              | 5                    |
| Perspective diversity | 1              | 5                    |
| **Total**             | **7 / 20**     | **19 / 20**          |

> Scores below are **agent-assessed** from the committed artifacts;
> human review and overrides welcome — the evidence links are inline.

## Scoring rationale

### Baseline — `wiki/baseline-result.md` (single qwen3.5:397b prompt)

- **Flaws 3** — correctly identified the meta-flaw ("a product name is
  not a specification"), but the analysis terminated there; no flaws
  *within* any interpretation, because none was attempted.
- **Spec completeness 2** — refused to elaborate: "unactionable,
  unassessable". Accurate, terminal, no reconstruction offered.
- **Actionability 1** — no next actions at all; verdict was return-for-
  revision with nothing to return *to*.
- **Perspective diversity 1** — one generic evaluator voice, no
  opposing views, no self-critique.

### Kumite — `wiki/verdict-20260829-134912-d3bb30c.md` (4-model debate)

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

## Links to artifacts

- Baseline: [`wiki/baseline-result.md`](wiki/baseline-result.md)
- Full debate: [`wiki/verdict-20260829-134912-d3bb30c.md`](wiki/verdict-20260829-134912-d3bb30c.md) · [`wiki/round-1-…md`](wiki/round-1-20260829-134912-d3bb30c.md) · [`wiki/round-2-…md`](wiki/round-2-20260829-134912-d3bb30c.md)
- Raw evidence (full request + verbatim response per call):
  [`trajectories/20260829-134912-d3bb30c/`](trajectories/20260829-134912-d3bb30c/)
