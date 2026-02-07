# ADR-0004: Validation の責務分離

## Status

Accepted

## Context

Validation には2つの異なる用途がある：

1. **イベント適用前のゲート**: 不正なイベントを拒否する
2. **既存状態の健全性確認**: 監査・バックグラウンドチェック

これらを混同すると、「影響がある＝禁止」という過度に保守的なポリシーになり、運用が破綻する。

## Decision

**Validation を2つの関数に分離し、責務を明確にする。**

### ValidateEvent（適用前ゲート）

```go
func ValidateEvent(ctx context.Context, g *Graph, e Event) *ValidationResult
```

**責務**:
- イベント単体の妥当性チェック
- 局所不変条件の確認

**チェック内容**（実装に応じて拡張可能）:

| イベント | チェック |
|---------|---------|
| `NodeAdded(n)` | 同一 ID のノードが既に存在しないか |
| `NodeRemoved(n)` | 参照されていないか（dangling を作らないか） |
| `EdgeAdded(u, v, l)` | 両端ノードが存在するか |
| `EdgeRemoved(u, v, l)` | 対象エッジが存在するか |
| `AttrUpdated(n)` | 対象ノードが存在するか |

上記は最低限のチェック。将来的に必須属性や SCC 閾値などを追加可能。

**タイミング**: イベント適用前。拒否された場合は適用しない。

### Validate（全体チェック）

```go
func Validate(ctx context.Context, g *Graph) *ValidationResult
```

**責務**:
- グラフ全体の健全性確認
- 監査用途
- バックグラウンド / 定期実行

**用途**:
- 運用中の定期監査
- デバッグ時の状態確認
- ログ破損の検知

## Consequences

### Positive

- 「影響がある＝禁止」を防げる
- UX と安全性を両立できる
- Event Sourcing と相性が良い（イベント単位でゲート）
- 責務が明確で保守しやすい

### Negative

- 2つの関数を適切に使い分ける必要がある
- ValidateEvent の呼び出しを忘れると不正なイベントが適用される

## Notes

- Impact は「何が影響を受けるか」の**情報提供**
- Validation は「コミットしてよいか」の**判定**
- この2つを混同しないことが設計の要点

## References

- ADR-0001: Event Log を Source of Truth とする
- docs/theory/formal-model.md（6.2 Validation のタイミング）
