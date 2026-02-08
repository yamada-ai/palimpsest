# ベンチマーク

## 実行方法

```sh
GOCACHE=/tmp/go-build-cache go test -bench . -run '^$' -benchtime 5s ./...
```

### 注意

- `GOCACHE` is set to avoid permission issues in the default cache.
- ベンチはCPU/OSに依存します。絶対値より傾向を重視してください。

---

## 実行環境

- OS: macOS (darwin)
- CPU: Apple M1
- Go: `go1.25.1`

---

## 最新結果（一覧）

```
# Core
BenchmarkReplay/N10k_M30k-8            769     6.65ms/op   13.83MB/op   80,083 allocs/op
BenchmarkReplay/N50k_M150k-8           146    41.58ms/op   68.30MB/op  400,277 allocs/op
BenchmarkImpact/N10k_M30k-8        156,918   44.4µs/op     40.5KB/op      205 allocs/op
BenchmarkImpact/N50k_M150k-8        17,799  341.3µs/op    284.3KB/op      894 allocs/op
BenchmarkImpactWorstCase/N10k/Hub-8    1,596   3.22ms/op   5.51MB/op      343 allocs/op
BenchmarkImpactWorstCase/N10k/Chain-8  1,408   3.96ms/op   5.00MB/op   20,322 allocs/op
BenchmarkImpactWorstCase/N50k/Hub-8      331  19.79ms/op  24.22MB/op    1,126 allocs/op
BenchmarkImpactWorstCase/N50k/Chain-8    225  23.82ms/op  20.68MB/op  101,098 allocs/op
BenchmarkSimulateEvent/N10k_M30k-8        925   6.48ms/op   4.58MB/op   40,496 allocs/op
BenchmarkSimulateEvent/N50k_M150k-8        121  45.38ms/op  23.09MB/op  202,144 allocs/op
BenchmarkSimulateTx/N10k/Tx1-8           1,094   5.28ms/op   4.58MB/op   40,498 allocs/op
BenchmarkSimulateTx/N10k/Tx10-8          1,033   6.20ms/op   5.25MB/op   41,703 allocs/op
BenchmarkSimulateTx/N10k/Tx100-8           567  10.46ms/op  10.88MB/op   52,269 allocs/op
BenchmarkSimulateTx/N50k/Tx1-8             147  51.22ms/op  23.09MB/op  202,146 allocs/op
BenchmarkSimulateTx/N50k/Tx10-8            100  59.65ms/op  25.68MB/op  207,761 allocs/op
BenchmarkSimulateTx/N50k/Tx100-8            58 118.51ms/op  51.09MB/op  257,817 allocs/op
BenchmarkValidateEvent/N10k_M30k-8   100,000,000   51.1ns/op   48B/op   1 allocs/op
BenchmarkValidateEvent/N50k_M150k-8  119,348,324   50.4ns/op   48B/op   1 allocs/op
BenchmarkValidateFull/N10k_M30k-8        1,222   6.54ms/op   4.48MB/op   40,002 allocs/op
BenchmarkValidateFull/N50k_M150k-8          100  53.81ms/op  22.40MB/op  200,002 allocs/op

# Attrs size (ReplayWithAttrs)
BenchmarkReplayWithAttrs/N10k_M30k/Attrs1-8     1,002   6.25ms/op  13.35MB/op   70,083 allocs/op
BenchmarkReplayWithAttrs/N10k_M30k/Attrs5-8       944   6.42ms/op  13.35MB/op   70,083 allocs/op
BenchmarkReplayWithAttrs/N10k_M30k/Attrs20-8      915   6.78ms/op  13.35MB/op   70,083 allocs/op
BenchmarkReplayWithAttrs/N50k_M150k/Attrs1-8      142  41.77ms/op  65.90MB/op  350,277 allocs/op
BenchmarkReplayWithAttrs/N50k_M150k/Attrs5-8      124  46.77ms/op  65.90MB/op  350,277 allocs/op
BenchmarkReplayWithAttrs/N50k_M150k/Attrs20-8     116  50.10ms/op  65.90MB/op  350,277 allocs/op

# Snapshot + Tail Replay
BenchmarkBuildGraphFromSnapshot/SnapEvery1000-8   272  22.16ms/op  23.35MB/op  200,131 allocs/op
BenchmarkBuildGraphFromSnapshot/SnapEvery5000-8   273  22.19ms/op  23.35MB/op  200,131 allocs/op
BenchmarkBuildGraphFromSnapshot/SnapEvery10000-8  270  22.41ms/op  23.35MB/op  200,131 allocs/op

# Edge density (M/N)
BenchmarkImpactEdgeDensity/N10k_M30k/MperN1-8   189,831  39.6µs/op   38.8KB/op     243 allocs/op
BenchmarkImpactEdgeDensity/N10k_M30k/MperN3-8   159,627  38.8µs/op   48.4KB/op     243 allocs/op
BenchmarkImpactEdgeDensity/N10k_M30k/MperN10-8   96,859  54.7µs/op   82.0KB/op     243 allocs/op
BenchmarkImpactEdgeDensity/N50k_M150k/MperN1-8   24,033 255.4µs/op  297KB/op      1,067 allocs/op
BenchmarkImpactEdgeDensity/N50k_M150k/MperN3-8   23,002 282.1µs/op  345KB/op      1,067 allocs/op
BenchmarkImpactEdgeDensity/N50k_M150k/MperN10-8  17,041 386.2µs/op  513KB/op      1,067 allocs/op

# Relation mix (ImpactFromEventFiltered)
BenchmarkImpactRelationMix/N10k_M30k/None-8      1,428  4.26ms/op  4.13MB/op   20,242 allocs/op
BenchmarkImpactRelationMix/N10k_M30k/Every10-8   1,603  3.76ms/op  4.24MB/op   20,263 allocs/op
BenchmarkImpactRelationMix/N10k_M30k/Every5-8    1,597  4.20ms/op  4.35MB/op   20,272 allocs/op
BenchmarkImpactRelationMix/N50k_M150k/None-8       296 19.72ms/op 17.18MB/op  100,824 allocs/op
BenchmarkImpactRelationMix/N50k_M150k/Every10-8    271 29.15ms/op 17.62MB/op  100,871 allocs/op
BenchmarkImpactRelationMix/N50k_M150k/Every5-8     242 27.43ms/op 18.06MB/op  100,904 allocs/op
```

---

## 目的と読み方（短く）

- **なぜ測るか**: O(K) が効いているか／最悪ケースで破綻しないかを確認するため。
- **Impact**: 影響範囲に比例して増えるのが正常（O(K)）。
- **Worst-case (Hub/Chain)**: K≈N の上限。ここが許容範囲なら安心。
- **SimulateTx**: Tx長にほぼ線形ならOK。
- **ValidateEvent**: 定数時間で動けば仕様通り。
- **Replay / ValidateFull**: O(N+M) なのでスケール感の確認用。
- **Edge density**: M/N が増えても緩やかに増えるなら良い（爆発しないことを確認）。
- **Relation mix**: Relation ノードの混在で大崩れしないことを確認。
