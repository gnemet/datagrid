# Agent Rules for Datagrid Project

## Documentation
1.  **Source of Truth**: `docs/project_documentation.md` is the central source of truth for the project.
2.  **Continuous Updates**: Functionality changes (e.g., CursorPool movements, Catalog parsing), key implementation details (e.g., refcursor mode SQL), and verification results MUST be merged into `docs/project_documentation.md` immediately upon completion.
3.  **No Scattering**: Avoid leaving important context solely in `task.md` or walkthroughs. Consolidate it into the project documentation.

## UI/UX Rules (Antigravity Standards)

### Design Philosophy
- **Premium & Elegant**: Clean, high-end interface with generous spacing and professional typography.
- **Harmony**: Use curated, harmonious color palettes (e.g., Slate/Indigo).
- **Theme Support**: Native Dark and Light modes on all components with seamless synchronization.

### Typography & Icons
- **System Font Stack**: Prioritize performance with system fonts.
- **FontAwesome Icons**: Use FontAwesome (`fas`, `fab`, etc.) as the standard icon set for compatibility with adjacent projects.
- **Visual Hierarchy**: Use opacity and weight to distinguish active/inactive states and primary/secondary information.

### Interactions & UX
- **HTMX + CursorPool**: Use HTMX for dynamic table updates and directional cursor commands (FIRST, NEXT, etc.).
- **Draggable Columns**: Support professional data grid interactions like column reordering and resizing.
- **Metadata-Driven**: Column labels, icons, and visibility MUST be driven by the Catalog (`personnel.json`) or backend metadata, validated against `internal/data/schemas/datagrid.schema.json`.

### Clean Code (UI)
- **NO HARDCODING**: **NEVER** hardcode UI strings, labels, or column names in Go or JS. Use the Catalog.
- **Template Separation**: Keep datagrid partials in `pkg/datagrid/ui/templates/partials/datagrid/`.

## Back-end Rules

### Architecture
- **Standard Library**: Prefer Go standard library for core logic.
- **CursorPool Lifecycle**: Maintain strict session state, idle timeouts, and capacity limits for refcursor sessions.
- **Graceful Failures**: Return clean error fragments or log context for 500 errors to aid debugging.

## Configuration & Versioning
1.  **Strict Ignoring**: All `.env*` files MUST be ignored by git.
2.  **Automatic Versioning**: The AI agent MUST automatically increment the `version` in `config.yaml` whenever a new functional feature is added or a blocker is fixed.
3.  **Versioning Pattern**: Use semantic versioning (`major.minor.patch`). Increment `minor` for new features (like CursorPool) and `patch` for fixes.
4.  **Config-Driven**: Connection strings, timeouts, and pool limits MUST be loaded from `config.yaml` and environment variables.
