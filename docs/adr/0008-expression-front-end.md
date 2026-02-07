# Palimpsest Expression Language — v1.0 MVP Frozen Spec

**ADR#0008: Expression Front-end (Parser / AST / Dependency Extraction)**  
**Status:** Frozen (MVP)  
**Scope:** `packages/expr` — Parser / AST / Resolve / DepExtract  
**Depends on:** ADR#0001 (Event Schema / Dependency Graph / Impact Analysis)

---

## 0. Scope Constraint（最重要の前提）

> **この言語と依存グラフは、設定変更（config changes）のみを対象とする。**
> 行データ（レコード値）の変更は EventLog の対象外であり、依存グラフを通じた再計算をトリガーしない。

これにより、SchemaDeps が「マスタデータの値変更で O(N) 発火する」問題は構造的に発生しない。

---

## 1. 設計原則

| # | 原則 | 根拠 |
|---|------|------|
| 1 | **参照は安定IDのみ** | 文字列から参照を動的生成する構文は存在しない |
| 2 | **依存抽出は AST 走査だけで完結** | 実行不要 |
| 3 | **`Deps_runtime(e, ρ) ⊆ Deps_static(e)`** が文法レベルで保証 | 健全性 |
| 4 | **Pratt parser で実装可能** | 左再帰なし、演算子優先順位で解決 |

---

## 2. 名前空間プレフィックス（閉じたセット）

| Prefix    | 意味                            | NodeType 対応  | 例                            |
|-----------|---------------------------------|----------------|-------------------------------|
| `field:`  | フィールド（カラム・データ項目） | `Field`        | `$field:order.subtotal`       |
| `entity:` | エンティティ（スキーマコンテナ）| `Entity`       | `$entity:products`            |
| `expr:`   | 他の計算式の結果                | `Expression`   | `$expr:calc_tax`              |
| `param:`  | システムパラメータ・定数        | `Param`        | `$param:tax_rate`             |

### 2.1 予約規則

- `$namespace:path` の形式は常に構文的に有効
- 未知の namespace → パース成功、Resolve フェーズでエラー
- 将来の拡張候補: `$view:`, `$rule:`, `$workflow:` 等

### 2.2 entity: の役割

`entity:` は**スキーマコンテナへの参照**として機能する。

- 式の中で `$entity:products` と書くこと自体は稀（通常は `$field:products.unit_price`）
- 主な用途は **SchemaDeps の登録先**（LOOKUP の動的カラム参照など）
- 既存の Core グラフでは `Entity --derives--> Field` の親子関係が成立済み

---

## 3. EBNF（Final）

```ebnf
(* ============================== *)
(*  Palimpsest Expression Grammar *)
(*  Version: 1.0 (MVP Frozen)    *)
(* ============================== *)

(* --- Top Level --- *)
Expression      = ConditionalExpr ;

(* --- 条件式（IFのみ。CASE/IFS/SWITCHはMVP外） --- *)
ConditionalExpr = LogicalOrExpr
                | "IF" "(" Expression "," Expression "," Expression ")" ;

(* --- 論理演算 --- *)
LogicalOrExpr   = LogicalAndExpr { "||" LogicalAndExpr } ;
LogicalAndExpr  = ComparisonExpr { "&&" ComparisonExpr } ;

(* --- 比較 --- *)
ComparisonExpr  = AdditiveExpr { CompOp AdditiveExpr } ;
CompOp          = "==" | "!=" | "<" | ">" | "<=" | ">=" ;

(* --- 算術 --- *)
AdditiveExpr    = MultiplicativeExpr { ("+" | "-") MultiplicativeExpr } ;
MultiplicativeExpr = UnaryExpr { ("*" | "/" | "%") UnaryExpr } ;
UnaryExpr       = ( "-" | "!" ) UnaryExpr
                | PostfixExpr ;

(* --- 後置（関数呼び出し・プロパティアクセス） --- *)
(* FunctionCall は Identifier + "(" で統一。ホワイトリスト判定は Resolve で行う *)
PostfixExpr     = PrimaryExpr { "." Identifier
                              | "(" ArgList? ")" } ;

(* --- 一次式 --- *)
(* NOTE: FunctionCall は独立構文ではない。                    *)
(*   Identifier 単体 → MVPでは Resolve でエラー               *)
(*   Identifier + "(" → 関数呼び出し（PostfixExprで処理）     *)
PrimaryExpr     = Ref
                | Literal
                | Identifier                (* 関数名候補 or 将来の変数 *)
                | "(" Expression ")" ;

(* --- 参照（安定ID） --- *)
Ref             = "$" Namespace ":" QualifiedName ;
Namespace       = "field" | "entity" | "expr" | "param" ;
QualifiedName   = Identifier { "." Identifier } ;

(* --- 引数リスト --- *)
ArgList         = Expression { "," Expression } ;

(* --- リテラル --- *)
Literal         = NumberLiteral
                | StringLiteral
                | BoolLiteral
                | NullLiteral ;

NumberLiteral   = Digit+ [ "." Digit+ ] ;
StringLiteral   = '"' { StringChar } '"' ;
BoolLiteral     = "true" | "false" ;
NullLiteral     = "null" ;

(* --- 基本トークン --- *)
Identifier      = Letter { Letter | Digit | "_" } ;
Letter          = "a".."z" | "A".."Z" | "_" ;
Digit           = "0".."9" ;
StringChar      = (* any char except '"' and '\', or escape sequence *) ;
```

