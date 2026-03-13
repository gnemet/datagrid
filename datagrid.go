package datagrid

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// UIAssets embeds the standardized CSS, JS and Templates
//
//go:embed ui
var UIAssets embed.FS

func sanitizeParam(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
func (h *Handler) ParseParams(r *http.Request) RequestParams {
	q := r.URL.Query()
	limit := 10
	if l := q.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	} else if len(h.Config.Defaults.PageSizes) > 0 {
		limit = h.Config.Defaults.PageSizes[0]
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	filters := make(map[string][]string)
	for key, values := range q {
		if key != "search" && key != "sort" && key != "limit" && key != "offset" && key != "code" && key != "_" {
			filters[key] = values
		}
	}

	return RequestParams{
		Search:  q.Get("search"),
		Sort:    q["sort"],
		Filters: filters,
		Limit:   limit,
		Offset:  offset,
	}
}

// RenderPivot2 renders a hierarchical pivot2 tree-grid from pre-fetched records.
// This is the public API for external callers (e.g. jiramntr's BIQueryExecuteHandler).
// It writes the rendered HTML directly to w.
func RenderPivot2(w io.Writer, records []map[string]interface{}, cfg *Pivot2Config, title string, lang string) error {
	pivotRes := Pivot2Data(records, cfg)

	res := &TableResult{
		Title:        title,
		ViewMode:     "pivot2",
		Pivot2Result: pivotRes,
		Lang:         lang,
		CurrentLang:  lang,
	}
	// Propagate Pivot2Config.Links so the template can resolve $.Config.Links
	if cfg != nil && cfg.Links != nil {
		res.Config.Links = cfg.Links
	}

	funcs := TemplateFuncs()
	tmpl, err := template.New("pivot2_standalone").Funcs(funcs).ParseFS(UIAssets,
		"ui/templates/partials/datagrid/pivot2.html",
	)
	if err != nil {
		return fmt.Errorf("pivot2 template parse error: %w", err)
	}

	return tmpl.ExecuteTemplate(w, "datagrid_pivot2", res)
}

// Handler handles the grid data requests
type Handler struct {
	DB                  *sql.DB
	TableName           string
	Columns             []UIColumn
	Config              DatagridConfig
	Catalog             Catalog
	Lang                string
	IconStyleLibrary    string
	ListEndpoint        string // Default endpoint for HTMX updates
	PivotEndpoint       string // Endpoint for pivot data
	ExecuteEndpoint     string // Endpoint for query execution
	LOVChooserThreshold int
	AppName             string
	Catalogs            map[string]string
	QueryParams         []QueryParam // Resolved parameters (with LOV options populated)
	QuerySQL            string       // Raw SQL template with :param placeholders
	IsQueryMode         bool         // true when catalog type == "query"
	CurrentUser         string       // Set by host app for constant:current_user resolution
}

func NewHandler(db *sql.DB, tableName string, cols []UIColumn, cfg DatagridConfig) *Handler {
	return &Handler{
		DB:        db,
		TableName: tableName,
		Columns:   cols,
		Config:    cfg,
	}
}

// NewHandlerFromCatalog initializes a Handler using a MIGR/JiraMntr JSON catalog file
func NewHandlerFromCatalog(db *sql.DB, catalogPath string, lang string) (*Handler, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	return NewHandlerFromData(db, data, lang)
}

// NewHandlerFromDataWithUser initializes a Handler with user context for RLS-aware LOV resolution.
func NewHandlerFromDataWithUser(db *sql.DB, data []byte, lang string, currentUser string) (*Handler, error) {
	h, err := NewHandlerFromData(db, data, lang)
	if err != nil {
		return nil, err
	}
	h.CurrentUser = currentUser

	// Re-resolve LOV queries that reference :current_user (RLS)
	for i := range h.QueryParams {
		lovSQL := h.QueryParams[i].resolvedLOVQuery()
		if lovSQL != "" && strings.Contains(lovSQL, ":current_user") {
			lovSQL = strings.ReplaceAll(lovSQL, ":current_user", "'"+strings.ReplaceAll(currentUser, "'", "''")+"'")
			h.QueryParams[i].Options = nil // Clear stale options

			itype := h.QueryParams[i].InputType()
			rows, err := db.Query(lovSQL)
			if err == nil {
				defer rows.Close()
				cols, _ := rows.Columns()
				nCols := len(cols)
				for rows.Next() {
					switch {
					case itype == "lov-tree" && nCols >= 3:
						var val, label string
						var depth int
						if err := rows.Scan(&val, &label, &depth); err == nil {
							h.QueryParams[i].Options = append(h.QueryParams[i].Options, LOVItem{Value: val, Label: label, Depth: depth})
						}
					case nCols >= 2:
						var val, label string
						if err := rows.Scan(&val, &label); err == nil {
							h.QueryParams[i].Options = append(h.QueryParams[i].Options, LOVItem{Value: val, Label: label})
						}
					default:
						var val string
						if err := rows.Scan(&val); err == nil {
							h.QueryParams[i].Options = append(h.QueryParams[i].Options, LOVItem{Value: val, Label: val})
						}
					}
				}
			} else {
				slog.Error("RLS LOV query error for param", "name", h.QueryParams[i].Name, "error", err)
			}
		}
	}

	return h, nil
}

// NewHandlerFromData initializes a Handler using MIGR/JiraMntr JSON data (bytes)
func NewHandlerFromData(db *sql.DB, data []byte, lang string) (*Handler, error) {
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, err
	}

	// Populate PageSize helper for templates
	if len(cat.Datagrid.Defaults.PageSizes) > 0 {
		cat.Datagrid.Defaults.PageSize = cat.Datagrid.Defaults.PageSizes[0]
	}

	// v2.0: build objects from datagrid.columns + source when objects[] is absent
	if len(cat.Objects) == 0 && cat.Source != "" {
		var cols []ColumnDef
		for name, def := range cat.Datagrid.Columns {
			cols = append(cols, ColumnDef{
				Name:   name,
				Type:   def.Type,
				Labels: def.Labels,
			})
		}
		cat.Objects = []ObjectDef{{
			Name:    cat.Source,
			Columns: cols,
		}}
	}

	if len(cat.Objects) == 0 {
		return nil, fmt.Errorf("no objects found in catalog")
	}

	obj := cat.Objects[0]

	// Merge Defaults.Filters into Config.Filters (Fix for nested filters in catalog)
	if cat.Datagrid.Filters == nil {
		cat.Datagrid.Filters = make(map[string]FilterDef)
	}

	colTypes := make(map[string]string)
	for _, col := range obj.Columns {
		colTypes[col.Name] = strings.ToLower(col.Type)
	}

	for field, val := range cat.Datagrid.Defaults.Filters {
		if _, exists := cat.Datagrid.Filters[field]; !exists {
			// If value implies enabled (true or object), add definition
			enabled := false
			if b, ok := val.(bool); ok && b {
				enabled = true
			} else if _, ok := val.(map[string]interface{}); ok {
				enabled = true
			}

			if enabled {
				fd := FilterDef{Column: field}
				// Infer type from column definition
				if cType, ok := colTypes[field]; ok {
					switch cType {
					case "boolean", "bool":
						fd.Type = "boolean"
					case "int_bool":
						fd.Type = "int_bool"
					case "integer", "int", "numeric":
						fd.Type = "number"
					default:
						fd.Type = "text"
					}
				} else {
					fd.Type = "text"
				}
				cat.Datagrid.Filters[field] = fd
			}
		}
	}

	uiCols := []UIColumn{}

	for _, col := range obj.Columns {
		label := col.Name
		if l, ok := col.Labels[lang]; ok {
			label = l
		} else if l, ok := col.Labels["en"]; ok {
			label = l
		}

		visible := true
		var icon string
		var overrideLov interface{}
		if override, ok := cat.Datagrid.Columns[col.Name]; ok {
			if override.Visible != nil {
				visible = *override.Visible
			}
			icon = override.Icon
			if l, ok := override.Labels[lang]; ok {
				label = l
			} else if l, ok := override.Labels["en"]; ok {
				label = l
			} else if override.Display != "" && !strings.Contains(override.Display, "%") {
				// Fallback: If no label but display is a static string, use it as label
				label = override.Display
			}
			overrideLov = override.LOV
		}

		// Process LOV (Static list or Dynamic SQL)
		lovItems := []LOVItem{}
		addLov := func(item LOVItem) {
			for _, existing := range lovItems {
				if existing.Value == item.Value {
					return
				}
			}
			lovItems = append(lovItems, item)
		}

		// 1. Check Global LOVs in Datagrid Config
		if globalLov, ok := cat.Datagrid.LOVs[col.Name]; ok {
			for _, item := range globalLov {
				addLov(processLovItem(item, lang))
			}
		}

		// 2. Add Inline/Override LOV (Dynamic SQL or Static List)
		lovSource := col.LOV
		if overrideLov != nil {
			lovSource = overrideLov
		}

		switch v := lovSource.(type) {
		case string: // Reference to global LOV or Dynamic SQL
			if globalDef, ok := cat.Datagrid.LOVs[v]; ok {
				for _, item := range globalDef {
					addLov(processLovItem(item, lang))
				}
			} else if strings.Contains(strings.ToUpper(v), "SELECT") {
				query := strings.ReplaceAll(v, "{lang}", lang)
				rows, err := db.Query("SELECT datagrid.datagrid_execute_json($1, '{}'::jsonb)", query)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var rowJSON string
						if err := rows.Scan(&rowJSON); err == nil {
							var ri map[string]interface{}
							if err := json.Unmarshal([]byte(rowJSON), &ri); err == nil {
								val := ri["code"]
								if val == nil {
									val = ri["value"]
								}
								lbl := ri["label"]
								if lbl == nil {
									lbl = ri["display"]
								}
								if lbl == nil {
									lbl = val
								}
								if val != nil {
									addLov(LOVItem{Value: val, Label: fmt.Sprintf("%v", lbl)})
								}
							}
						}
					}
				}
			}
		case []any: // Static List
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {

					li := LOVItem{Value: m["value"]}
					if labels, ok := m["labels"].(map[string]any); ok {
						li.Labels = make(map[string]string)
						for k, v := range labels {
							if s, ok := v.(string); ok {
								li.Labels[k] = s
							}
						}
					} else if labels, ok := m["label"].(map[string]any); ok { // Handle "label" as object

						li.Labels = make(map[string]string)
						for k, v := range labels {
							if s, ok := v.(string); ok {
								li.Labels[k] = s
							}
						}
					}
					if s, ok := m["label"].(string); ok {
						li.Label = s
					} else if m["label"] != nil && li.Label == "" {
						li.Label = fmt.Sprintf("%v", m["label"])
					}

					if s, ok := m["display"].(string); ok {
						li.Display = s
					}
					if s, ok := m["rowStyle"].(string); ok {
						li.RowStyle = s
					}
					if s, ok := m["rowClass"].(string); ok {
						li.RowClass = s
					}

					addLov(processLovItem(li, lang))
				}
			}
		}

		var cssClass string
		var displayPattern string
		var linkPattern string
		if override, ok := cat.Datagrid.Columns[col.Name]; ok {
			cssClass = override.CSS
			displayPattern = override.Display
			linkPattern = override.Link
		}

		// Hardcode Elimination: Automatically add generic classes based on metadata
		nameLower := strings.ToLower(col.Name)
		if isNumericType(col.Type) {
			if cssClass != "" {
				cssClass = "col-number " + cssClass
			} else {
				cssClass = "col-number"
			}
		}
		if col.PrimaryKey || nameLower == "id" || nameLower == "sid" {
			if cssClass != "" {
				cssClass = "col-id " + cssClass
			} else {
				cssClass = "col-id"
			}
		}

		uiCols = append(uiCols, UIColumn{
			Field:    col.Name,
			Label:    label,
			CSS:      cssClass,
			Display:  displayPattern,
			Sortable: true,
			Visible:  visible,
			Record:   true,
			Type:     strings.ToLower(col.Type),
			Icon:     icon,
			Link:     linkPattern,
			LOV:      lovItems,
		})

	}

	iconStyle := strings.TrimSpace(cat.Datagrid.IconStyleLibrary)
	if iconStyle == "" {
		iconStyle = "FontAwesome"
	}

	h := &Handler{
		DB:               db,
		TableName:        obj.Name,
		Columns:          uiCols,
		Config:           cat.Datagrid,
		Catalog:          cat,
		Lang:             lang,
		IconStyleLibrary: iconStyle,
	}

	// Query mode: resolve parameters from catalog
	if strings.ToLower(cat.Type) == "query" && len(cat.Parameters) > 0 {
		h.IsQueryMode = true
		h.QuerySQL = cat.SQL

		today := time.Now().Format("2006-01-02")
		params := make([]QueryParam, len(cat.Parameters))
		copy(params, cat.Parameters)

		for i := range params {
			// Detect array types (e.g. TEXT[], INTEGER[])
			if strings.HasSuffix(strings.ToUpper(params[i].Type), "[]") {
				params[i].IsArray = true
				params[i].Type = strings.TrimSuffix(params[i].Type, "[]")
			}

			// Resolve select options
			if itype := params[i].InputType(); itype == "select" {
				opts := params[i].resolvedSelectOptions()
				for _, o := range strings.Split(opts, ",") {
					o = strings.TrimSpace(o)
					if o != "" {
						params[i].Options = append(params[i].Options, LOVItem{Value: o, Label: o})
					}
				}
			}

			// Resolve LOV/tree/grouped options from DB
			itype := params[i].InputType()
			lovSQL := params[i].resolvedLOVQuery()
			if lovSQL != "" && (itype == "lov" || itype == "lov-multi" || itype == "lov-tree" || itype == "lov-grouped") {
				rows, err := db.Query(lovSQL)
				if err == nil {
					defer rows.Close()
					cols, _ := rows.Columns()
					nCols := len(cols)
					for rows.Next() {
						switch {
						case itype == "lov-tree" && nCols >= 3:
							var val, label string
							var depth int
							if err := rows.Scan(&val, &label, &depth); err == nil {
								params[i].Options = append(params[i].Options, LOVItem{Value: val, Label: label, Depth: depth})
							}
						case itype == "lov-grouped" && nCols >= 3:
							var group, val, label string
							if err := rows.Scan(&group, &val, &label); err == nil {
								params[i].Options = append(params[i].Options, LOVItem{Value: val, Label: label, Group: group})
							}
						case nCols >= 2:
							var val, label string
							if err := rows.Scan(&val, &label); err == nil {
								params[i].Options = append(params[i].Options, LOVItem{Value: val, Label: label})
							}
						default:
							var val string
							if err := rows.Scan(&val); err == nil {
								params[i].Options = append(params[i].Options, LOVItem{Value: val, Label: val})
							}
						}
					}
				} else {
					slog.Error("LOV query error for param", "name", params[i].Name, "error", err)
				}
			}

			// Resolve default dates
			def := params[i].Default
			if def == "CURRENT_DATE" || def == "CURRENT_TIMESTAMP" {
				params[i].ResolvedDefault = today
			} else if def != "" && def != "NULL" && def != "Session user" {
				params[i].ResolvedDefault = def
			}
		}
		h.QueryParams = params
	}

	return h, nil
}

// ServeHTTP handles the grid lifecycle: metadata, filters, data, and templates.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := h.ParseParams(r)
	
	result, err := h.Execute(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result.Lang = h.Lang
	result.CurrentLang = h.Lang
	result.ListEndpoint = h.ListEndpoint

	funcs := TemplateFuncs()
	tmpl, err := template.New("datagrid").Funcs(funcs).ParseFS(UIAssets, 
		"ui/templates/partials/datagrid/*.html",
	)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "datagrid_full", result); err != nil {
		slog.Error("datagrid render error", "error", err)
	}
}

// Execute performs the SQL query and returns the results using datagrid_sql.go logic
func (h *Handler) Execute(params RequestParams) (*TableResult, error) {
	return h.FetchData(params)
}

func processLovItem(item LOVItem, lang string) LOVItem {
	li := LOVItem{
		Value:    item.Value,
		Display:  item.Display,
		Labels:   item.Labels,
		Label:    item.Label,
		RowStyle: item.RowStyle,
		RowClass: item.RowClass,
	}
	if item.Labels != nil {
		if l, ok := item.Labels[lang]; ok {
			li.Label = l
		} else if l, ok := item.Labels["en"]; ok {
			li.Label = l
		}
	}
	return li
}
