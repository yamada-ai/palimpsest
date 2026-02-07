# ADR-0006: Immutable Snapshot + Tail Replay による独立Graph

## Status

Accepted

## Context

Palimpsest は「安全に変更できること」を価値の中心に置く。共有ミュータブルな Graph キャッシュは、
中間状態の観測や排他制御の複雑化につながり、バグの温床になりやすい。

一方で、リクエストごとに Event Log 全体を Replay する方式は、
Enterprise 規模では性能上のボトルネックになる。

## Decision

**共有ミュータブルな Graph を持たず、immutable snapshot を共有し、
各リクエストは snapshot + tail replay により独立Graphを構築する。**

- Snapshot は読み取り専用・スレッドセーフで共有可能
- リクエストは snapshot を基点に tail events を適用して Graph を作成
- `SimulateEvent` はその Graph を専有する前提で動作する（排他不要）

## Consequences

### Positive

- 共有ミュータブル状態を排除でき、並行実行の安全性が高い
- 各リクエストが独立Graphを持つため、SimulateEvent の apply→rollback が安全
- フルReplayを回避でき、性能が現実的になる

### Negative

- Snapshot 生成/管理の仕組みが必要
- Snapshot 間隔やキャッシュ戦略を設計する必要がある

## Notes

- Snapshot 間隔、LRU/TTL 等のキャッシュ戦略は後続仕様で定める
- Snapshot は Graph の完全コピーではなく、読み取り専用の基点として扱う

## References

- ADR-0001: Event Log を Source of Truth とする
- ADR-0005: Delta ベースの適用とロールバック
