# Palimpsest

**増分計算理論に基づくローコード SaaS 基盤**

---

## 概要

Palimpsest はローコード SaaS の設定変更問題を、ビルドシステムの増分再ビルド問題と同型と捉え、確立された増分計算理論で解く。

### 既存 SaaS の問題

| 問題 | 原因 |
|------|------|
| 変更の影響範囲が不明 | 依存関係が不透明 |
| 障害原因を説明できない | 因果追跡手段なし |
| 変更が恐怖になる | 安全性検証不可 |

### Palimpsest のアプローチ

- **Event Log を Source of Truth に**: 操作履歴から状態を再構築
- **依存グラフで影響を可視化**: 変更前に「何が壊れるか」を提示
- **証拠パスで理由を説明**: 「なぜ壊れるか」を機械的に説明
- **$O(K)$ の増分計算**: 影響範囲のみを計算

---

## Quick Start

```bash
go test -v        # テスト実行
go run ./cmd/demo # デモ実行
go run ./cmd/visualize -mode all # 可視化デモ（Why/Impact/Remove/Scale/Repair/Repair-Cascade/Relation/Future/Bench）
go run ./cmd/visualize -mode bench -bench-nodes 50000 -bench-edges 150000
```

ベンチ結果は `docs/benchmarks.md` に集約。

## Sandbox / Speculative（利用例）

```go
// Snapshot + tail replay で request-local graph を作る
log := palimpsest.NewEventLog()
// ... log.Append(...)

snap := palimpsest.SnapshotFromLog(log, log.Len()-1)
sb := palimpsest.NewSandbox(snap, log, log.Len()-1)

ctx := context.Background()
res := sb.SimulateEvent(ctx, palimpsest.Event{
    Type:   palimpsest.EventAttrUpdated,
    NodeID: "field:order.subtotal",
    Attrs:  palimpsest.Attrs{"type": palimpsest.VString("decimal")},
})

// PreValidate / PreImpact / PostImpact などを参照
_ = res.PreValidate
_ = res.PreImpact
_ = res.PostImpact
```

### 出力例

```
=== Scenario 1: 税率フィールドの型変更 ===
Event: AttrUpdated on field:order.tax_rate
Impact: 7 nodes affected
  - expr:calc_tax: impacted via: field:order.tax_rate → expr:calc_tax
  - field:order.tax: impacted via: field:order.tax_rate → expr:calc_tax → field:order.tax
  ...
```

---

## アーキテクチャ

```
Event Log (SoT)
    │
    │ Replay
    ▼
Graph (Projection)
    │
    │ BFS from Seeds
    ▼
Impact Result + Evidence Path
```

### 核心の数式

グラフ:
$$G_r = (V_r, E_r), \quad E_r \subseteq V_r \times V_r \times L$$

Projection:
$$G_r = \text{Replay}([e_0, \ldots, e_r])$$

Impact:
$$\text{Impact}(S) = \{v \in V \mid \exists s \in S,\ s \leadsto v\}$$

計算量:
$$O(K) \quad \text{where} \quad K = K_V + K_E$$

---

## プロジェクト構造

```
palimpsest/
├── event.go           # イベント型 + Seeds 抽出
├── graph.go           # グラフ構造
├── replay.go          # ログ → グラフ射影
├── impact.go          # BFS + 証拠パス
├── validation.go      # Dangling 検出
├── value.go           # JSON-like Value
├── impact_test.go     # テスト
├── cmd/demo/main.go   # デモ
└── docs/
    ├── theory/
    │   ├── charter.md              # 理論憲章
    │   └── formal-model.md         # 形式モデル
    ├── rfcs/
    │   └── 0001-event-schema.md    # イベントスキーマ
    ├── adr/
    │   ├── 0001-event-log-as-sot.md
    │   ├── 0002-revision-and-simulation.md
    │   ├── 0003-before-after-graph-selection.md
    │   ├── 0004-validation-responsibility.md
    │   ├── 0005-delta-based-apply-and-rollback.md
    │   ├── 0006-immutable-snapshot-and-tail-replay.md
    │   ├── 0007-lazy-evidence.md
    │   └── 0008-expression-front-end.md
    ├── benchmarks.md              # ベンチ結果
    ├── roadmap.md                  # 実装順・進捗
    ├── specs/
    │   ├── mvp.md                  # MVP 仕様
    │   └── phase2.md               # Phase 2 仕様
    └── scenarios/
        └── ai-agent.md             # AI UX シナリオ
```

---

## 理論的基盤

| 理論/システム | 適用 |
|--------------|------|
| [Build Systems à la Carte](https://doi.org/10.1017/S0956796820000088) | 増分計算の形式化 |
| [Adapton](https://doi.org/10.1145/2594291.2594324) | Demand-driven computation |
| [rustc Incremental](https://rustc-dev-guide.rust-lang.org/queries/incremental-compilation-in-detail.html) | Red-green algorithm |
| [Bazel Skyframe](https://bazel.build/reference/skyframe) | 産業規模の増分計算 |
| [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) | ログを SoT とするパターン |

---

## ロードマップ

| Phase | 内容 |
|-------|------|
| PoC | Event Log + Replay + Impact + Validation |
| PoC+ | Delta apply/rollback + Simulate + Snapshot + Bench |
| v1 | Sandbox + Speculative Computation + LRU Cache |
| v2 | AI Simulation + Repair Plan |
| v3 | Production-ready 永続化 + API |

---

## License

MIT