### 3.1 前ドラフトからの変更点

| 変更 | 理由 |
|------|------|
| `CASE` / `CaseBranch` 削除 | IF ネストで十分。分岐が多いなら LOOKUP を使え |
| `FunctionCall` を PrimaryExpr から削除 | PostfixExpr の `(...)` に一本化。AST ノード種の削減 |
| `Identifier` を PrimaryExpr に追加 | 関数名は Identifier → `(` で認識 |
| `Namespace` に `entity` 追加 | SchemaDeps の登録先として必須 |

---

## 4. 許可関数（ホワイトリスト / MVP）

未知の関数名 → パースは通る（Identifier + PostfixExpr の `(`）→ Resolve でエラー。

### 4.1 条件・制御

| 関数       | シグネチャ                         | 依存抽出（over-approx）              |
|------------|-----------------------------------|--------------------------------------|
| `IF`       | `IF(cond, then, else) → T`        | `{cond} ∪ {then} ∪ {else}`          |
| `COALESCE` | `COALESCE(a, b, ...) → T`         | `{a} ∪ {b} ∪ ...`                   |

### 4.2 算術

`ROUND`, `FLOOR`, `CEIL`, `ABS`, `MIN`, `MAX`, `SUM`

### 4.3 文字列

`CONCAT`, `LEFT`, `RIGHT`, `LEN`, `TRIM`, `UPPER`, `LOWER`, `CONTAINS`

### 4.4 日付

`TODAY`, `NOW`, `DATE_ADD`, `DATE_DIFF`, `FORMAT_DATE`

### 4.5 テーブル操作（依存抽出に特別ルールあり）

| 関数     | シグネチャ                                          | 依存抽出                                               |
|----------|----------------------------------------------------|---------------------------------------------------------|
| `LOOKUP` | `LOOKUP(table_ref, key_expr, column_spec) → T`     | 下記参照                                                |
| `FILTER` | `FILTER(table_ref, predicate_expr) → Array`         | `{table_ref} ∪ Deps(predicate_expr)`                   |
| `COUNT`  | `COUNT(array_expr) → Number`                        | `Deps(array_expr)`                                      |

**LOOKUP の精密化ルール:**

| 第3引数（column_spec）| ExactDeps                                | SchemaDeps           |
|-----------------------|------------------------------------------|----------------------|
| 文字列リテラル `"unit_price"` | `{table_ref.unit_price, key_expr}` | なし                 |
| 式（動的）            | `{key_expr, column_spec}`                | `{entity:table_ref}` |

**補足**: `table_ref` が `entity:` を指す場合、Resolve フェーズで
`entity + 文字列リテラル` から `field:{entity}.{column}` を合成し、
ExactDeps に登録する。

### 4.6 禁止（言語仕様として存在しない）

| 禁止構文     | 理由                                           |
|-------------|------------------------------------------------|
| `INDIRECT`  | 動的参照生成 → `Deps_runtime ⊆ Deps_static` が壊れる |
| `EVAL`      | 同上                                           |
| 再帰        | 関数定義構文がないので構造的に不可能            |
| `FOR`/`WHILE` | ループなし。集合操作は FILTER/COUNT で        |
| 外部 I/O    | `HTTP`, `FETCH` なし                           |

