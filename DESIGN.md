# Kumite — design system

## The tournament ledger — the philosophy

Kumite (組手) is sparring: an idea enters the floor, the panel tests it, what survives is stronger. The interface is a **judges' scoresheet** from that tournament — warm white paper, visible ink, nothing shouting. The reader is the judge at the table: they read findings the way judges read scorecards, in long stretches, under scrutiny. Every element earns its place the way a technique does: no wasted motion, no decoration, no noise.

The product's output is text — findings, classifications, verdicts. The design's job is to make seventy findings scannable in one sitting, not to impress on first glance.

## The tournament register (identity layer, added with the "Tournament Ledger" direction)

The scoresheet alone was too plain — stationery without a tournament. The personality comes from four committed moves, all earned by the sparring frame:

- **A display face.** Archivo (self-hosted, `@fontsource-variable/archivo`), condensed width axis at 92, carries every heading, byline, uppercase micro-line, and primary button. The body stays system-ui — the document voice reads, the ledger voice rules. Weight 700–800 in display positions, never the platform sans as the display voice.
- **The vermilion seal.** A square stamp (`判` on the PSD, `組` in the header) in orange-700 — the hanko on an official record. It marks only the two things that are official: the app's identity and the verdict itself. It is the same color family the theme already reserved for "the bout is live" and Resume; the seal is its printed form.
- **The scorecard division rule.** `rule-division` — a 3px ink rule over a hairline — opens every division: app header, the intake card, the PSD header, each wave, section headers in settings. It is the printed double rule of a scorecard, not decoration.
- **One authored motion.** The live bout: the running agent's spinner ring pulses vermilion (`bout-live`, exponential ease). The PSD section body settles in once on open (`settle`). Nothing else moves; `prefers-reduced-motion` stills both.

Browser surfaces are themed from the palette in `layout.css`: ink-on-paper text selection, stone scrollbars (square, like cut paper), ink caret, visible ink focus rings, tabular numerals in every table and count. Square corners everywhere — a printed sheet has no rounded corners.

## Surfaces

| Token | Class | Role |
| --- | --- | --- |
| Paper | `bg-stone-100` | The page background. The scoresheet sheet. |
| Panel | `bg-white` | Cards, panels, tables: each record with a visible border. |
| Lines | `border-stone-300` | Borders and dividers — **visible**, never faint gray on gray. |
| Ink | `text-stone-900` | Headings, agent names, verdicts. |
| Ink (body) | `text-stone-700` | Reading text. |
| Ink (meta) | `text-zinc-500` | Labels, timestamps, placeholder. |

Contrast rule: a panel is white on stone with a stone-300 border — you can always see the edge of a record. Dark-on-dark was the failure mode of the first theme; the scoresheet never hides its gridlines.

## The primary action

Buttons for the one decisive action on a surface are **ink-on-paper**: `bg-stone-900 text-stone-50 hover:bg-stone-700`. A scoresheet judge commits with a dark pen, not a bright sticker. Outlined buttons (`border-stone-300 text-stone-800 hover:bg-stone-100`) are secondary and tertiary.

Vermilion/orange stays reserved for the **running state** and the primary recovery action (Resume). It appears rarely; when it appears, it means "the bout is live" or "you must act."

## Severity — colored markers, not backgrounds

Severity is a word with a color, not a chip:

- **critical** — `text-red-700`
- **high** — `text-orange-700`
- **medium** — `text-amber-600`
- **low** — `text-stone-500`

Findings sit on plain paper with a `border-l-2 border-stone-300` marker edge. The color accelerates scanning; it never replaces the word, and it never becomes a background the body text fights against.

## Type

System stack. Weight contrast (400/600) does the hierarchy work; monospace for identifiers, counts, and the PSD's frontmatter.

## Editorial conventions

- **Section numbers** in the PSD render as small bordered squares in the margin — the scoresheet's corner boxes, not decorative chips.
- **Attribution is a byline**: the PSD header carries the panel roster as an uppercase micro-line (`PANEL — ENGINEERING-SOFTWARE-ARCHITECT · …`); agent statuses read as bylines (uppercase, tracking-wide), not badges.
- Collapsible sections, open for the first two — the judge flips to a section, not through a scroll.

## Type of the PSD

The PSD renders from markdown through the typed parser in `lib/markdown.ts` — never `innerHTML`, so model output can never inject markup. Sections collapse independently; the raw `.md` downloads from the header.

## Status language

- **pending** — hollow circle
- **running** — spinner + "analyzing…"
- **done** — check
- **error** — cross, red-700, with the error text (failures are information)
- **skipped** — dash

## The monitor

The match structure at a glance: every wave, every competitor, all visible at once in white panels on paper. The current bout: spinner + name, darkest ink. The synthesis: the referee writes, the PSD streams in live.

## Density

The findings are the product. Reading density wins over breathing room — the panel runs long, the reader is committed, and the paper stays quiet. Breathing room is earned by hierarchy (weight, size, the square numbers), not by emptiness.
