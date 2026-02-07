# 形式モデル

## 数学的定義

本文書は [charter.md](./charter.md) で論じた設計判断を、数学的に厳密な形で記述する。各定義の動機と根拠は charter を参照されたい。

---

## 1. グラフ構造

### 1.1 なぜグラフか

設定要素間の依存関係を表現するモデルとして、有向グラフを選択する。理由：

- 依存は本質的に二項関係であり、辺として表現できる
- 到達可能性が「影響の伝播」に直接対応する
- グラフ理論のアルゴリズム（BFS、SCC 検出等）が利用可能

関係データベースや木構造では、多対多の依存や依存の種別を表現しにくい。

### 1.2 定義

リビジョン $r$ における設定状態を型付きラベル付き有向グラフとして定義：

$$
G_r = (V_r, E_r), \quad E_r \subseteq V_r \times V_r \times L
$$

- $V_r$: ノード集合。各ノードは設定要素（エンティティ、フィールド、フォーム等）
- $E_r$: ラベル付き有向辺集合。各辺は依存関係
- $L$: ラベル集合。依存の種別

### 1.3 辺の向き

Provider → Consumer 方向を採用：

$$
(u \xrightarrow{l} v) \in E_r \iff \text{$v$ は $u$ に依存する}
$$

$u$ が変更されると $v$ が影響を受ける。

**なぜこの向きか**: 影響は「変更元から変更先へ」伝播する。provider → consumer の向きでは、変更ノードから順方向に辿るだけで影響範囲を計算できる（$O(K)$）。逆向きでは全ノードから逆引きが必要になり $O(N)$ となる。

### 1.4 ラベル集合

$$
L = \{\texttt{uses}, \texttt{derives}, \texttt{controls}, \texttt{constrains}\}
$$

| $l$ | 意味 | 影響の性質 | 例 |
|-----|------|-----------|-----|
| `uses` | データ依存 | 値が変わると再計算 | `Field → Expression` |
| `derives` | 構造的所有 | 親が消えると子も無効 | `Entity → Field` |
| `controls` | 振る舞い制御 | 権限変更は対象に影響 | `Role → Form` |
| `constrains` | 制約条件 | 制約変更は検証対象に影響 | `Rule → Field` |

この 4 分類は理論的最小ではなく、運用上の安全性と説明可能性を優先した設計である。

### 1.5 ノード型

$$
T = \{\texttt{Entity}, \texttt{Field}, \texttt{Form}, \texttt{List}, \texttt{Expression}, \texttt{Role}\}
$$

各ノード $v \in V$ は型 $\tau(v) \in T$ と属性 $\text{attrs}(v)$ を持つ。

---

## 2. Event Log

### 2.1 なぜ Event Log か

状態を直接永続化する代わりに、状態を変化させたイベントの列を永続化する（Event Sourcing）。利点：

- ストレージ効率（差分のみ保存）
- GC の明確化（状態はキャッシュとして破棄可能）
- Seeds の直接抽出（差分計算不要）
- 監査（因果関係の追跡）

### 2.2 定義

Source of Truth としてのイベント列：

$$
\mathcal{L} = [e_0, e_1, \ldots, e_n]
$$

### 2.3 イベント集合

$$
\mathcal{E} = \{\texttt{NodeAdded}, \texttt{NodeRemoved}, \texttt{EdgeAdded}, \texttt{EdgeRemoved}, \texttt{AttrUpdated}, \texttt{TxMarker}\}
$$

各イベントは最小単位の操作を表す。TransactionMarker は複数イベントを論理的にグループ化する。

### 2.4 Revision

$$
\text{rev} : \mathbb{N} \to \text{offset in } \mathcal{L}
$$

リビジョンはログ内のオフセット（位置）として定義。これにより管理が単純化される。

### 2.5 Projection

ログからグラフへの射影：

$$
\text{Replay} : \mathcal{L}_{0..r} \to G_r
$$

$$
G_r = \text{Replay}([e_0, \ldots, e_r])
$$

この射影はキャッシュであり、必要に応じて破棄・再構築可能。

---

## 3. Impact Analysis

### 3.1 なぜ到達可能性か

依存は推移的である。A が B に依存し、B が C に依存するなら、A の変更は C にも影響する。この推移的閉包が到達可能性である。

### 3.2 到達可能性

Seeds $S \subseteq V$ から到達可能なノード集合：

$$
\text{Reach}_G(S) = \{v \in V \mid \exists s \in S,\ s \leadsto_G v\}
$$

ここで $s \leadsto_G v$ は $G$ 上の有向パス $s = v_0 \to v_1 \to \cdots \to v_k = v$ の存在。

### 3.3 Impact

$$
\text{Impact}(S) = \text{Reach}_G(S)
$$

### 3.4 Evidence Path

各 $x \in \text{Impact}(S)$ に対し、最短証拠パス：

$$
\pi(s, x) = \arg\min_{p: s \leadsto x} |p|
$$

BFS の性質により、最初に到達したパスが最短。これにより「なぜ影響を受けるか」を簡潔に説明できる。

---

## 4. Seeds 抽出

### 4.1 なぜイベントから直接抽出するか

