# MVP 仕様

## 最小構成で Day 2 の安心感を実証する

---

## 0. 本文書の位置づけ

本文書は、[charter.md](../theory/charter.md) で定義した理論を実装する最小構成を規定する。目標は「Day 2 の安心感」を最小限のコードで実証すること。

完全な製品ではなく、理論の検証と次フェーズへの学びを得るための PoC である。

---

## 1. 目的

設定変更の**影響範囲**と**理由**を事前に可視化し、変更の安全性を判断可能にする。

具体的には、以下の問いに答える：
- この変更で**何が壊れるか**（Impact）
- **なぜ壊れるか**（Evidence Path）
- **コミットしてよいか**（Validation）

---

## 2. スコープ

### In Scope

| コンポーネント | 機能 |
|--------------|------|
| Event Log | 6 イベント + TxMarker の記録・再生 |
| Graph | イベントログからグラフを構築 |
| Impact | BFS による $\text{Reach}(S)$ 計算 |
| Evidence | 最短証拠パスの生成 |
| Validation | Dangling edge 検出 |
| Cancellation | `context.Context` によるキャンセル |

### Out of Scope（v1 以降）

- 永続化（ファイル / DB）
- Snapshot / Checkpoint
- LRU Cache
- Speculative Computation
- Sandbox / AI Simulation
- Repair Plan
- UI / API

---

## 3. コンポーネント

### 3.1 Event Log

イベントの追記と読み取り。

```go
type EventLog struct { events []Event }

func (l *EventLog) Append(e Event) int   // offset を返す
func (l *EventLog) Get(offset int) Event
func (l *EventLog) Range(start, end int) []Event
```

### 3.2 Graph

イベントログからグラフ状態を構築。

$$
G_r = \text{Replay}([e_0, \ldots, e_r])
$$

```go
func Replay(log *EventLog, upToRevision int) *Graph
func IncrementalReplay(g *Graph, log *EventLog, toRevision int)
```

### 3.3 Impact

Seeds から到達可能なノードを列挙。

$$
\text{Impact}(S) = \{v \in V \mid \exists s \in S,\ s \leadsto v\}
$$

```go
func ComputeImpact(ctx context.Context, g *Graph, seeds []NodeID) *ImpactResult
```

計算量: $O(K)$ where $K = K_V + K_E$

### 3.4 Evidence

影響の理由を最短パスとして提示。

$$
\pi(s, x) = (s = v_0, v_1, \ldots, v_k = x)
$$

```go
func (r *ImpactResult) Explain(nodeID NodeID) string
// → "impacted via: subtotal → calc_tax → tax"
```

### 3.5 Validation

グラフの整合性チェック。

```go
func Validate(ctx context.Context, g *Graph) *ValidationResult
func ValidateSeeds(ctx context.Context, g *Graph, seeds []NodeID) *ValidationResult
```

MVP では dangling edge のみ。将来拡張で必須制約、SCC 閾値など。

---

## 4. Seeds 抽出ルール

| Event | Impact Seeds | Validation Seeds |
|-------|--------------|------------------|
| `NodeAdded(n)` | $\{n\}$ | $\{n\}$ |
| `NodeRemoved(n)` | $\{n\}$ | $\{n\}$ |
| `AttrUpdated(n)` | $\{n\}$ | $\{n\}$ |
| `EdgeAdded(u,v,uses/derives)` | $\{v\}$ | $\{u,v\}$ |
| `EdgeAdded(u,v,controls/constrains)` | $\{u,v\}$ | $\{u,v\}$ |
| `TxMarker` | $\emptyset$ | $\emptyset$ |

---

## 5. 受け入れ基準

### 機能要件

- [x] イベントログにイベントを追加できる
- [x] イベントログからグラフを構築できる
- [x] イベントから Seeds を抽出できる
- [x] Seeds から影響ノードを列挙できる
- [x] 各影響ノードに証拠パスを提示できる
- [x] Dangling edge を検出できる
- [x] 計算をキャンセルできる

### 非機能要件

- [x] 計算量が $O(K)$
- [ ] テストカバレッジ > 80%

---

## 6. 関連ドキュメント

- 実装状況・次のステップ: [roadmap.md](../roadmap.md)
- 理論的背景: [theory/charter.md](../theory/charter.md)
