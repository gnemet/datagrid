# Go Datagrid Component (v1.2.0)

A high-performance, metadata-driven datagrid component library for Go and HTMX, optimized for PostgreSQL. Built for high-density, expert-centric interfaces with a strict **No-Tailwind** architectural identity.

## 🚀 Premium Features (v1.2.0)

- **Metadata-Driven UI**: Configure columns, labels, icons, and behavior using localized JSON catalogs.
- **Expert Minimalist Sorting**: 
  - 3-phase sorting (ASC -> DESC -> NONE) with rank indicators.
  - Multi-column support (Ctrl+Click).
  - **Advanced Postgres Sorting**: Full support for `NULLS FIRST` and `NULLS LAST` directives.
  - **Dynamic JSON Sorting**: Sort by nested JSON paths using the `dyn-` prefix (leveraging Postgres `->>` operator).
- **Advanced Search & Thresholds**:
  - Configurable search columns, operators, and similarity thresholds.
  - **Transactional Thresholds**: Automatically sets `pg_trgm.similarity_threshold` within a transaction scope for high-precision fuzzy matching.
- **Robust SQL Generation**:
  - Automatic double-quoting of identifiers to prevent collisions with reserved keywords.
  - Parentheses wrapping for complex searchable column expressions.
- **Forensic DOM Standard**:
  - Rows tagged with `data-json` containing the full record metadata.
  - Cells tagged with `.col-{field}` for easy CSS targeting and scraping.
  - Integrated `escapeClass` logic for deterministic selection of nested JSON fields.
- **Record Detail Panel**: Integrated right-sidebar for high-density metadata inspection with a dedicated toggle button.
- **JSON Key Expansion**: Dynamically expand nested JSON objects into table columns at runtime.
- **Persistence Layer**: Automatic persistence of column visibility, width, order, and sorting in `localStorage`.

## 🛠 Architectural Identity

- **No-Tailwind Policy**: Built with pure CSS semantic classes (`dg-*`) to ensure standalone reliability and zero-dependency styling.
- **Design Tokens**: Standardized shadow (dg-shadow), radius (dg-radius), and Inter typography.
- **HTMX Native**: Zero-refresh updates, pagination, and filtering using HTMX standard triggers.

## 📋 Catalog Configuration

The datagrid behavior is defined by its JSON catalog, validated against the [`datagrid.schema.json`](file:///home/gnemet/GitHub/datagrid/schemas/datagrid.schema.json).

### Example Searchable Config
```json
"searchable": {
    "columns": ["name", "email", "(data->>'role')"],
    "operator": "%",
    "threshold": 0.3
}
```

## 🏗 Setup & Installation

### Prerequisites
- Go 1.21+
- PostgreSQL with `pg_trgm` extension enabled.
- **Critical**: Ensure the `public` schema is included in your connection's `search_path`.

### Database Initialization
```bash
# Enable similarity search extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

### Quick Run
```bash
./build_run.sh
```

## 📄 Documentation
For detailed technical implementation details, see [docs/datagrid.md](file:///home/gnemet/GitHub/datagrid/docs/datagrid.md).

## License
MIT