従来の差分計算：
$$
\Delta V = (V_{\text{current}} \setminus V_{\text{prev}}) \cup (V_{\text{prev}} \setminus V_{\text{current}}) \cup \ldots
$$
これには両グラフのロードが必要で $O(N)$。

Event Log では、イベント自体が差分情報を持つため、直接 Seeds を抽出できる。

### 4.2 Seeds 抽出関数

$$
\text{Seeds} : \mathcal{E} \to \mathcal{P}(V)
$$

### 4.3 Impact Seeds

$$
\text{Seeds}_I(e) = \begin{cases}
\{n\} & e \in \{\texttt{NodeAdded}(n), \texttt{NodeRemoved}(n), \texttt{AttrUpdated}(n)\} \\[6pt]
\{v\} & e = \texttt{EdgeAdded}(u, v, l),\ l \in \{\texttt{uses}, \texttt{derives}\} \\[6pt]
\{u, v\} & e = \texttt{EdgeAdded}(u, v, l),\ l \in \{\texttt{controls}, \texttt{constrains}\} \\[6pt]
\emptyset & e = \texttt{TxMarker}
\end{cases}
$$

**ラベルによる分岐の理由**: `uses`/`derives` はデータフローであり、consumer のみが影響を受ける。`controls`/`constrains` は制御フローであり、「何を制御しているか」と「誰に制御されているか」の両方が意味を持つため両端を含める。

### 4.4 Validation Seeds

$$
\text{Seeds}_V(e) = \begin{cases}
\{n\} & e \in \{\texttt{NodeAdded}(n), \texttt{NodeRemoved}(n), \texttt{AttrUpdated}(n)\} \\[6pt]
\{u, v\} & e \in \{\texttt{EdgeAdded}(u, v, l), \texttt{EdgeRemoved}(u, v, l)\} \\[6pt]
\emptyset & e = \texttt{TxMarker}
\end{cases}
$$

Validation は整合性チェックであり、常に両端を検査する。

### 4.5 Impact と Validation の分離

| 種別 | 目的 | 性質 |
|------|------|------|
| Impact | 影響範囲の情報提供 | 影響があること自体は禁止理由にならない |
| Validation | コミット可否の判定 | 不変条件違反があれば禁止 |

この分離により、過度に保守的なポリシー（「影響があるから禁止」）を避けられる。

---

## 5. 計算量

### 5.1 記号

| 記号 | 定義 |
|------|------|
| $N$ | $\|V\|$（全ノード数） |
| $M$ | $\|E\|$（全辺数） |
| $K_V$ | $\|\text{Reach}(S)\|$（影響ノード数） |
| $K_E$ | 影響部分グラフの辺数 |
| $K$ | $K_V + K_E$ |

### 5.2 BFS の計算量

隣接リスト表現において：

$$
T(\text{Impact}) = O(K_V + K_E) = O(K)
$$

$$
S(\text{Impact}) = O(K_V)
$$

**重要**: 計算量は全体サイズ $N$ ではなく影響範囲 $K$ に比例。多くの場合 $K \ll N$ であり、これが増分計算の利点。

### 5.3 $O(K)$ の実現条件

理論上の $O(K)$ を実装で達成するための条件：

1. **Lazy Loading**: 訪問ノードの隣接リストのみロード
2. **連番 ID**: ディスク I/O の局所性確保
3. **LRU Cache**: ホットノードのメモリ保持
4. **Small Seeds**: $|S|$ の最小化

逆に、全ノードの事前ロードや隣接リスト取得に $O(N)$ かかる場合は成り立たない。

---

## 6. 不変条件

### 6.1 参照整合性

任意のリビジョンで参照は閉じている：

$$
\forall (u \to v) \in E_r : u \in V_r \land v \in V_r
$$

存在しないノードへの参照（dangling edge）は許可しない。

### 6.2 SCC サイズ制限

強連結成分のサイズに上限を設ける：

$$
\forall C \in \text{SCC}(G_r) : |C| \leq \theta
$$

**理由**: 循環依存があると、SCC 全体が相互に影響し合い、増分計算の効果が薄れる。閾値 $\theta$ 超過時は警告または拒否し、設計の見直しを促す。

---

## 7. Snapshot と復元

### 7.1 Snapshot

リビジョン $r_0$ でのグラフ状態を保存：

$$
\text{snap}(r_0) = G_{r_0}
$$

### 7.2 復元

$$
G_r = \text{Replay}(\text{snap}(r_0), [e_{r_0+1}, \ldots, e_r])
$$

スナップショットからの差分 replay により、古いリビジョンへのアクセスを高速化。

---

## 8. 理論的背景

本モデルは以下の確立された理論・システムに基づく：

| 概念 | 出典 | 適用箇所 |
|------|------|---------|
| 増分計算の形式化 | Build Systems à la Carte [Mokhov+ 2020] | グラフモデル、依存追跡 |
| Demand-driven computation | Adapton [Hammer+ 2014] | 必要な部分のみ計算 |
| Red-green algorithm | rustc incremental | 変更検出と再計算 |
| Event Sourcing | Fowler | ログを SoT とするパターン |
| Abstract Interpretation | Cousot & Cousot [1977] | over-approximation の正当化 |

これらは数十年の研究と実運用に裏打ちされており、理論的健全性と実用的スケーラビリティが検証済みである。
