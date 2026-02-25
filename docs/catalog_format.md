# Datagrid Catalog JSON Format

The Datagrid is driven by a metadata-based configuration system called the **Catalog**. This JSON file defines everything from database table mapping to UI aesthetics.

## Root Properties

| Property | Type | Description |
| :--- | :--- | :--- |
| `version` | `string` | Catalog specification version (e.g., `"1.5"`). |
| `title` | `string` | Main header title for the page. |
| `icon` | `string` | Page/Tab icon (Phosphor or FontAwesome). |
| `type` | `string` | Default view mode: `"grid"` (default) or `"pivot"`. |
| `datagrid` | `object` | Core visualization configuration. |
| `objects` | `array` | Table/View metadata (names and types). |

---

## The `datagrid` Object

### `defaults`
Defines the initial state of the grid.
- `page_size_default` (`int`): Records per page on load.
- `page_size` (`array`): Available options for the page size selector.
- `sort_column` (`string`): Field name for initial ordering.
- `sort_direction` (`string`): `"asc"`, `"desc"`, or Postgres nulls-ordering variants.

### `searchable`
Configures global fuzzy search.
- `columns` (`array`): Field names to be indexed.
- `operator` (`string`): Usually `%` (similarity) or `ILIKE`.
- `threshold` (`float`): Similarity threshold (0.0 to 1.0).

### `columns` (Overrides)
Fine-tune UI behavior per column. The key is the field name.
- `visible` (`bool`): Toggle default visibility.
- `css` (`string`): custom classes (e.g., `"col-id"`, `"col-currency"`).
- `display` (`string`): Header label or pattern (e.g., `"%name% (%role%)"`).
- `labels` (`object`): Multi-lang headers (e.g., `{"en": "Name", "hu": "Név"}`).
- `icon` (`string`): Icon shown next to header text.
- `lov` (`string\|array`): Reference to a global LOV or inline array.

---

## Analytics: `pivot` configuration

The `pivot` object allows deep analytical cross-tabulation.

```json
"pivot": {
  "rows": [{"column": "department", "css": "bold"}],
  "columns": [{"column": "gender"}],
  "values": [
    {"column": "salary", "func": "SUM", "label": "Total Salary"}
  ],
  "subtotals": true,
  "multiplier": 1.0
}
```

- **Dimensions**: Defined in `rows` and `columns`. Supports simple strings or objects with `css`.
- **Values**: Aggregated metrics. Requires `column` and `func` (`SUM`, `AVG`, `COUNT`, `MIN`, `MAX`).
- **Subtotals**: Enables hierarchical totals rendering.

---

## Analytics: `pivot2` configuration

The `pivot2` object enables hierarchical tree grids with advanced measure configurations.

```json
"pivot2": {
  "levels": [
    {"column": "project_name", "label": "Project"},
    {"column": "issue_key", "label": "Issue"}
  ],
  "values": [
    {"column": "estimated_hours", "func": "SUM", "label": "Est. Hours"},
    {"column": "logged_hours", "func": "SUM", "label": "Logged Hours"},
    {"expr": "Est. Hours - Logged Hours", "label": "Remaining", "cssRules": [
      {"when": "< 0", "class": "text-danger"}
    ]}
  ]
}
```

- **Levels**: Array of `{column, label}` defining the nested row hierarchy.
- **Values**: Aggregated or computed metrics.
  - Basic aggregations use `column` and `func`.
  - Computed measures use `expr` (arithmetic expressions referencing other measure labels).
  - Both types support `format` (e.g., `"%.2f"`), `showAt` (array of depth levels to render the measure, e.g. `[0]`), and `cssRules` (array of `{when, class}` thresholds).

---

## Data Integration: `lovs`

Lists of Values (LOVs) power dropdown filters and conditional row styling.

### Static LOV
```json
"status": [
  {"value": "A", "label": "Active", "rowClass": "table-success"},
  {"value": "I", "label": "Inactive", "rowStyle": "opacity: 0.5"}
]
```

### Dynamic LOV (SQL-driven)
```json
"department": "SELECT id as code, name as label FROM departments WHERE active = true"
```
*Note: `{lang}` in SQL is automatically replaced by the current session language.*

---

## Database Mapping: `objects`

Defines the structure of the data source.
- `name`: Table or View name in PostgreSQL.
- `columns`: Array of `{name, type, primary_key}` objects.
