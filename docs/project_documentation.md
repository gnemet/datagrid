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

### 🌳 Hierarchical Pivot2 Table
An advanced hierarchical data tree grid for exploring nested group summaries.
- **Collapsible Levels**: Nested row levels (e.g. Project -> Issue -> User) that can be expanded or collapsed.
- **Computed Measures**: Runtime evaluation of arithmetic expressions combining multiple aggregated values (`expr`).
- **Conditional Formatting**: Measure-specific CSS classes applied via threshold rules (`cssRules`).
- **Smart Filter**: Integrated client-side search supporting `{column} > value` inline syntax, exact matches, text fallback, and implicit `AND` operators across multiple space-separated terms.

### 📈 Future Roadmap: Graphs
Upcoming versions will include integrated charting (Bar, Line, Pie) directly derived from Pivot/Grid datasets, leveraging the same streaming architecture.

## SQL & Execution Modes

The project standardizes on a **Unified Hybrid SQL** model. SQL is generated via Go templates (`grid.sql.tmpl`) and executed through Postgres wrappers.

### RLS-Aware Initialization
The datagrid includes context-aware session injection. By using `NewHandlerFromDataWithUser()`, the datagrid injects the current authenticated user identity into PostgreSQL prior to executing queries or resolving dynamic SQL LOVs. This ensures full compliance with Row Level Security (RLS) policies defined in the database.

### Logging
All Go code uses `log/slog` for structured logging (`slog.Info`, `slog.Warn`, `slog.Error` with key-value pairs). The legacy `log.Printf` package is not used.

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

## Environment & Secret Management

The project uses **GPG-encrypted environment templates** stored in `opt/envs/`. Raw `.env` files are git-ignored; only `.gpg` artifacts are committed.

### Scripts

| Script | Purpose |
| :--- | :--- |
| `scripts/vault.sh` | GPG AES256 lock/unlock/verify/diff/status with shortcut resolution |
| `scripts/switch_env.sh` | Hostname auto-detection with GPG auto-unlock fallback |
| `scripts/deploy_butalam.sh` | Full 6-phase deployment pipeline to LAN server |

### Environment Switching Flow
```
hostname → opt/envs/.env_{hostname} → .env (root)
              ↑ auto-unlock from .gpg if raw missing
```

### Vault Passphrase Resolution
1. `$VAULT_PASS` environment variable
2. `./.vault_pass` file (project root)
3. `~/.vault_pass` file (home directory)

## Deployment Architecture

### Butalam (LAN Server)
The butalam server is offline (no internet access). Deployment is fully self-contained:

1. **Cross-compile** on dev machine: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`
2. **Package**: binary + config + UI + catalogs + systemd service
3. **Transfer**: `scp` via SSH key (`~/.ssh/butala`)
4. **Install**: Extract to `/opt/datagrid`, create system user, setup systemd
5. **Run**: `systemctl restart datagrid`

Target: `http://sys-butalam01:8085`

### Systemd Service
```ini
[Service]
Type=simple
User=datagrid
WorkingDirectory=/opt/datagrid
ExecStart=/opt/datagrid/bin/datagrid-server
EnvironmentFile=/opt/datagrid/.env
```

