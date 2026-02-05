# 形式モデル

## 数学的定義

---

## 1. グラフ構造

### 1.1 定義

リビジョン $r$ における設定状態を型付きラベル付き有向グラフとして定義：

$$
G_r = (V_r, E_r), \quad E_r \subseteq V_r \times V_r \times L
$$

- $V_r$: ノード集合（設定要素）
- $E_r$: ラベル付き有向辺集合
- $L$: ラベル集合

### 1.2 辺の向き

Provider → Consumer 方向を採用：

$$
(u \xrightarrow{l} v) \in E_r \iff \text{$v$ は $u$ に依存する}
$$

$u$ が変更されると $v$ が影響を受ける。

### 1.3 ラベル集合

$$
L = \{\texttt{uses}, \texttt{derives}, \texttt{controls}, \texttt{constrains}\}
$$

| $l$ | 意味 | 例 |
|-----|------|-----|
| `uses` | データ依存 | `Field → Expression` |
| `derives` | 構造的所有 | `Entity → Field` |
| `controls` | 振る舞い制御 | `Role → Form` |
| `constrains` | 制約条件 | `Rule → Field` |

### 1.4 ノード型

$$
T = \{\texttt{Entity}, \texttt{Field}, \texttt{Form}, \texttt{List}, \texttt{Expression}, \texttt{Role}\}
$$

各ノード $v \in V$ は型 $\tau(v) \in T$ と属性 $\text{attrs}(v)$ を持つ。

---

## 2. Event Log

### 2.1 定義

Source of Truth としてのイベント列：

$$
\mathcal{L} = [e_0, e_1, \ldots, e_n]
$$

### 2.2 イベント集合

$$
\mathcal{E} = \{\texttt{NodeAdded}, \texttt{NodeRemoved}, \texttt{EdgeAdded}, \texttt{EdgeRemoved}, \texttt{AttrUpdated}, \texttt{TxMarker}\}
$$

### 2.3 Revision

$$
\text{rev} : \mathbb{N} \to \text{offset in } \mathcal{L}
$$

### 2.4 Projection

ログからグラフへの射影：

$$
\text{Replay} : \mathcal{L}_{0..r} \to G_r
$$

$$
G_r = \text{Replay}([e_0, \ldots, e_r])
$$

---

## 3. Impact Analysis

### 3.1 到達可能性

Seeds $S \subseteq V$ から到達可能なノード集合：

$$
\text{Reach}_G(S) = \{v \in V \mid \exists s \in S,\ s \leadsto_G v\}
$$

ここで $s \leadsto_G v$ は $G$ 上の有向パス $s = v_0 \to v_1 \to \cdots \to v_k = v$ の存在。

### 3.2 Impact

$$
\text{Impact}(S) = \text{Reach}_G(S)
$$

### 3.3 Evidence Path

各 $x \in \text{Impact}(S)$ に対し、最短証拠パス：

$$
\pi(s, x) = \arg\min_{p: s \leadsto x} |p|
$$

BFS により $O(K)$ で計算可能。

---

## 4. Seeds 抽出

イベントから Seeds を抽出する関数：

$$
\text{Seeds} : \mathcal{E} \to \mathcal{P}(V)
$$

### 4.1 Impact Seeds

$$
\text{Seeds}_I(e) = \begin{cases}
\{n\} & e \in \{\texttt{NodeAdded}(n), \texttt{NodeRemoved}(n), \texttt{AttrUpdated}(n)\} \\[6pt]
\{v\} & e = \texttt{EdgeAdded}(u, v, l),\ l \in \{\texttt{uses}, \texttt{derives}\} \\[6pt]
\{u, v\} & e = \texttt{EdgeAdded}(u, v, l),\ l \in \{\texttt{controls}, \texttt{constrains}\} \\[6pt]
\emptyset & e = \texttt{TxMarker}
\end{cases}
$$

### 4.2 Validation Seeds

$$
\text{Seeds}_V(e) = \begin{cases}
\{n\} & e \in \{\texttt{NodeAdded}(n), \texttt{NodeRemoved}(n), \texttt{AttrUpdated}(n)\} \\[6pt]
\{u, v\} & e \in \{\texttt{EdgeAdded}(u, v, l), \texttt{EdgeRemoved}(u, v, l)\} \\[6pt]
\emptyset & e = \texttt{TxMarker}
\end{cases}
$$

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

### 5.3 $O(K)$ の実現条件

1. Lazy Loading: 訪問ノードの隣接リストのみロード
2. 連番 ID: ディスク I/O の局所性
3. LRU Cache: ホットノードのメモリ保持
4. Small Seeds: $|S|$ の最小化

---

## 6. 不変条件

### 6.1 参照整合性

任意のリビジョンで参照は閉じている：

$$
\forall (u \to v) \in E_r : u \in V_r \land v \in V_r
$$

### 6.2 SCC サイズ制限

強連結成分のサイズに上限：

$$
\forall C \in \text{SCC}(G_r) : |C| \leq \theta
$$

閾値 $\theta$ 超過時は警告または拒否。

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

ここで $\text{Replay}$ はスナップショットに差分イベントを適用。

---

## 8. 理論的背景

| 概念 | 出典 |
|------|------|
| 増分計算の形式化 | Build Systems à la Carte [Mokhov+ 2020] |
| Demand-driven computation | Adapton [Hammer+ 2014] |
| Red-green algorithm | rustc incremental |
| Event Sourcing | Fowler |