### 4.7 @dynamic（将来拡張・隔離枠）

`@dynamic` アノテーション付きの式のみ、動的参照を許可する（MVP外）。

- 影響解析は**ワイルドカード依存**（`entity:*` 等）として扱う
- UI で「精度が落ちる」ことを明示
- デフォルトパスでは `Deps_runtime ⊆ Deps_static` を絶対に壊さない

---

## 5. AST ノード定義

```
ASTNode
  ├─ Span: SourceSpan       (* 必須: UTF-8 バイトオフセット *)
  ├─ Kind: NodeKind
  └─ Children / Payload（Kind別）

SourceSpan {
    Start int    // 0-indexed, UTF-8 byte offset (inclusive)
    End   int    // 0-indexed, UTF-8 byte offset (exclusive)
}
```

### 5.1 NodeKind

```
NodeKind =
  | Ref { namespace: string, path: []string, resolved_id: NodeID? }
  | Literal { value: Value, lit_type: Type }
  | Identifier { name: string }
  | BinaryOp { op: Operator }
  | UnaryOp { op: Operator }
  | Call { callee: ASTNode(Identifier), args: []ASTNode }
  | If { cond: ASTNode, then_branch: ASTNode, else_branch: ASTNode }
  | PropertyAccess { object: ASTNode, property: string }
  | Group { inner: ASTNode }
```

### 5.2 前ドラフトからの変更点

| 変更 | 理由 |
|------|------|
| `FuncCall` → `Call` | callee は Identifier ノード。PostfixExpr の `(...)` から生成 |
| `Case` 削除 | CASE 文法削除に伴い不要 |
| `Identifier` ノード追加 | 関数名候補 / 将来の変数として PrimaryExpr に必要 |

### 5.3 Ref ノードの詳細

```
Ref {
  namespace:     string       // "field" | "entity" | "expr" | "param"
  path:          []string     // ["order", "subtotal"]
  resolved_id:   NodeID?      // Resolve 後に埋まる（パース直後は nil）
  expected_type: Type?        // Typecheck 後に埋まる
  span:          SourceSpan   // "$field:order.subtotal" 全体の位置
}
```

- `resolved_id == nil` → 未解決参照（未解決参照インデックスに登録）
- `resolved_id != nil` → 依存辺の張り先が確定

### 5.4 型（MVP）

```
Type = Unknown | Number | String | Bool | Date | Null | Array | Object | Enum
```

MVPでは「危険な破壊（Number→String 等）」の検知のみ。厳密な型検査は v2。

---

## 6. コンパイルパイプライン（MVP: 3段）

```
                     ┌──────────────────────────────────────────┐
  source string ──→  │ 1. Parse (Pratt)                         │ ──→ AST (with Span)
                     │    - Ref: namespace + path まで分解       │
                     │    - Identifier: 関数名候補として保持     │
                     │    - resolved_id = nil                    │
                     └──────────────────────────────────────────┘
                                      │
                                      ▼
                     ┌──────────────────────────────────────────┐
  NodeID registry ─→ │ 2. Resolve + DepExtract                  │ ──→ AST (resolved)
  Function          │    - Ref.path → NodeID lookup             │     + DepSummary
  whitelist ──────→ │    - Identifier("SUM") + Call → 検証      │     + Diagnostics
                     │    - Identifier 単体 → Diagnostic(Error)  │
                     │    - 未知関数 → Diagnostic(Error)         │
                     │    - 未解決Ref → 未解決参照インデックス    │
                     │    - 解決済Ref → DepSummary に追加        │
                     │    - 軽量 Typecheck（明らかな型不一致）   │
                     └──────────────────────────────────────────┘
                                      │
                                      ▼
                     ┌──────────────────────────────────────────┐
                     │ 3. Evaluate                               │ ──→ Value
                     │    - AST tree-walk interpreter            │
                     │    - (将来: bytecode/IR)                  │
                     └──────────────────────────────────────────┘
```

---

## 7. DepSummary（依存要約）— Core との契約

