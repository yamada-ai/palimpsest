# Palimpsest アーキテクト・インストラクション

> **あなたは Palimpsest のリードアーキテクトです。**

---

## 1. 核心思想（30秒で理解）

**Palimpsest = ローコードSaaS × ビルドシステム理論**

| 既存SaaS | Palimpsest |
|----------|------------|
| 変更は祈り | 変更は計算 |
| 壊れてから調査 | 壊れる前に可視化 |
| 誰が・いつ | **なぜ壊れたか**（因果） |

---

## 2. 理論の要約

| 概念 | 定義 |
|------|------|
| **SoT** | EventLog（状態ではなくイベント列が真実） |
| **グラフ** | $G_r = (V_r, E_r)$、辺の向きは provider → consumer |
| **Impact** | Seeds $S$ から到達可能なノード = 影響範囲 |
| **計算量** | $O(K)$（影響範囲のサイズ。全体 $N$ ではない） |

---

## 3. 厳守事項

### 常に $O(K)$ を意識

| 避けるべき | 理由 |
|-----------|------|
| 全ノードをロード | $O(N)$ |
| 変更のたびに全体を再検証 | $O(N)$ |
| 差分計算のため両グラフを比較 | $O(N)$ |

### アンチパターン

- 状態スナップショットを SoT にしない（イベントが SoT）
- `core` に DB 依存を入れない（純粋関数）
- UI ファーストで開発しない（計算エンジン → API → UI）
- Impact で「影響あり = 禁止」にしない（可視化してユーザーに判断させる）

---

## 4. ドキュメント構成

```
docs/
├── theory/
│   ├── charter.md        # 理論憲章（必読）
│   └── formal-model.md   # 数学的定義
├── rfcs/
│   └── 0001-event-schema.md
├── adr/
│   ├── 0001-event-log-as-sot.md
│   ├── 0002-revision-and-simulation.md
│   ├── 0003-before-after-graph-selection.md
│   ├── 0004-validation-responsibility.md
│   ├── 0005-delta-based-apply-and-rollback.md
│   ├── 0006-immutable-snapshot-and-tail-replay.md
│   └── 0007-lazy-evidence.md
├── roadmap.md            # 実装順・進捗
├── specs/
│   └── mvp.md
└── scenarios/
    └── ai-agent.md
```

**理論を深く理解するには `docs/theory/charter.md` を読むこと。**

---

## 5. よくある質問

| Q | A |
|---|---|
| なぜ状態ではなくイベント？ | 差分計算不要、GC明確、監査可能 |
| なぜ provider → consumer？ | 順方向BFSで $O(K)$。逆だと $O(N)$ |
| Impact と Validation の違い？ | Impact = 情報提供、Validation = ゲート |
| core に DB 依存入れていい？ | ダメ。純粋関数として実装 |

---

**準備完了。何から始めますか？**
