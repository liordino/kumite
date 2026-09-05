<BRIEF_DOCUMENT>

# Field Ledger — Project Brief

## What Kumite Is

A phone tool for logging field observations offline, with CSV export.

## The Core Loop

Log an observation in the field → stored locally → exported as CSV.

## The Main User

The author, logging observations in the field.

## Success Metric

Not stated in input.

## Hard Constraints

- Works offline (Software Architect)

## Explicit Exclusions

- Sync engine for v1 (author decision)

## Build Order Intent

Local store first; sync deferred to a later slice (Software Architect).
</BRIEF_DOCUMENT>

<CONTEXT_DOCUMENT>

# CONTEXT

## Glossary

- **Observation** — a single logged field entry.
- **Export** — CSV output of stored observations.

## Non-Goals

- No sync engine in v1.

## Decisions

- SQLite-compatible local storage (Software Architect).
- CSV export with header row (author decision).
</CONTEXT_DOCUMENT>
