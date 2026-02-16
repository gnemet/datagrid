# Project Documentation: datagrid

## Overview
`datagrid` is a high-performance, metadata-driven UI component library for Go. It enables rapid development of advanced data tables and analytical pivots by consolidating SQL generation logic in Go and offloading execution to optimized PostgreSQL streaming functions.

## Technology Stack

The project leverages a modern, lean, and high-performance stack:

| Layer | Technology | Role |
| :--- | :--- | :--- |
| **Backend** | **Go (Golang)** | Core logic, SQL template rendering, and HTTP handling. |
| **Database** | **PostgreSQL** | Optimized execution wrappers, specialized JSONB/CSV streaming, and stateful cursor management. |
| **Frontend** | **HTMX** | Facilitates seamless AJAX-based partial updates without complex JavaScript frameworks. |
| **Styling** | **Vanilla CSS** | Modern CSS3 with Flexbox/Grid for layout, using a "Glassmorphism" aesthetic. |
| **Icons** | **Phosphor / FA** | Supports Phosphor Icons (primary) and FontAwesome (legacy/fallback). |
| **Templating** | **Go Templates** | Used for both HTML UI partials and server-side SQL generation. |

## Data Visualization Modes

### 📊 Data Grid (Default)
The standard view for record browsing and management.
- **Features**: Multi-column sorting, advanced filtering, column chooser, and detail sidebar.
- **Paging**: Supports both offset-based and directional cursor-based navigation.

### 🔄 Pivot Table
An analytical mode for data aggregation and cross-tabulation.
- **Dimensions**: Configurable row and column dimensions.
- **Measures**: Support for multiple aggregations (SUM, AVG, COUNT, etc.).
- **Interactivity**: Support for dimension swapping and subtotal toggling.

### 📈 Future Roadmap: Graphs
Upcoming versions will include integrated charting (Bar, Line, Pie) directly derived from Pivot/Grid datasets, leveraging the same streaming architecture.

## SQL & Execution Modes

The project standardizes on a **Unified Hybrid SQL** model. SQL is generated via Go templates (`grid.sql.tmpl`) and executed through Postgres wrappers.

### 1. Normal (Connectless) Mode
The stateless connection method, ideal for scalable APIs.
- **Mechanism**: Direct execution of generated SQL using `$1` (JSONB configuration) and `$2` (Parameters).
- **Pagination**: Uses `LIMIT` and `OFFSET`.

### 2. Cursor (Stateful) Mode
Optimized for high-concurrency and large datasets where consistent navigation is required.
- **Mechanism**: Utilizes `CursorPool` to manage `refcursor` life-cycles.
- **Stability**: Prevents deep-paging performance degradation by maintaining server-side state.
- **Indicators**: UI displays `SCROLLABLE CURSOR ACTIVE` when in this mode.

## Datagrid JSON Format (Catalog)

The system is driven by a metadata `Catalog` defined in JSON.

### Catalog Structure
```json
{
  "version": "1.5.2",
  "title": "Personnel Analytics",
  "type": "pivot",
  "datagrid": {
    "defaults": {
      "page_size_default": 25,
      "page_size": [10, 25, 50, 100],
      "sort_column": "id",
      "sort_direction": "desc"
    },
    "searchable": {
      "columns": ["name", "department", "role"],
      "operator": "%",
      "threshold": 0.3
    },
    "columns": {
      "id": { "visible": false },
      "salary": { "css": "col-currency", "display": "Salary" },
      "department": { "display": "Dept", "lov": "department" }
    },
    "pivot": {
      "rows": [{ "column": "department", "css": "dg-dim-dept" }],
      "columns": [{ "column": "gender" }],
      "values": [{ "column": "salary", "func": "SUM", "label": "Total Salary" }],
      "subtotals": true
    }
  },
  "objects": [
    {
      "name": "personnel",
      "columns": [
        { "name": "id", "type": "integer", "primary_key": true },
        { "name": "name", "type": "text" }
      ]
    }
  ]
}
```

### Key Configuration Objects

#### `defaults`
- **`page_size_default`**: The initial number of records per page.
- **`page_size`**: Array of available page size options for the UI.
- **`sort_column` / `sort_direction`**: Initial sort state.

#### `searchable`
- **`columns`**: List of columns included in the global search.
- **`operator`**: PostgreSQL operator for search (e.g., `%` for similarity, `ILIKE` for patterns).
- **`threshold`**: Similarity threshold for the `%` operator.

#### `columns` (Overrides)
- **`visible`**: Boolean to show/hide column.
- **`css`**: Custom CSS classes added to the column cells.
- **`lov`**: Reference to a global LOV or an inline definition.

#### `lovs` (Global LOVs)
- **Static**: Array of value/label pairs. Includes `rowStyle` and `rowClass` for conditional formatting.
- **Dynamic**: A SQL string (containing `{lang}`) that returns `code` and `label`.

#### `pivot`
- **`rows` / `columns`**: Dimensions for the pivot grid. Can include `css` for custom styling.
- **`values`**: Aggregated measures. Requires a column and a function (`SUM`, `AVG`, `MIN`, `MAX`, `COUNT`).
- **`subtotals`**: Boolean to enable/disable row/column subtotals.

#### `operations`
- **`add` / `edit` / `delete`**: Booleans to enable/disable UI actions for record management.

## Streaming Architecture

To maximize memory efficiency, the system avoids loading entire result sets into Go memory.
- **JSON Streaming**: PostgreSQL `datagrid_execute_json` returns `SETOF jsonb`, processed row-by-row in Go.
- **CSV Streaming**: `datagrid_execute_csv` generates formatted CSV lines in the database, streamed directly to the HTTP response writer.

## Verification Status
- **Automated**: Full coverage of SQL generation and alphabetical CSV column sorting.
- **Manual**: Verified in `testapp` (Stateless) and `cursorapp` (Stateful).
