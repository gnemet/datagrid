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
