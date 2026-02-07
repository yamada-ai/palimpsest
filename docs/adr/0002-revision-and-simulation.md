# ADR-0002: Revision と Simulation の扱い

## Status

Accepted

## Context

Palimpsest では Event Log を Source of Truth として扱い、Revision はログ内オフセットで定義される。PoC/MVP では、イベントの適用前後をシミュレーションし、Impact/Validation の結果を返す必要がある。

Simulation では "after" 状態を提示するために便宜的なリビジョン表現が欲しくなるが、Revision を実在しない値として扱うと監査性・再現性・整合性が損なわれる懸念がある。

## Decision

- **Revision は常に Event Log の実在オフセットのみを指す。**
- Simulation の "after" 表現は **仮想的な表示**として扱い、**UI/説明用途に限る**。
- コアロジック内では `AfterRevision` を持ってよいが、**仮想値であることを明記**し、ログ上の正式な Revision と混同しない。

## Consequences

### Positive

- 監査性と再現性が保たれる（Revision の定義が一意）。
- Event Log を SoT とする設計と整合する。
- Simulation の「未来表示」を提供しつつ、実在状態との混同を防げる。

### Negative

- UI/説明側で「仮想値」であることを明示する必要がある。
- Revision の単純なインクリメントをそのまま表示できないケースがある。

## Notes

- `AfterRevision` はあくまで **仮想値**（Before+1）であり、
  実際のログ位置ではない。
- 将来、Tx 単位のコミットや Sandbox を導入しても、
  **Revision = Log offset** の定義は維持する。

## Related ADRs

- ADR-0001: Event Log を Source of Truth とする
- ADR-0003: Impact/Evidence のグラフ選択（before/after）
- ADR-0004: Validation の責務分離

## References

- docs/specs/mvp.md
- docs/theory/formal-model.md
