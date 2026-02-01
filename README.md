# Go Datagrid Component

A high-performance, metadata-driven datagrid component library for Go and HTMX, optimized for PostgreSQL.

## Features

- **Metadata-Driven**: Configure columns, labels, and behavior using JSON catalogs.
- **Expert Minimalist Sorting**: 3-phase sorting (ASC -> DESC -> NONE) with multi-column support (Ctrl+Click).
- **PostgreSQL Similarity Search**: Native fuzzy search support using `pg_trgm`.
- **LOV Support**: Static and dynamic (SQL-based) List of Values for filters.
- **Multi-Language**: Built-in support for multiple languages in labels and LOVs.
- **HTMX Integration**: Zero-refresh updates and pagination.

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL with `pg_trgm` extension.

### Installation
```bash
go get github.com/gnemet/datagrid
```

## Running the Demo

1. Initialize the database:
   ```bash
   PGPASSWORD=yourpass psql -h localhost -U user -d db -f database/init_db.sql
   ```
2. Configure `.env`:
   ```env
   DB_HOST=localhost
   DB_PORT=5433
   DB_USER=root
   DB_PASSWORD=soa123
   DB_NAME=db01
   DB_SCHEMA=datagrid
   ```
3. Run the application:
   ```bash
   ./build_run.sh
   ```

## License
MIT
