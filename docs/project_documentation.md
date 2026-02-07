# Project Documentation: datagrid

## Overview
`datagrid` is a high-performance Go-based table component with advanced metadata-driven UI capabilities, supporting multi-sort, column-chooser, and directional server-side pagination via PostgreSQL cursors.

## Source of Truth
This document is the primary reference for all project features and architectural decisions.

## Architecture & Blueprint

### CursorPool Implementation (v1.5.1)
The `CursorPool` manages database sessions and `refcursor` life-cycles.
- **Tuning**: Configurable `max_connections`, `idle_timeout`, and `abs_timeout`.
- **Capacity**: Explicit limits to prevent session exhaustion.
- **Branding**: Displays `REFCURSOR MODE` and `SCROLLABLE CURSOR ACTIVE` indicators when active.

### Standard Connection Mode (Connectless)
The classic stateless connection method for base implementations.
- **Mechanism**: Direct SQL query execution with `LIMIT`/`OFFSET` for pagination.
- **Indicators**: Displays `Standard Connect Mode` in the UI to distinguish from stateful cursor sessions.
- **Use Case**: Preferred for stateless API integrations and environments where persistent database cursors are not required.

### Technical Implementation Details (v1.5.1)

#### 📄 Pagination & PageSize Logic
The system uses a hierarchical resolution for determining the result set size:
1.  **Request Parameter**: The `limit` query parameter (if provided) has the highest priority.
2.  **Catalog Defaults**: If no parameter is present, the handler falls back to `cat.Datagrid.Defaults.PageSizes[0]`.
3.  **Internal Helper**: The system populates a `PageSize` helper at initialization for template rendering.

#### 📋 List of Values (LOVs)
LOVs drive both UI dropdowns and server-side filter validation:
- **Global vs. Local**: Supports global definitions in `datagrid.lovs` and per-column overrides.
- **Dynamic SQL**: SQL strings in the catalog are executed at runtime. The placeholder `{lang}` is automatically replaced with the active session language.
- **Static Lists**: Localized JSON objects providing fixed value/label mappings.
- **Processing**: Labels are resolved at handler initialization based on the user's `lang` or the English fallback.

#### 🔍 Configuration & Filtering
- **Filter Inference**: The system automatically infers filter types (`text`, `number`, `boolean`, `int_bool`) from the underlying column metadata if not explicitly defined.
- **Dynamic JSON Sorting**: Support for `dyn-` prefix (e.g., `dyn-data.role`) which is translated to PostgreSQL JSONB operators (`->>`) at runtime.
- **Schema Isolation**: Datagrid logic is isolated from MCP extensions via separate schema validation in `internal/data/schemas/`.

### UI Integration Standards
- **Icons**: FontAwesome (`fas`) is the standard icon set for compatibility.
- **Navigation**: Uses directional pagination (FIRST, PRIOR, NEXT, LAST).
- **Aesthetics**: Adopts premium breadcrumb-style headers and glassmorphism-inspired components.

## Work Log & Feature History

### Feb 2026 - Johanna Integration & CursorPool
- **Professionalization**: Enhanced `CursorPool` for production readiness.
- **Documentation**: Adopted the Johanna documentation method with this centralized file.
- **Rules**: Enforced professional development rules via `.agent/rules.md`.
- **Versioning**: Implemented automatic semantic versioning (Current: `1.5.1`).
- **Modernization**: Updated `CursorApp` with premium breadcrumbs and indicators while maintaining FontAwesome compatibility.
- **Structural Integrity**: Migrated core logic to `pkg/datagrid/` to restore external importability (Standard Go Library pattern).
- **Schema Consolidation**: Grouped JSON schemas in `internal/data/schemas/` and isolated MCP keys from Datagrid logic.

## Usage Guide
### Running CursorApp
```bash
DB_MODE=refcursor DB_PORT=5433 DB_USER=root DB_PASSWORD=soa123 DB_NAME=db01 DB_SCHEMA=datagrid go run cmd/cursorapp/main.go
```
The app will be available at `http://localhost:8089`.

## Verification Status
- **Automated**: Verified connection pool tuning and config parsing.
- **Manual**: UI and pagination verified in `CursorApp` (v1.5.1).
