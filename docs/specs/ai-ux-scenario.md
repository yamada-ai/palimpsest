# AI エージェント UX シナリオ

## 「保存 = 賭け」から「保存 = 確定」へ

---

## シナリオ概要

**状況**: 売上計算ロジックに「地域別ランク」を追加したい

**登場人物**:
- ユーザー: 情シス担当（非エンジニア、責任は重い）
- AI: Palimpsest 内蔵エージェント
- システム: Event Log + Graph + Impact/Validation

---

## 1. ユーザーの依頼

> 「売上計算のロジックに『地域別ランク』を加味したい」

AI はこの自然言語を**イベント列**に変換する。

---

## 2. AI の処理

### 2.1 仮説イベント生成

Sandbox 上に仮想トランザクションを作成：

$$
\begin{aligned}
e_1 &= \texttt{NodeAdded}(\text{rank\_field}, \texttt{Field}, \{\ldots\}) \\
e_2 &= \texttt{EdgeAdded}(\text{rank\_field}, \text{sales\_expr}, \texttt{uses}) \\
e_3 &= \texttt{AttrUpdated}(\text{sales\_expr}, \{\text{formula}: \ldots\}) \\
e_4 &= \texttt{TxMarker}(\text{tx-001})
\end{aligned}
$$

本番ログを汚さない。TransactionMarker で一貫したプレビューを保証。

### 2.2 Projection

Sandbox 上でグラフ $G'$ を構築（必要部分のみ lazy load）。

### 2.3 Impact 計算

Seeds をイベント列から抽出：

$$
S = \{\text{rank\_field}, \text{sales\_expr}\}
$$

BFS で到達可能性を計算：

$$
\text{Impact}(S) = \{\text{sales\_expr}, \text{report\_A}, \text{report\_B}, \text{report\_C}, \ldots\}
$$

証拠パスも生成：

$$
\pi = \text{sales\_expr} \to \text{monthly\_total} \to \text{report\_A}
$$

### 2.4 Validation

- Dangling edge なし
- 必須制約違反なし
- SCC 閾値内

### 2.5 修正案生成

影響を受けるノードに対して最小修正を提案：

$$
\begin{aligned}
e'_1 &= \texttt{AttrUpdated}(\text{report\_A\_expr}, \{\text{formula}: \ldots\}) \\
e'_2 &= \texttt{AttrUpdated}(\text{report\_B\_expr}, \{\text{formula}: \ldots\}) \\
e'_3 &= \texttt{AttrUpdated}(\text{report\_C\_expr}, \{\text{formula}: \ldots\})
\end{aligned}
$$

---

## 3. AI の回答

> その変更を行うと、**月次帳票 A / B / C** が壊れます。
>
> **理由**: 売上計算式の変更が帳票の集計式に伝播するためです。
>
> ```
> sales_expr → monthly_total → report_A
> ```
>
> **修正案**: 帳票側の式を以下のように更新すれば整合性が保てます。
>
> プレビューしますか？

### 提示要素

| 優先度 | 要素 |
|--------|------|
| 1 | 影響一覧（重大度付き） |
| 2 | 影響理由（証拠パス） |
| 3 | 修正案（イベント列） |
| 4 | プレビュー |

---

## 4. 体験の反転

### Before（既存 SaaS）

```
変更 → 保存 → 祈る → 障害発覚 → 原因調査 → 修正
```

### After（Palimpsest）

```
変更 → 影響確認 → 修正案適用 → 保存（確定）
```

保存前に：
- 破壊的変更が**見えている**
- 修正案が**同時に出る**
- 因果が**記録される**

---

## 5. MVP 必須コンポーネント

| コンポーネント | 役割 |
|--------------|------|
| Event Log | 6 イベント + TxMarker |
| Projection | ログ → グラフ構築 |
| Impact | $\text{Reach}(S)$ + 証拠パス |
| Validation | Dangling + 必須制約 |
| Sandbox | 仮説イベント適用 |
| Repair Plan | 影響先の修正案生成 |

---

## 6. 価値命題

> ローコードは「作れる」だけでは価値にならない。
> **運用で変え続けられること**が価値になる。

Palimpsest は変更を「儀式」から「科学」に変える：

- **ログ**で因果を追跡
- **グラフ**で構造を可視化
- **計算**で未来を予測

AI は「気の利いた文章を返す存在」ではなく、**変更の共同責任者**になる。
