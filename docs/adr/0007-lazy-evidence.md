# ADR-0007: Evidence の遅延評価（オンデマンド生成）

## Status

Accepted

## Context

Impact 計算時に「全影響ノードの証拠パス」を事前構築すると、
大規模グラフでメモリと時間が爆発する。特にエンタープライズ規模では、
Evidence を常時保持する設計は現実的でない。

### ベンチマーク（Before）

```
Bench config: nodes=50000 edges=150000 seed=42
Replay: 31.758625ms
Impact: 22.887198709s (impacted=25001)
Memory delta: 7714.64 MB
```

上記は「全件Evidence構築」を行っていた時の測定結果。
Impact 自体はO(K)だが、Evidence の全件構築で O(K * path_length) が発生し、
時間・メモリが許容範囲を超える。

## Decision

**Evidence は遅延評価（オンデマンド生成）とする。**

- Impact 計算では EvidencePath を生成せず、
  **parent/seedOf** のみ保持する
- 必要なタイミングで `EvidencePath(node)` / `Path(node)` を計算する

## Result (After)

```
Bench config: nodes=50000 edges=150000 seed=42
Replay: 36.082ms
Impact: 8.590167ms (impacted=25001)
Memory delta: 9.86 MB
```

Evidence 遅延評価により、時間・メモリが大幅に改善された。

## Consequences

### Positive

- 大規模グラフでも Impact が実用的な時間とメモリに収まる
- Evidence の計算は「必要なときだけ」実行できる
- Enterprise での実運用に耐える

### Negative

- Evidence を要求する都度、パス復元が必要
- すべての影響ノードの Evidence を一括で取得すると、従来同様にコストが増大

## Notes

- Evidence のUI表示は「クリック時に1件生成」が推奨
- parent/seedOf の保持は O(K) に抑えられる

## References

- ADR-0003: Impact/Evidence のグラフ選択（before/after）
- ADR-0005: Delta ベースの適用とロールバック
