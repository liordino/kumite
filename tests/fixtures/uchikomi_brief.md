---
title: "Field Logger"
input_language: "English"
input_maturity: "raw"
source_documents: [1]
learning_intent: null
distilled_by: "uchikomi"
---

# Brief: Field Logger

## 1. Overview

A phone tool for logging field observations offline, with CSV export.

**Main user:** The author, logging observations in the field.
**Core loop:** Unknown — not stated in input.

## 2. What the Author Wants

- Offline observation logging with CSV export `[A]`

## 3. Constraints

**Hard:**

- Works offline `[A]`

**Preferences:**

- CSV export `[A]`

## 4. Declared Exclusions

- Sync engine for v1 — author chose offline-only for now `[A]`

## 5. Declared Success Signal

Not stated in input.

## 6. Features Described

- **Observation logging** `[A]`
  - On the phone, offline

## 7. Undecided Candidates `[M]`

- CRDT-based sync engine — proposed by a model; author said "makes sense" and moved on
- ML-based tagging — proposed by a model; never engaged with

## 8. Contradictions

- Earlier: sync was discussed → Later: offline-only for v1

## 9. Undefined Terms

- "fast" — used without a concrete definition.

## 10. Gaps

- No target platform stated
- No data schema
