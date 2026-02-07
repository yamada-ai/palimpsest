# ADR-0005: Delta ベースの適用とロールバック

## Status

Accepted

## Context

PoC/MVP では `SimulateEvent` が安全性のために Graph を `Clone()` してからイベントを適用している。しかし、この方式は **O(N+M)** のコストが毎回発生し、Enterprise 規模ではボトルネックになり得る。

一方で「適用 → 検証 → 差し戻し」を低コストに行うには、イベント適用による変更を **差分（Delta）として明示的に保持**するのが有効である。

## Decision

**Graph の適用は Delta を返す方式に移行する。**

- `ApplyEvent(g, e) -> Delta` の形で変更集合を返す
- `RollbackDelta(g, d)` で元に戻せるようにする
- undo クロージャ方式は採用しない（Delta を値として扱う）

## Rationale

Delta を値として扱うことで以下の利点が得られる。

- **テストしやすい**: Delta の内容が検証可能
- **Tx に自然**: `[]Delta` を積んで逆順にロールバック可能
- **監査/永続化に強い**: 差分ログとして保持できる
- **並行制御に寄せやすい**: 変更セットが明示される
- **ImpactDiff などに拡張しやすい**

クロージャ返却型の undo は、
- 変更内容が不透明
- テスト困難
- ログ/監査と相性が悪い
という理由で採用しない。

## Design Sketch

最小インターフェース案:

```go
// Delta はイベント適用による変更集合を表す
// すべて「適用前の情報」を含み、Rollback に必要十分であること
// (例) NodeRemoved はノード全体のスナップショットが必要

type Delta struct {
  // node changes
  AddedNodes    []NodeID
  RemovedNodes  []NodeSnapshot
  UpdatedAttrs  []AttrChange

  // edge changes
  AddedEdges    []Edge
  RemovedEdges  []Edge

  // metadata
  BeforeRev int
  Event     Event
}

func ApplyEvent(g *Graph, e Event) (Delta, error)
func RollbackDelta(g *Graph, d Delta) error
```

### NodeRemoved の扱い

`RemovedNodes` には **ノードの完全スナップショット**（attrs + incoming/outgoing）を保持する。これにより、Rollback で完全復元できる。

## Consequences

### Positive

- SimulateEvent を **O(Δ)** に近づけられる
- Enterprise 規模でもボトルネックになりにくい
- Tx や監査への拡張が容易

### Negative

- Delta の設計が必要（NodeRemoved のスナップショット等）
- 実装がやや複雑になる

## References

- ADR-0002: Revision と Simulation の扱い
- ADR-0003: Impact/Evidence のグラフ選択
- ADR-0004: Validation の責務分離