```go
package expr

import "github.com/user/palimpsest/pkg/core"

// Span represents a range in the source string.
// UTF-8 byte offsets, 0-indexed, [Start, End).
type Span struct {
    Start int
    End   int
}

// DepSummary is the output of static analysis.
// This is the contract between packages/expr and packages/core.
type DepSummary struct {
    // SelfID: The NodeID of this expression in the graph.
    SelfID core.NodeID

    // TargetField: Where the expression result is stored.
    // Edge: SelfID --uses--> TargetField
    TargetField core.NodeID

    // ExactDeps: Dependencies on specific node values.
    // Edge: dep --uses--> SelfID
    // Trigger: AttrUpdated(dep) → this expression is in impact set.
    ExactDeps []DepEntry

    // SchemaDeps: Dependencies on container structure (schema).
    // Registered against Entity nodes (not individual fields).
    // Trigger: AttrUpdated(child field) propagates to Entity via
    //          seed expansion (see §8), then to expressions with SchemaDeps.
    // Does NOT trigger on: row-level data changes (out of scope, see §0).
    SchemaDeps []DepEntry

    // Unresolved: References that could not be resolved to a NodeID.
    // Stored in a separate index: name → []exprID.
    // When the name becomes valid, the expression must be re-compiled.
    Unresolved []UnresolvedRef

    // Diagnostics: Errors and warnings from parsing, resolution, type checking.
    Diagnostics []Diagnostic
}

// NOTE: Core グラフ上の依存辺は `uses` のみを使用する。
// SchemaDeps / ExactDeps の区別は DepSummary 側で保持し、グラフには反映しない。

// DepEntry represents a single dependency edge.
type DepEntry struct {
    NodeID core.NodeID
    Span   Span // Source location of the Ref that caused this dependency.
}

// UnresolvedRef is a reference that could not be resolved.
type UnresolvedRef struct {
    Namespace string   // "field", "entity", "expr", "param"
    Path      []string // ["order", "subtotal"]
    Span      Span
}

// DiagnosticLevel indicates severity.
type DiagnosticLevel int
const (
    DiagError DiagnosticLevel = iota
    DiagWarning
)

// Diagnostic is a compiler message with source location.
type Diagnostic struct {
    Level   DiagnosticLevel
    Span    Span
    Message string
    Code    string // e.g. "UNRESOLVED_REF", "TYPE_MISMATCH", "UNKNOWN_FUNCTION"
}
```

### 7.1 DepSummary → グラフ辺への変換

```
for each dep in summary.ExactDeps:
    emit EdgeAdded(from=dep.NodeID, to=summary.SelfID, label="uses")

for each dep in summary.SchemaDeps:
    emit EdgeAdded(from=dep.NodeID, to=summary.SelfID, label="uses")
    // SchemaDeps かどうかは DepSummary 側で管理。
    // グラフ上のラベルは "uses" で統一（BFS が単純になる）。

emit EdgeAdded(from=summary.SelfID, to=summary.TargetField, label="uses")
```

**エッジラベルの方針:**
- グラフ上は **`uses` 一本**。`uses_schema` という別ラベルは作らない。
- ExactDeps か SchemaDeps かの区別は `DepSummary` 側（＝ `packages/expr` の責務）で保持。
- BFS（Impact Analysis）は全 `uses` 辺を辿る。SchemaDeps の区別が必要な場面（将来の精密化）では DepSummary を参照する。
- 既存の Core エッジラベル体系（`uses` / `derives` / `controls` / `constrains`）と衝突しない。

---

## 8. SchemaDeps の伝播経路

### 8.1 問題

フィールドの型が変更された場合（`AttrUpdated` on `field:products.price`）、その親 `entity:products` に SchemaDeps を持つ式に伝播させる必要がある。しかし現行の BFS は `field:products.price` を seed として辺を辿るだけなので、`entity:products` には到達しない。

### 8.2 解法：Seed 拡張方式

`ImpactSeeds()` を拡張し、**Field の AttrUpdated は親 Entity も seed に含める**。

