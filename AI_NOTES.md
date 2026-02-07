# AI Notes for Palimpsest

This document is tailored for AI agents (and new contributors) to quickly grasp the intent, invariants, and working constraints of this repository.

## 1) One‑minute summary
Palimpsest is a PoC for a low‑code SaaS foundation that treats configuration changes as an incremental computation problem (build system analogy). The system uses an **append‑only Event Log** as the Source of Truth, replays it into a **typed, labeled directed graph**, and computes **impact** (reachability) and **evidence paths** in **O(K)** where K is the size of the affected subgraph. Validation is separate from impact; in the PoC it only checks dangling edges.

## 2) Core model (non‑negotiable)
- **SoT is the Event Log**, not snapshots of state.
- **Graph edges are provider → consumer** so impact = forward reachability.
- **Impact is informational**, not a blocker. Validation can block.
- **Complexity target: O(K)** (affected subgraph), avoid global O(N).

## 3) Domain terms
- **Event Log**: Append‑only list of configuration events (atomic changes).
- **Replay / Projection**: Build a graph from the log (cacheable).
- **Impact**: Nodes reachable from seeds (change origins).
- **Evidence Path**: Shortest path from seed to impacted node.
- **Validation**: Invariant checks (PoC: dangling edges only).

## 4) Event types (PoC)
- NodeAdded / NodeRemoved
- EdgeAdded / EdgeRemoved
- AttrUpdated
- TransactionMarker (TxMarker)

Seeds (Impact):
- NodeAdded/Removed/AttrUpdated → {node}
- EdgeAdded/Removed: usually {to}, but for controls/constrains → {from, to}
- TxMarker → {}

Seeds (Validation):
- NodeAdded/Removed/AttrUpdated → {node}
- EdgeAdded/Removed → {from, to}
- TxMarker → {}

## 5) Labels and semantics
Labels are intentionally small but operationally meaningful:
- uses (data dependency)
- derives (structural ownership)
- controls (behavioral control)
- constrains (validation constraint)

## 6) Code map
- `event.go`: event types, labels, seeds, event log
- `graph.go`: graph structure + mutation during replay
- `replay.go`: log → graph projection
- `impact.go`: BFS impact + evidence paths
- `validation.go`: dangling‑edge checks
- `cmd/demo/main.go`: PoC scenario
- `impact_test.go`: key tests

## 7) Guardrails / anti‑patterns
- Do NOT treat snapshots as SoT.
- Do NOT add DB dependencies to core logic.
- Do NOT compute impact by scanning all nodes.
- Do NOT block on impact alone; only validation can block.

## 8) Suggested next milestones (from docs)
- RFC + Protobuf schema
- Core logic module separation
- Sandbox + speculative computation
- AI simulation + repair plan
- Production: persistence + API + UI

## 9) Quick commands
- Tests: `go test -v`
- Demo: `go run ./cmd/demo`
