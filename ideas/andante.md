# Andante — slow-feed reader for RSS

## Problem
Feed readers optimize for throughput, which turns consumption into
doom-scrolling: users either fall behind or binge. Andante's bet:
deliver feeds a a small daily batch, curated for relevance — a digest,
not a stream — with explicit "done for today" closure. No infinite
scroll, no unread counter.

## User
People who want to follow technical blogs and news without the
infinite-feed treadmill. The product is paid for with attention, so
friction must stay near zero.

## Core loop
Subscribe → receive small daily batch → read → mark done.

## Design stance
Local-first, open source. A deterministic core computes everything.
An optional LLM layer only ranks and summarizes — it never acts on
subscriptions or data. The AI provider is the user's choice; nothing
routes to a provider they didn't configure.

## Open questions the author hasn't settled
- What "curated for relevance" means with no interaction history (cold start).
- Whether the LLM layer is needed at all for v1, or if batch + manual
  pins suffice.
