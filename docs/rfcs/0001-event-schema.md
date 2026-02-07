# Event Schema

## イベントログの構造定義

---

## 1. 設計原則

| 原則 | 説明 |
|------|------|
| Append-only | 追記のみ、変更・削除しない |
| Atomic | 各イベントは最小単位の操作 |
| Self-contained | 各イベントは単独で意味を持つ |
| Immutable | 記録後は不変 |

---

## 2. イベント型

### 2.1 一覧

| Type | 説明 | Impact Seeds | Validation Seeds |
|------|------|--------------|------------------|
| `NodeAdded` | ノード追加 | $\{n\}$ | $\{n\}$ |
| `NodeRemoved` | ノード削除 | $\{n\}$ | $\{n\}$ |
| `EdgeAdded` | エッジ追加 | $\{v\}$ or $\{u,v\}$ | $\{u,v\}$ |
| `EdgeRemoved` | エッジ削除 | $\{v\}$ or $\{u,v\}$ | $\{u,v\}$ |
| `AttrUpdated` | 属性更新 | $\{n\}$ | $\{n\}$ |
| `TxMarker` | トランザクション境界 | $\emptyset$ | $\emptyset$ |

### 2.2 詳細

#### NodeAdded

```
NodeAdded(id: NodeID, type: NodeType, attrs: Attrs)
```

- `id`: テナント内で一意
- `type`: $\in T = \{\texttt{Entity}, \texttt{Field}, \ldots\}$
- `attrs`: 初期属性

#### NodeRemoved

```
NodeRemoved(id: NodeID)
```

削除前に dangling edge チェックをトリガー。

#### EdgeAdded / EdgeRemoved

```
EdgeAdded(from: NodeID, to: NodeID, label: Label)
EdgeRemoved(from: NodeID, to: NodeID, label: Label)
```

- 方向: `from`（provider）→ `to`（consumer）
- `label`: $\in L = \{\texttt{uses}, \texttt{derives}, \texttt{controls}, \texttt{constrains}\}$

Impact Seeds:
- $l \in \{\texttt{uses}, \texttt{derives}\}$ → $\{v\}$
- $l \in \{\texttt{controls}, \texttt{constrains}\}$ → $\{u, v\}$

#### AttrUpdated

```
AttrUpdated(id: NodeID, attrs: Attrs)
```

`attrs` 内で `null` は削除を意味する。

#### TxMarker

```
TxMarker(tx_id: string, meta: map[string]string)
```

複数操作を論理的にグループ化。

**将来の用途**:
- Projection はこの単位で可視性を進める（途中状態を見せない）
- Sandbox での仮想トランザクション境界

**PoC での扱い**: no-op（Seeds 抽出で空を返し、Replay でスキップ）。可視性制御は将来実装。

---

## 3. ラベル定義

| Label | 意味 | 例 |
|-------|------|-----|
| `uses` | データ依存 | `Field → Expression` |
| `derives` | 構造的所有 | `Entity → Field` |
| `controls` | 振る舞い制御 | `Role → Form` |
| `constrains` | 制約条件 | `Rule → Field` |

---

## 4. ノード型

MVP:

$$
T = \{\texttt{Entity}, \texttt{Field}, \texttt{Form}, \texttt{List}, \texttt{Expression}, \texttt{Role}, \texttt{Param}\}
$$

将来拡張: `CsvSchema`, `ApiSchema`, `Workflow`, `PermissionRule`, ...

---

## 5. JSON Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "oneOf": [
    {
      "type": "object",
      "properties": {
        "type": { "const": "NodeAdded" },
        "node_id": { "type": "string" },
        "node_type": {
          "enum": ["Entity", "Field", "Form", "List", "Expression", "Role", "Param"]
        },
        "attrs": { "type": "object" }
      },
      "required": ["type", "node_id", "node_type"]
    },
    {
      "type": "object",
      "properties": {
        "type": { "const": "NodeRemoved" },
        "node_id": { "type": "string" }
      },
      "required": ["type", "node_id"]
    },
    {
      "type": "object",
      "properties": {
        "type": { "const": "EdgeAdded" },
        "from": { "type": "string" },
        "to": { "type": "string" },
        "label": {
          "enum": ["uses", "derives", "controls", "constrains"]
        }
      },
      "required": ["type", "from", "to", "label"]
    },
    {
      "type": "object",
      "properties": {
        "type": { "const": "EdgeRemoved" },
        "from": { "type": "string" },
        "to": { "type": "string" },
        "label": {
          "enum": ["uses", "derives", "controls", "constrains"]
        }
      },
      "required": ["type", "from", "to", "label"]
    },
    {
      "type": "object",
      "properties": {
        "type": { "const": "AttrUpdated" },
        "node_id": { "type": "string" },
        "attrs": { "type": "object" }
      },
      "required": ["type", "node_id", "attrs"]
    },
    {
      "type": "object",
      "properties": {
        "type": { "const": "TxMarker" },
        "tx_id": { "type": "string" },
        "meta": { "type": "object" }
      },
      "required": ["type", "tx_id"]
    }
  ]
}
```

---

## 6. Go 実装

```go
type EventType int

const (
    EventNodeAdded EventType = iota
    EventNodeRemoved
    EventEdgeAdded
    EventEdgeRemoved
    EventAttrUpdated
    EventTransactionMarker
)

type EdgeLabel string

const (
    LabelUses       EdgeLabel = "uses"
    LabelDerives    EdgeLabel = "derives"
    LabelControls   EdgeLabel = "controls"
    LabelConstrains EdgeLabel = "constrains"
)

type Event struct {
    Type     EventType
    NodeID   NodeID
    NodeType NodeType
    Attrs    Attrs
    FromNode NodeID
    ToNode   NodeID
    Label    EdgeLabel
    TxID     string
    TxMeta   map[string]string
}
```
