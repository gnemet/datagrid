# Datagrid Engineering & UI/UX Standards

This document codifies the mandatory standards for the standalone `datagrid` library, synthesized from the `jiramntr` ecosystem and the original `migr` implementation.

## ⚖️ Engineering Standards

### 1. No-Inline (Clean Code) Policy
- **0% Inline Logic**: No `onclick`, `onchange`, or `style` attributes in HTML.
- **Behavioral Dispatch**: Use the `data-action` pattern in `datagrid.js` to handle all interactive events.
- **Theme-Aware CSS**: Use CSS variables (tokens) exclusively for styling. Standardized tokens include `--dg-accent`, `--dg-border`, and `--dg-card-bg`.

### 2. Forensic DOM Standard
- **Row Metadata**: Every `<tr>` MUST contain a `data-json` attribute containing the full record metadata for AI/Testing extraction.
- **Cell Tagging**: Every `<td>` MUST have a class matching the pattern `col-{field}` for easier selector targeting.
- **State Transparency**: Sortable headers MUST contain a `data-sort` attribute reflecting their current state (`ASC`, `DESC`, `NONE`).

### 3. Catalog-Driven Architecture
- **Metadata First**: Grid structure, labels, and LOVs (List of Values) MUST be driven by JSON catalogs.
- **I18n Compliance**: Labels must support multi-language objects (e.g., `{"en": "...", "hu": "..."}`).
### 4. Strict Backward Compatibility (Additive Only)
- **Library First**: Datagrid is a library consumed by external projects. Stability is the highest priority.
- **Additive Evolution**: New features MUST be strictly additive. Do not modify or remove existing public APIs, JSON schema properties, or established CSS classes.
- **No Breaking Changes**: If a behavior must be changed, introduce it as a new, optional opt-in feature while keeping the original behavior as the default.

---

## 🎨 UI/UX Patterns

### 1. Expert Interactive Grid
- **3-Phase Loop Sort**: Cycles through ASC → DESC → NONE instead of just binary sorting.
- **Shift+Click Hide**: Quick-hide functionality for expert users to declutter views without menus.
- **High-Density Focus**: Minimal whitespace. Table headers MUST use `white-space: nowrap`.
- **JSON Expansion**: Dynamic column injection for nested JSONB data keys to facilitate forensic data analysis.

### 2. Iconographic Precedence
- **Metadata-Driven Icons**: Icons for specific columns MUST NOT be hardcoded in templates. They must be defined in the catalog using the `icon` property.
- **Dual Library Support**: Supports both **FontAwesome** (`fas`) and **Phosphor** (`ph`). Toggled via `iconStyleLibrary` in the catalog.
- **Auto-Prefixing**: Templates MUST handle icon names with or without library prefixes (`fa-`, `ph-`) for maximum catalog flexibility.
- **Graceful Fallback**: Templates MUST check for the existence of an icon in the metadata and fallback to labels only if no icon is defined.
- **Action Buttons**: ALWAYS use icon-only buttons with `title` attributes for tooltips in the toolbar. This should also be driven by action metadata.

### 3. Premium Feedback
- **Hover States**: Interactive zones (resizers, headers) use `var(--dg-accent-light)` with `0.3` opacity.
- **Tabular Nums**: Numeric columns MUST use monospaced fonts (`JetBrains Mono`) and `tabular-nums` for alignment.

---
*Standardized for Datagrid v1.5.1 | Antigravity AI Powered Coding*
