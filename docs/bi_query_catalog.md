# BI Query Catalog — JSON Format Reference

This document describes the JSON catalog format used to define parameterized BI queries in the `internal/data/catalog/` directory. This is the datagrid equivalent of jiramntr's markdown-based BI query system.

## Quick Start

Set `"type": "query"` in any `.json` catalog to enable the parameter form + custom SQL flow:

```json
{
    "version": "1.0",
    "title": "My Report",
    "type": "query",
    "icon": "fas fa-search",
    "parameters": [ ... ],
    "sql": "SELECT ... WHERE col = :param_name",
    "objects": [ ... ]
}
```

---

## Top-Level Fields

| Field | Type | Required | Description |
|:------|:-----|:---------|:------------|
| `version` | string | yes | Schema version (`"1.0"`) |
| `title` | string | yes | English title shown in toolbar |
| `title_hu` | string | no | Hungarian title |
| `type` | string | yes | Must be `"query"` for parameterized queries |
| `icon` | string | no | Font Awesome icon class |
| `description` | string | no | Short description of the query |
| `parameters` | array | yes | Query parameters (can be empty `[]`) |
| `sql` | string | yes | SQL with `:param_name` placeholders |
| `objects` | array | yes | Column metadata for the result set |
| `datagrid` | object | no | Grid display configuration |
| `notes` | array | no | Array of note strings |

---

## Parameters

Each parameter is an object with these fields:

| Field | Type | Required | Description |
|:------|:-----|:---------|:------------|
| `name` | string | yes | Parameter name (matches `:name` in SQL) |
| `type` | string | yes | SQL type: `DATE`, `TEXT`, `INTEGER`, `NUMERIC`, `TIMESTAMPTZ`, `BIGINT` |
| `default` | string | yes | Default value: literal, `NULL`, `CURRENT_DATE`, `CURRENT_TIMESTAMP`, or `Session user` |
| `input` | string | yes | Input control type (see below) |
| `description` | string | no | Hint text shown below the input |
| `label` | string | no | Custom display label (auto-generated from name if empty) |

### Input Types

The `input` field is a simple type keyword. SQL and options go in dedicated fields:

| `input` | Renders | Used with |
|:--------|:--------|:----------|
| `text` | Text input | — |
| `number` | Number input | — |
| `date` | Date picker | — |
| `select` | Static dropdown | `select_options` |
| `lov` | DB-populated dropdown | `lov_query` or `lov_name` |
| `lov` + `TEXT[]` type | Multi-select dropdown | `lov_query` or `lov_name` |
| `lov-tree` | Indented hierarchy select | `lov_query` (3 cols: value, label, depth) |
| `lov-grouped` | Grouped `<optgroup>` select | `lov_query` (3 cols: group, value, label) |
| `constant` | Hidden, server-resolved | `constant` |

> [!TIP]
> Legacy inline prefix syntax (e.g. `"input": "lov:SELECT ..."`, `"input": "select:a,b,c"`) is still supported for backward compatibility.

### Parameter Fields Reference

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | string | Parameter name (used as `:name` in SQL) |
| `type` | string | SQL type: `TEXT`, `INTEGER`, `DATE`, `TIMESTAMPTZ`, `TEXT[]` |
| `default` | string | `NULL`, `CURRENT_DATE`, `CURRENT_TIMESTAMP`, or literal |
| `input` | string | Input type keyword (see table above) |
| `lov_query` | string | SQL query for LOV options |
| `lov_name` | string | Named LOV function → auto-builds `SELECT code, name FROM dwh.<lov_name>()` |
| `select_options` | string | Comma-separated values for select type |
| `constant` | string | Constant key, e.g. `current_user` |
| `description` | string | Hint text shown below the input |
| `label` | string | Display label (auto-generated from name if empty) |

### Multi-Select (Array) Parameters

Set `type` to an array type (e.g. `TEXT[]`) to render a multi-select `<select multiple>`:

