# ADR-0003: Impact/Evidence のグラフ選択（before/after）

## Status

Accepted

## Context

Impact 計算と Evidence 生成において、イベント適用前（before）と適用後（after）のどちらのグラフを使うべきかが問題になる。

- 削除系イベント（NodeRemoved, EdgeRemoved）では、適用後にはノードやエッジが存在しないため、因果パスを説明できない
- 追加系イベント（NodeAdded, EdgeAdded, AttrUpdated）では、適用前には新しい依存が存在しないため、影響範囲が空になる

## Decision

**イベント種別により、Impact/Evidence 計算に使うグラフを分ける。**

| イベント | 使用グラフ | 理由 |
|---------|-----------|------|
| `NodeRemoved` | **before** | 削除後はノードが存在せず、因果パスが定義不能 |
| `EdgeRemoved` | **before** | 削除後はエッジが存在せず、依存関係を説明不能 |
| `NodeAdded` | **after** | 追加前はノードが存在せず、影響がゼロになる |
| `EdgeAdded` | **after** | 追加前はエッジが存在せず、新たな依存を説明不能 |
| `AttrUpdated` | **after** | 変更後の状態で影響を見るのが自然 |

### 実装上の責務分離

`SimulateEvent` は **Pre（before）と Post（after）の両方の Impact/Evidence を返す**。
どちらを表示・利用するかは **呼び出し側（UI/API）が選択** する。

これにより：
- コアロジックは両方を計算するだけで、選択ロジックを持たない
- UI/API がイベント種別に応じて適切な結果を表示できる
- 将来的に両方を比較表示する ImpactDiff にも対応しやすい

### EdgeRemoved の Evidence

- before graph のパスを返す
- 「この依存関係が**存在していた**」という説明になる
- ビルドシステムの依存削除と同型の意味論

### ImpactDiff について

before/after の差分を提示する ImpactDiff は **MVP では不要**。

理由：
- 実装・説明コストが高い
- AI/UX が本当に必要になった段階で追加すればよい

## Consequences

### Positive

- 削除イベントの説明力を維持できる（「何を壊したか」が分かる）
- 追加イベントの影響範囲が正しく計算できる
- イベントの意味論に沿った自然な設計

### Negative

- 実装が複雑になる（イベント種別で分岐が必要）
- Simulation で before/after 両方のグラフを保持する必要がある

## Notes

- 「全部 before に統一」は削除イベントの説明力を失うため NG
- 「全部 after に統一」は追加イベントで影響がゼロになり UX が破綻

## References

- ADR-0002: Revision と Simulation の扱い
- docs/theory/formal-model.md
