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

## 最新結果

### Core (N10k/M30k, N50k/M150k)

```
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
```

---

## 目的と読み方（短く）

- **なぜ測るか**: O(K) が効いているか／最悪ケースで破綻しないかを確認するため。
- **Impact**: 影響範囲に比例して増えるのが正常（O(K)）。
- **Worst-case (Hub/Chain)**: K≈N の上限。ここが許容範囲なら安心。
- **SimulateTx**: Tx長にほぼ線形ならOK。
- **ValidateEvent**: 定数時間で動けば仕様通り。
- **Replay / ValidateFull**: O(N+M) なのでスケール感の確認用。