```json
{
    "name": "department",
    "type": "TEXT[]",
    "default": "NULL",
    "input": "lov",
    "lov_query": "SELECT DISTINCT department FROM dwh.dim_user_h WHERE department IS NOT NULL ORDER BY 1",
    "description": "Filter to specific departments"
}
```

- When no items selected → substituted as `NULL`
- When items selected → substituted as `ARRAY['val1','val2']`
- SQL pattern: `WHERE (:department IS NULL OR u.department = ANY(:department))`

### Hierarchical Tree Select (`lov-tree`)

`lov_query` must return **3 columns**: `value`, `label`, `depth` (integer). Depth controls visual `──` indentation.

```json
{
    "name": "manager",
    "type": "TEXT",
    "default": "NULL",
    "input": "lov-tree",
    "lov_query": "WITH RECURSIVE org AS (SELECT user_key, full_name, 0 AS depth FROM dwh.dim_user_h WHERE manager_key IS NULL AND is_active UNION ALL SELECT u.user_key, u.full_name, o.depth+1 FROM dwh.dim_user_h u JOIN org o ON u.manager_key = o.user_key WHERE u.is_active) SELECT user_key, full_name, depth FROM org ORDER BY depth, full_name",
    "description": "Select from org hierarchy"
}
```

### Grouped Select (`lov-grouped`)

`lov_query` must return **3 columns**: `group_label`, `value`, `label`. Groups render as `<optgroup>`.

```json
{
    "name": "user",
    "type": "TEXT",
    "default": "NULL",
    "input": "lov-grouped",
    "lov_query": "SELECT department, user_key, full_name FROM dwh.dim_user_h WHERE is_active AND valid_period @> CURRENT_TIMESTAMP ORDER BY 1, 3",
    "description": "Select user by department"
}
```

### Named LOV Functions (`lov_name`)

Instead of writing SQL in `lov_query`, reference a DWH function by name:

```json
{
    "name": "project",
    "type": "TEXT",
    "input": "lov",
    "lov_name": "lov_project",
    "description": "Select project"
}
```

This auto-generates: `SELECT code, name FROM dwh.lov_project()`

### LOV Queries

- **1 column** returned → used as both value and label
- **2 columns** returned → col1 = value (submitted), col2 = label (displayed)

### LOV Functions (recommended)

Reusable functions returning `TABLE(code TEXT, name TEXT)`:

| Function | Example Output |
|:---------|:---------------|
| `dwh.lov_project()` | `MAKIIER – MAK IIER` |
| `dwh.lov_user()` | `kiss.janos – Kiss János (IT)` |
| `dwh.lov_department()` | `IT` |
| `dwh.lov_issue_type()` | `3 – Task` |
| `dwh.lov_status()` | `Open (To Do)` |
| `dwh.lov_priority()` | `3 – Major` |
| `dwh.lov_status_category()` | `To Do` |

---

## SQL

- Parameters are referenced as `:param_name` in the SQL string.
- Values are substituted at runtime (single quotes escaped, `::` casts preserved).
- `NULL` default means the parameter is optional — use `(:param IS NULL OR col = :param)` pattern.
- Numeric types (`INTEGER`, `NUMERIC`, `BIGINT`) are substituted unquoted.
- Text types are automatically quoted with single quotes.

---

## Objects

Defines column metadata so the grid renderer knows types and display labels:

```json
"objects": [{
    "name": "result_set_name",
    "columns": [
        {"name": "column_name", "type": "TEXT", "labels": {"en": "Display Name", "hu": "Magyar név"}}
    ]
}]
```

---

## Datagrid Config

Optional grid display settings:

```json
"datagrid": {
    "defaults": {
        "page_size": [25, 50, 100],
        "sort_column": "column_name",
        "sort_direction": "desc"
    },
    "columns": {
        "numeric_col": {"css": "col-number"},
        "named_col": {"labels": {"en": "English", "hu": "Magyar"}}
    }
}
```

