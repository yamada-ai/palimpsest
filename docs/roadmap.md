# Roadmap

## 現在のフェーズ

**PoC 完了（Go 実装） + Phase 2 の一部先行**

| ファイル | 内容 | 状態 |
|---------|------|------|
| `event.go` | 6イベント + TxMarker + Seeds抽出 | Done |
| `graph.go` | 隣接リスト表現のグラフ | Done |
| `replay.go` | ログ → グラフ射影 | Done |
| `impact.go` | BFS + 証拠パス | Done |
| `validation.go` | Dangling検出 | Done |
| `delta.go` | Delta適用 + Rollback | Done |
| `simulate.go` | Pre/Apply/Postフロー | Done |
| `snapshot.go` | Snapshot + tail replay | Done |
| `core_bench_test.go` | Coreベンチ（Replay/Impact/Validate） | Done |

---

## マイルストーン

```
Phase 0: PoC (現在地)
    ↓
Phase 1: RFC + Protobuf
    - RFC-0001: Event Schema（Protobuf 正式定義）
    - RFC-0002: Core Logic（replay/impact/validate 仕様化）
    ↓
Phase 2: Sandbox + Speculative
    - 仮説イベント適用
    - LRU Cache
    - Speculative Computation
    ↓
Phase 3: AI Simulation
    - 変更前シミュレーション
    - Repair Plan 生成
    ↓
Phase 4: Production
    - 永続化（ファイル / DB）
    - Snapshot / Checkpoint
    - API サーバー
    ↓
Phase 5: UI
    - 影響一覧、証拠パス、検証理由の可視化
    - デバッガとして設計
```

---

## 実装の優先順位

**UI は最後**。まず計算エンジンを固める。

1. **スキーマ定義** (`packages/schema/`)
   - Protobuf でイベント・グラフの型を定義
   - Go / TS コード生成

2. **Core ロジック** (`packages/core/`)
   - DB 依存なしの純粋関数
   - replay, impact, validate

3. **API** (`apps/api/`)
   - 薄いファサード
   - Core の結果を HTTP/gRPC で公開

4. **UI** (`apps/web/`)
   - 内部状態を可視化する「デバッガ」
   - ユーザーの不安を解消する設計

---

## Out of Scope（当面やらない）

- マルチテナント対応
- 動的依存の精密化（self-adjusting computation）
- 不動点計算（循環依存の完全対応）