```go
func (e Event) ImpactSeeds(graph *Graph) []NodeID {
    switch e.Type {
    case EventAttrUpdated:
        seeds := []NodeID{e.NodeID}
        // Field の変更は、親 Entity もseedに追加
        // （スキーマコンテナへの SchemaDeps を持つ式に伝播させるため）
        if e.NodeType == NodeField {
            if parent := graph.ParentEntity(e.NodeID); parent != "" {
                seeds = append(seeds, parent)
            }
        }
        return seeds
    // ... 他のケースは既存通り
    }
}
```

`graph.ParentEntity()` は `derives` 辺を逆引きして取得（`Entity --derives--> Field` の逆）。

### 8.3 この方式を選んだ理由

| 方式 | 利点 | 欠点 |
|------|------|------|
| ① Schema コンテナノードに辺を張る | 理論的に美しい | Field→Entity の逆辺が `derives` と衝突。`uses` で張ると意味が歪む |
| **② Seed 拡張（採用）** | **辺を追加しない。既存グラフ構造を変えない** | seed の由来が仕様を読まないと分からない → ADR で明記して解決 |

### 8.4 トリガーまとめ

| 依存リスト  | 依存先の例                | トリガーイベント                     | Core の振る舞い         |
|------------|--------------------------|-------------------------------------|------------------------|
| ExactDeps  | `field:order.subtotal`   | `AttrUpdated`（値 or 型の変更）      | Impact あり（再計算）   |
| ExactDeps  | `field:order.subtotal`   | `NodeRemoved`                       | Impact あり（参照切れ） |
| SchemaDeps | `entity:products`        | `NodeAdded`（カラム追加）           | Impact あり             |
| SchemaDeps | `entity:products`        | `NodeRemoved`（カラム削除）         | Impact あり             |
| SchemaDeps | `entity:products`        | Seed 拡張で到達（子 Field の型変更）| Impact あり             |

**「行データの値変更」は EventLog の対象外（§0）なので、SchemaDeps が O(N) 発火する問題は構造的に発生しない。**

---

## 9. 未解決参照インデックス

依存グラフ（Core）は**実在するノード ID にだけ辺を張る**。未解決参照は別インデックスで管理。

```
UnresolvedIndex: map[string][]core.NodeID
// key: "field:order.discount"（namespace:path の文字列表現）
// value: この名前を参照している式の NodeID リスト
```

### 9.1 解決フロー

1. `NodeAdded(field:order.discount)` が EventLog に入る
2. UnresolvedIndex を引いて、該当する式の NodeID リストを取得
3. 該当する式を**再コンパイル**（Parse → Resolve+DepExtract）
4. DepSummary が更新され、グラフに辺が張られる
5. UnresolvedIndex からエントリを削除

---

## 10. 式の例と依存抽出

### 例1: 単純な計算

```
$field:order.subtotal * (1 + $param:tax_rate)
```

| | 内容 |
|---|---|
| ExactDeps | `field:order.subtotal`, `param:tax_rate` |
| SchemaDeps | なし |

### 例2: IF（over-approx）

```
IF($field:order.is_domestic,
   $field:order.subtotal * $param:domestic_tax,
   $field:order.subtotal * $param:international_tax)
```

| | 内容 |
|---|---|
| ExactDeps | `field:order.is_domestic`, `field:order.subtotal`, `param:domestic_tax`, `param:international_tax` |
| SchemaDeps | なし |

実行時は片方しか評価しないが、静的には全分岐を依存に含む（安全側）。

### 例3: LOOKUP（リテラル精密化）

```
LOOKUP($entity:products, $field:order.product_id, "unit_price")
```

| | 内容 |
|---|---|
| ExactDeps | `field:products.unit_price`, `field:order.product_id` |
| SchemaDeps | なし |

第3引数がリテラル → `field:products.unit_price` に精密化。

### 例4: LOOKUP（動的カラム → SchemaDeps）

```
LOOKUP($entity:products, $field:order.product_id, $param:target_column)
```

| | 内容 |
|---|---|
| ExactDeps | `field:order.product_id`, `param:target_column` |
| SchemaDeps | `entity:products` |

カラム名が動的 → entity 全体にスキーマ依存。

### 例5: IF ネスト（CASE の代替）

```
IF($field:order.region == "JP",
   $field:order.subtotal * $param:jp_tax,
   IF($field:order.region == "EU",
      $field:order.subtotal * $param:eu_tax,
      $field:order.subtotal * $param:intl_tax))
```