---

## Available Query Catalogs

| Catalog | Title | Parameters |
|:--------|:------|:-----------|
| `daily_worklog_trend` | Daily Worklog Trend | `effective_date`, `days` |
| `department_headcount` | Department Headcount | `effective_date`, `department` (LOV) |
| `employee_hierarchy` | Employee Hierarchy | `effective_date`, `current_user` (constant) |
| `etl_health_check` | ETL Health Check | `limit` |
| `individual_issue_dashboard` | Individual Issue Dashboard | `project_key` (LOV) |
| `issue_aging_report` | Issue Aging Report | `effective_date`, `aging_threshold_days`, `project_key` |
| `issue_status_distribution` | Issue Status Distribution | `effective_date`, `project_key` |
| `issue_type_distribution` | Issue Type Distribution | `project_key` (LOV) |
| `monthly_project_effort` | Monthly Project Effort | `effective_date`, `project_key` |
| `project_issue_summary` | Project Issue Summary | _(none)_ |
| `project_performance_overview` | Project Performance Overview | `effective_date`, `project_key`, `lookback_days` |
| `project_time_allocation` | Project Time Allocation | `effective_date`, `period` (select) |
| `sla_breach_report` | SLA Breach Report | `effective_date`, `lookback_days`, `project_key` |
| `team_worklog_summary` | Team Worklog Summary | `effective_date`, `current_user` (constant), `period` (select) |
| `user_productivity_kpi` | User Productivity KPI | `effective_date`, `current_user` (constant), `lookback_days` |

---

## Full Example

```json
{
    "version": "1.0",
    "title": "Daily Worklog Trend",
    "title_hu": "Napi munkaidő trend",
    "type": "query",
    "icon": "fas fa-chart-area",
    "description": "Daily total hours logged across all projects.",
    "parameters": [
        {
            "name": "effective_date",
            "type": "DATE",
            "default": "2026-01-31",
            "input": "date",
            "description": "End date for the trend"
        },
        {
            "name": "days",
            "type": "INTEGER",
            "default": "30",
            "input": "select:7,14,30,60",
            "description": "Number of days to look back"
        }
    ],
    "sql": "SELECT c.date_actual, c.day_name, ROUND(SUM(f.hours_worked)::numeric, 1) AS total_hours FROM dwh.fact_daily_worklogs_h f JOIN dwh.dim_calendar c ON f.calendar_id = c.id WHERE upper_inf(f.valid_period) AND c.date_actual >= :effective_date::date - :days AND c.date_actual <= :effective_date::date GROUP BY c.date_actual, c.day_name, c.id ORDER BY c.id",
    "objects": [{
        "name": "daily_worklog_trend",
        "columns": [
            {"name": "date_actual", "type": "DATE", "labels": {"en": "Date", "hu": "Dátum"}},
            {"name": "day_name", "type": "TEXT", "labels": {"en": "Day", "hu": "Nap"}},
            {"name": "total_hours", "type": "NUMERIC", "labels": {"en": "Total Hours", "hu": "Összes óra"}}
        ]
    }],
    "datagrid": {
        "defaults": {"page_size": [25, 50, 100], "sort_column": "date_actual", "sort_direction": "asc"},
        "columns": {"total_hours": {"css": "col-number"}}
    },
    "notes": ["Days with zero worklogs won't appear."]
}
```

---

## Markdown → JSON Migration

| Markdown Section | JSON Equivalent |
|:-----------------|:----------------|
| Frontmatter `icon`, `title_en`, `title_hu` | Top-level `icon`, `title`, `title_hu` |
| `## Parameters` table | `parameters` array |
| `## SQL` code block | `sql` string |
| `## COLUMNS` table | `objects[0].columns[].labels` + `datagrid.columns` |
| `## Description` | `description` string |
| `## Notes` bullet points | `notes` string array |
| `.folder` `username` field | _(handled separately by app-level permissions)_ |
