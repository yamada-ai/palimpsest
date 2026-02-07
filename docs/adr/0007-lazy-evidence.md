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

## Benchmark (Core)

以下は `go test -bench . -run '^$' -benchtime 10s ./...` の結果（Apple M1）。
Impact は seed をローテーションして CPU キャッシュ偏りを抑えている。

```
BenchmarkReplay/N10k_M30k-8                 7.57ms/op     13.8MB/op   80,083 allocs/op
BenchmarkReplay/N50k_M150k-8                42.49ms/op    68.3MB/op   400,277 allocs/op
BenchmarkImpact/N10k_M30k-8                 32.2µs/op     32KB/op     205 allocs/op
BenchmarkImpact/N50k_M150k-8                275µs/op      244KB/op    894 allocs/op
BenchmarkSimulateEvent/N10k_M30k-8          4.29ms/op     4.56MB/op   40,496 allocs/op
BenchmarkSimulateEvent/N50k_M150k-8         37.93ms/op    22.99MB/op  202,144 allocs/op
BenchmarkValidateEvent/N10k_M30k-8          51.5ns/op     48B/op      1 allocs/op
BenchmarkValidateEvent/N50k_M150k-8         53.3ns/op     48B/op      1 allocs/op
BenchmarkValidateFull/N10k_M30k-8           4.66ms/op     4.48MB/op   40,002 allocs/op
BenchmarkValidateFull/N50k_M150k-8          44.71ms/op    22.40MB/op  200,002 allocs/op
```

### Interpretation

- Impact/ValidateEvent はスケールしても非常に軽い（O(K) / 定数に近い）。
- Replay/Simulate/ValidateFull は O(N+M) 相当のコスト。
- 「全体検証は監査用途」「Impactは頻繁に使う」という設計が妥当であることを示す。

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
