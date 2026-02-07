# ADR-0001: Event Log を Source of Truth とする

## Status

Accepted

## Context

v0.2 では各リビジョンのグラフ $G_r$ を直接永続化していた。

問題:
1. **ストレージ爆発**: リビジョン数に比例してグラフが蓄積
2. **GC 困難**: どのリビジョンを破棄してよいか不明
3. **差分計算コスト**: $\Delta V$ の計算に両グラフのロードが必要

## Decision

**Event Log を唯一の Source of Truth とする。**

$$
\text{SoT} = \mathcal{L} = [e_0, e_1, \ldots, e_n]
$$

グラフはログからの射影:

$$
G_r = \text{Replay}([e_0, \ldots, e_r])
$$

この射影はキャッシュであり、破棄・再構築可能。

### 帰結

1. **Revision = ログオフセット**: 状態はログ再生で再構築
2. **Seeds はイベントから直接抽出**: 差分計算の前処理不要
3. **Projection は破棄可能**: LRU/TTL で管理
4. **Snapshot でチェックポイント**: $G_r = \text{Replay}(\text{snap}(r_0), [e_{r_0+1}, \ldots, e_r])$

## Consequences

### Positive

- ストレージ効率: 差分（イベント）のみ保持
- GC 明確: Projection はキャッシュとして TTL/LRU 管理
- 監査強力: イベント列から因果追跡可能
- AI フレンドリー: 仮説イベントを sandbox に差し込み可能

### Negative

- 復元コスト: 古いリビジョン参照時に replay 必要
  - Mitigation: Snapshot
- スキーマ進化: イベント形式変更は慎重に
  - Mitigation: バージョニング、マイグレーション

## References

- [Event Sourcing (Fowler)](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Kafka Log Compaction](https://docs.confluent.io/kafka/design/log_compaction.html)
- [charter.md](../theory/charter.md)
