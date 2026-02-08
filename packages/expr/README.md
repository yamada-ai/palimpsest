# expr package

Palimpsest の式言語を提供する。Parse → Analyze → Eval の3段階で処理する。

---

## 概要

```
ソース文字列
    │
    │ Parse()
    ▼
   AST
    │
    │ Analyze(resolver)
    ▼
DepSummary (依存情報 + Diagnostics)
    │
    │ BuildDepEvents()
    ▼
[]core.Event (EdgeAdded イベント列)
```

評価が必要な場合は `Eval()` を使う。

---

## クイックスタート

```go
// 1. パース
ast, diags := expr.Parse(`IF($field:order.subtotal > 0, $field:order.subtotal * 1.1, 0)`)
if len(diags) > 0 {
    // エラー処理
}

// 2. 依存解析
summary := expr.Analyze(ast, resolver, "expr:calc_total", "field:order.total")

// 3. イベント生成
events := expr.BuildDepEvents(summary)
// → [EdgeAdded(field:order.subtotal → expr:calc_total), EdgeAdded(expr:calc_total → field:order.total)]

// 4. 評価（オプション）
value, err := expr.Eval(ast, valueResolver)
```

---

## 式言語の構文

### リテラル

| 型 | 例 |
|----|-----|
| 数値 | `123`, `3.14` |
| 文字列 | `"hello"`, `"with \"escape\""` |
| 真偽値 | `true`, `false` |
| null | `null` |

### 参照（Reference）

`$namespace:path` の形式で設定要素を参照する。

| 構文 | 意味 | 例 |
|------|------|-----|
| `$field:entity.column` | フィールド参照 | `$field:order.subtotal` |
| `$entity:name` | エンティティ参照 | `$entity:order` |
| `$param:name` | パラメータ参照 | `$param:tax_rate` |
| `$rel:relation.attr` | リレーション属性参照 | `$rel:order_product.quantity` |

リレーション参照は `relation.attr` の2セグメントが必須（`$rel:order_product` だけだとエラー）。

### 演算子

| 種類 | 演算子 | 優先順位 |
|------|--------|---------|
| 論理 OR | `\|\|` | 低 |
| 論理 AND | `&&` | |
| 等価 | `==`, `!=` | |
| 比較 | `<`, `>`, `<=`, `>=` | |
| 加減 | `+`, `-` | |
| 乗除 | `*`, `/`, `%` | 高 |
| 単項 | `-`, `!` | 最高 |

### 関数

#### 条件

| 関数 | 説明 | 例 |
|------|------|-----|
| `IF(cond, then, else)` | 条件分岐 | `IF($field:qty > 0, $field:price, 0)` |
| `COALESCE(a, b, ...)` | 最初の非null値 | `COALESCE($field:nick, $field:name)` |

#### 数値

| 関数 | 説明 |
|------|------|
| `ROUND(x)` | 四捨五入 |
| `FLOOR(x)` | 切り捨て |
| `CEIL(x)` | 切り上げ |
| `ABS(x)` | 絶対値 |
| `MIN(a, b, ...)` | 最小値 |
| `MAX(a, b, ...)` | 最大値 |

#### 集計

| 関数 | 説明 |
|------|------|
| `SUM(...)` | 合計 |
| `COUNT(...)` | 件数 |

#### 文字列

| 関数 | 説明 |
|------|------|
| `CONCAT(a, b, ...)` | 文字列結合 |
| `LEFT(s, n)` | 左からn文字 |
| `RIGHT(s, n)` | 右からn文字 |
| `LEN(s)` | 文字数 |
| `TRIM(s)` | 前後の空白除去 |
| `UPPER(s)` / `LOWER(s)` | 大文字/小文字変換 |
| `CONTAINS(s, sub)` | 部分文字列検索 |

#### 日付（構文として許可）

| 関数 | 説明 |
|------|------|
| `TODAY()` | 今日の日付 |
| `NOW()` | 現在日時 |
| `DATE_ADD(d, n, unit)` | 日付加算 |
| `DATE_DIFF(d1, d2, unit)` | 日付差分 |
| `FORMAT_DATE(d, fmt)` | 日付フォーマット |

#### ルックアップ

| 関数 | 説明 |
|------|------|
| `LOOKUP(table, key, column)` | 他テーブルの値を参照 |
| `FILTER(table, cond, ...)` | テーブルをフィルタ |

---

## 依存解析

### ExactDeps vs SchemaDeps

| 種類 | 説明 | 例 |
|------|------|-----|
| **ExactDeps** | 静的に解決できる依存 | `$field:order.subtotal` |
| **SchemaDeps** | 動的に決まる依存（over-approximation） | `LOOKUP($entity:products, key, dynamicColumn)` |

`LOOKUP` の第3引数が変数の場合、どのカラムを参照するか静的に決まらない。
この場合、エンティティ全体を SchemaDeps として記録する（安全側に倒す）。

### エッジの向き

生成されるエッジは `provider → consumer` の向き：

```
依存先(dep) → 式(self) → 出力先(targetField)
```

例: `expr:calc_total` が `field:order.subtotal` を参照し、`field:order.total` に出力する場合

```
field:order.subtotal → expr:calc_total → field:order.total
```

すべて `uses` ラベルで生成される。

---

## インターフェース

### Resolver（依存解析用）

```go
type Resolver interface {
    // $namespace:path を NodeID に解決
    ResolveRef(namespace string, path []string) (core.NodeID, bool)

    // LOOKUP のカラム解決（entity + column名 → field NodeID）
    ResolveEntityField(entityID core.NodeID, column string) (core.NodeID, bool)
}
```

### ValueResolver（評価用）

```go
type ValueResolver interface {
    // $namespace:path を実行時の値に解決
    ResolveValue(namespace string, path []string) (core.Value, bool)
}
```

---

## Diagnostic（エラー情報）

解析中に発生したエラー・警告は `Diagnostic` として返される。

```go
type Diagnostic struct {
    Level   DiagnosticLevel  // DiagError or DiagWarning
    Span    Span             // ソース上の位置（バイトオフセット）
    Message string           // 人間向けメッセージ
    Code    string           // 機械向けコード
}
```

### エラーコード一覧

| Code | 意味 |
|------|------|
| `LEX_ERROR` | 字句解析エラー（不正な文字など） |
| `PARSE_ERROR` | 構文エラー |
| `UNRESOLVED_REF` | 参照が解決できない |
| `UNDEFINED_IDENTIFIER` | 未定義の識別子 |
| `UNKNOWN_FUNCTION` | 未知の関数名 |
| `INVALID_CALL` | 不正な関数呼び出し |
| `BAD_ARITY` | 引数の数が不正 |
| `REL_ATTR_REQUIRED` | `$rel:` に属性指定がない |
| `UNKNOWN_COLUMN` | LOOKUP のカラムが見つからない |

---

## 評価器の制限

評価器（`Eval`）は最小実装であり、以下の制限がある：

- `LOOKUP` / `FILTER` は**評価では未実装**（依存抽出のための構文）
- 日付系関数は未実装
- `SUM` は引数リストを合計、`COUNT` は配列のみ対応

これらは将来的に拡張可能だが、MVP では依存抽出が主目的。

---

## 関連ドキュメント

- [ADR-0008: Expression Front-end](../../docs/adr/0008-expression-front-end.md)
- [理論憲章](../../docs/theory/charter.md) - 依存グラフの設計思想
