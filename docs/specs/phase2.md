# Phase 2 Spec: Sandbox + Speculative + Cache

## 0. 目的

Phase 2 の目的は、**安全に「仮説イベント」を適用し、影響/検証を返せる**こと。
UIやAIに接続するための「事前確認API」を整える。

---

## 1. 主要コンポーネント

### 1.1 Sandbox

- **共有ミュータブル Graph は使わない**
- Snapshot + tail replay で request-local graph を構築
- SimulateEvent / SimulateTx を提供

```go
snap := SnapshotFromLog(log, log.Len()-1)
sb := NewSandbox(snap, log, log.Len()-1)

res := sb.SimulateEvent(ctx, Event{Type: EventAttrUpdated, NodeID: "field:order.subtotal"})
```

### 1.2 Speculative Simulation

- `SimulateEvent` / `SimulateTx` は
  - PreImpact
  - PreValidate
  - Apply → PostValidate → PostImpact
  - Rollback
 という流れで実行

> 影響は「情報提供」、検証は「拒否判定」

### 1.3 Snapshot + Tail Replay

- Snapshot は immutable
- request-local graph は `ReplayFromSnapshot` で生成
- Snapshot は LRU cache で再利用可能

### 1.4 LRU Cache

- `SnapshotCache` により snapshot を再利用
- Snapshot は immutable として共有

---

## 2. 公開API（Phase 2）

```go
// Snapshot
SnapshotFromLog(log, revision) *Snapshot
SnapshotFromGraph(g) *Snapshot
ReplayFromSnapshot(snap, log, rev) *Graph

// Sandbox
NewSandbox(snapshot, log, revision) *Sandbox
(*Sandbox).SimulateEvent(ctx, e) *SimulationResult
(*Sandbox).SimulateTx(ctx, events) *SimulationTxResult

// Cache
NewSnapshotCache(capacity int) *SnapshotCache
(*SnapshotCache).Get(rev int) (*Snapshot, bool)
(*SnapshotCache).Put(snap *Snapshot)
```

---

## 3. 安全性の前提

- Snapshot は immutable であり、共有して良い
- Graph は request-local としてのみ利用
- SimulateEvent/Tx は Graph を一時的に変更するため
  **共有Graphに対しては排他が必要**

---

## 4. 期待される効果

- 保存前に「何が壊れるか」を提示
- 「なぜ壊れるか」を Evidence で説明
- ValidateEvent を同期ゲートとして利用

---

## 5. 未実装（Phase 3以降）

- AI Simulation / Repair Plan
- UI 統合
- 永続化（DB / Snapshot storage）

---

## 6. References

- ADR-0002: Revision と Simulation
- ADR-0005: Delta apply/rollback
- ADR-0006: Immutable snapshot + tail replay
- ADR-0007: Lazy evidence
- ADR-0008: Expression front-end (MVP spec)
