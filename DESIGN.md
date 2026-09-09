# Kumite — design system

## The philosophy

Kumite (組手) is sparring: an idea enters the floor, the panel tests it, what survives is stronger. The interface is the arena floor — **dark, disciplined, high-contrast**. Every element earns its place the way a technique does: no wasted motion, no decoration, no noise. The reader is the practitioner, not a visitor.

The product's output is text — findings, classifications, verdicts. Reading density wins over breathing room. The design's job is to make seventy findings scannable in one sitting, not to impress on first glance.

## Surfaces

| Token | Class | Role |
|---|---|---|
| Floor | `bg-zinc-950` | The page background. The arena at night. |
| Mat | `bg-zinc-900` | Cards, panels: the boundary of each bout. |
| Lines | `border-zinc-800` | Borders and dividers: the tatami edges. |
| Ink | `text-zinc-100` | Primary text. |
| Ink (secondary) | `text-zinc-400` | Labels, meta, phase text. |
| Ink (muted) | `text-zinc-500` | Timestamps, placeholder. |

## The accent

One color carries urgency: **vermilion** — the traditional shu-iro of the kumite flag. Reserved for critical severity and the running state. Everything else is monochrome discipline: the reader's eye goes where it must, never where decoration pulls it.

Severity is always paired with a text label — the color accelerates scanning, never replaces the word.

## Type

System stack. Weight contrast (400/600) does the hierarchy work; monospace for identifiers, counts, and the PSD's frontmatter.

## Status language

- **pending** — hollow circle
- **running** — spinner + "analyzing…"
- **done** — check
- **error** — cross, vermilion, with the error text (failures are information)
- **skipped** — dash

## The monitor

The match structure at a glance: every wave, every competitor, all visible at once. The current bout: spinner + name. The synthesis: the referee writes, the PSD streams in live.

## Density

The findings are the product. Reading density wins over breathing room — the panel runs long, the reader is committed, and the interface stays out of the way.