| | 内容 |
|---|---|
| ExactDeps | `field:order.region`, `field:order.subtotal`, `param:jp_tax`, `param:eu_tax`, `param:intl_tax` |
| SchemaDeps | なし |

---

## 11. 安全装置（Limits）

| 制約                     | MVP 上限       | 根拠                            |
|-------------------------|---------------|--------------------------------|
| 式の文字数               | 4,096 chars   | パース時間を制限                |
| AST ノード数             | 1,000 nodes   | 巨大な式の検知                  |
| 関数ネスト深度            | 32 levels     | スタックオーバーフロー防止      |
| 評価ステップ数            | 10,000 steps  | 無限ループ相当の防止            |
| 1式あたりの依存数         | 256 deps      | 過大依存の警告                  |

ネスト深度 32 は IF のネストにも適用される。32 分岐を超える条件分岐は「LOOKUP + マスタテーブル」で設計すべきであり、式でやるべきではない。

---

## 12. 将来拡張（この文法が壊さずに受け入れられるもの）

| 拡張 | 影響範囲 |
|------|---------|
| 新しい名前空間（`$view:`, `$rule:`） | Namespace の enum に追加するだけ |
| CASE/SWITCH 構文 | ConditionalExpr に分岐を追加。AST に Case ノード追加 |
| 配列リテラル `[1, 2, 3]` | PrimaryExpr に追加 |
| ラムダ（FILTER 用）`FILTER(..., (row) => row.amount > 100)` | 構文追加。依存抽出は body を走査するだけ |
| tree-sitter 移行 | AST が Span（バイトオフセット）を持つ限り、パーサ実装の差し替えは内部の話 |
| Typecheck 独立フェーズ化 | Resolve と DepExtract を分離し、間に Typecheck を挟む |
| trace-based 依存（v2） | DepSummary に `TraceDeps []DepEntry` を追加して、静的依存との差分を可視化 |
| `@dynamic` アノテーション | パーサに構文追加。SchemaDeps としてワイルドカード依存を登録 |
| Seed 拡張 → Schema コンテナ辺方式への移行 | DepSummary → グラフ辺変換を変更するだけ。EBNF は不変 |

---

## 13. ADR Summary

### Context

Palimpsest の依存グラフ（ADR#0001）に式（Expression）を接続するためのフロントエンド（Parser / AST / 依存抽出）が必要。式の設計が依存グラフの健全性を決定する。

### Decision

1. **参照は安定 ID のみ**（`$namespace:path` 形式）。INDIRECT / EVAL は禁止
2. **名前空間は `field` / `entity` / `expr` / `param` の 4 種**（MVP）
3. **依存抽出は静的（AST 走査）**。`Deps_runtime ⊆ Deps_static` を保証
4. **動的依存は over-approx**。IF は全分岐依存、LOOKUP はリテラル精密化 / 動的ワイルドカード
5. **SchemaDeps の伝播は Seed 拡張方式**（Field の AttrUpdated で親 Entity を seed に追加）
6. **エッジラベルは `uses` で統一**。ExactDeps / SchemaDeps の区別は DepSummary 側で管理
7. **EventLog は設定変更のみ対象**。行データの値変更はスコープ外
8. **条件分岐は IF のみ**（MVP）。CASE/SWITCH は将来拡張
9. **関数呼び出しは PostfixExpr に一本化**。ホワイトリスト判定は Resolve フェーズ

### Consequences

- False positive（過大評価）はあり得るが、false negative（過小評価）はない → 安全
- trace-based 依存（精密化）は将来拡張
- `@dynamic` は隔離された拡張ポイントとして予約

### Alternatives Rejected

| 案 | 却下理由 |
|----|---------|
| 文字列正規表現で依存抽出 | 構文エラーに弱い。位置情報が取れない |
| 実行トレースのみ | 監査・再現性・説明責任が弱い |
| フル Excel 互換（INDIRECT 等） | `Deps_runtime ⊆ Deps_static` が壊れる |
| エッジラベルに `uses_schema` を新設 | BFS が複雑化。`uses` 一本の方が実装が単純 |
| Schema コンテナノードに逆辺を張る | `derives` との意味的衝突。辺が増える |
| CASE/IFS を MVP に含める | IF ネストで十分。AST ノード種が増える |
