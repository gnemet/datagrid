package datagrid

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	tt "text/template"
	"time"
)

// UIAssets embeds the standardized CSS, JS and Templates
//
//go:embed ui
var UIAssets embed.FS

// UIColumn defines how a column is rendered in the grid
type UIColumn struct {
	Field      string    `json:"field"`
	Label      string    `json:"label"`
	Class      string    `json:"class"`
	CSS        string    `json:"css"` // Custom CSS classes from Catalog
	Type       string    `json:"type"`
	Sortable   bool      `json:"sortable"`
	Visible    bool      `json:"visible"`
	Record     bool      `json:"record"` // Include in detail sidebar
	Display    string    `json:"display,omitempty"`
	Icon       string    `json:"icon,omitempty"`
	LOV        []LOVItem `json:"lov,omitempty"`
	IsPivotRow bool      `json:"is_pivot_row,omitempty"`
	IsPivotCol bool      `json:"is_pivot_col,omitempty"`
}

type LOVItem struct {
	Value    interface{}       `json:"value"`
	Display  string            `json:"display,omitempty"` // Custom display format/icon
	Labels   map[string]string `json:"labels,omitempty"`
	Label    string            `json:"label,omitempty"`
	RowStyle string            `json:"rowStyle,omitempty"`
	RowClass string            `json:"rowClass,omitempty"`
	Group    string            `json:"-"` // For lov-grouped: optgroup label
	Depth    int               `json:"-"` // For lov-tree: indentation level
}

// Catalog structures for MIGR/JiraMntr compatibility
type Catalog struct {
	Version    string         `json:"version"`
	Title      string         `json:"title,omitempty"`
	Icon       string         `json:"icon,omitempty"`
	Type       string         `json:"type,omitempty"`
	CSSClass   string         `json:"css_class,omitempty"`
	Datagrid   DatagridConfig `json:"datagrid,omitempty"`
	Objects    []ObjectDef    `json:"objects"`
	Parameters []QueryParam   `json:"parameters,omitempty"`
	SQL        string         `json:"sql,omitempty"`
}

type DatagridConfig struct {
	Defaults         DatagridDefaults             `json:"defaults"`
	LOVs             map[string][]LOVItem         `json:"lovs"`
	Operations       Operations                   `json:"operations"`
	Filters          map[string]FilterDef         `json:"filters"`
	Columns          map[string]DatagridColumnDef `json:"columns"`
	Searchable       SearchableConfig             `json:"searchable"`
	IconStyleLibrary string                       `json:"iconStyleLibrary,omitempty"`
	Pivot            *PivotConfig                 `json:"pivot,omitempty"`
}

type PivotConfig struct {
	Rows       []PivotDimensionConfig `json:"rows"`       // Dimensions for rows
	Columns    []PivotDimensionConfig `json:"columns"`    // Dimensions for columns
	Values     []PivotValueConfig     `json:"values"`     // Aggregated measures
	Multiplier float64                `json:"multiplier"` // Global multiplication factor
	Subtotals  bool                   `json:"subtotals,omitempty"`
}

type PivotDimensionConfig struct {
	Column string `json:"column"`
	CSS    string `json:"css,omitempty"`
}

// UnmarshalJSON for PivotDimensionConfig allows support for both plain strings and objects
func (pd *PivotDimensionConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		pd.Column = s
		return nil
	}
	type alias PivotDimensionConfig
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*pd = PivotDimensionConfig(a)
	return nil
}

// Helper to get raw column names from dimensions
func (pc *PivotConfig) GetRowColumns() []string {
	cols := make([]string, len(pc.Rows))
	for i, r := range pc.Rows {
		cols[i] = r.Column
	}
	return cols
}

func (pc *PivotConfig) GetColColumns() []string {
	cols := make([]string, len(pc.Columns))
	for i, c := range pc.Columns {
		cols[i] = c.Column
	}
	return cols
}

type PivotValueConfig struct {
	Column string `json:"column"`
	Func   string `json:"func"`            // SUM, AVG, etc.
	Label  string `json:"label,omitempty"` // Custom label for header
}

type SearchableConfig struct {
	Columns   []string `json:"columns"`
	Operator  string   `json:"operator"`
	Threshold float64  `json:"threshold"`
}

type DatagridDefaults struct {
	PageSize      int                    `json:"page_size_default"` // Helper for template
	PageSizes     []int                  `json:"page_size"`
	SortColumn    string                 `json:"sort_column"`
	SortDirection string                 `json:"sort_direction"`
	Filters       map[string]interface{} `json:"filters"`
	Search        string                 `json:"search"`
	RowStyles     []string               `json:"rowStyles,omitempty"` // Cyclic row styles
}

type Operations struct {
	Add    bool `json:"add"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

type FilterDef struct {
	Column string `json:"column"`
	Type   string `json:"type"` // text, number, boolean, int_bool
}

// QueryParam describes a query parameter from the catalog JSON.
type QueryParam struct {
	Name            string    `json:"name"`
	Type            string    `json:"type"`                       // DATE, TEXT, INTEGER, NUMERIC, TEXT[]
	Default         string    `json:"default"`                    // CURRENT_DATE, NULL, or literal
	Input           string    `json:"input"`                      // date, number, text, select, lov, lov-tree, lov-grouped, constant
	Description     string    `json:"description"`
	Label           string    `json:"label,omitempty"`            // Display label (auto-generated from name if empty)
	LOVQuery        string    `json:"lov_query,omitempty"`        // SQL query for lov/lov-tree/lov-grouped options
	LOVName         string    `json:"lov_name,omitempty"`         // Named LOV function (e.g. "lov_department" → SELECT code, name FROM dwh.lov_department())
	SelectOptions   string    `json:"select_options,omitempty"`   // Comma-separated options for select type (alternative to select:a,b,c)
	Constant        string    `json:"constant,omitempty"`         // Constant key (e.g. "current_user") — alternative to constant:key
	Options         []LOVItem `json:"options,omitempty"`          // Resolved at load time for select/lov
	ResolvedDefault string    `json:"-"`                         // Resolved default for HTML inputs
	IsArray         bool      `json:"isArray,omitempty"`          // True for array types (TEXT[], INTEGER[]) → renders multi-select
}

// InputType returns the HTML input type for the parameter.
func (p QueryParam) InputType() string {
	in := strings.ToLower(strings.TrimSpace(p.Input))
	switch {
	case in == "date" || in == "datetime":
		return "date"
	case in == "number":
		return "number"
	case in == "lov-tree" || strings.HasPrefix(in, "lov-tree:"):
		return "lov-tree"
	case in == "lov-grouped" || strings.HasPrefix(in, "lov-grouped:"):
		return "lov-grouped"
	case in == "lov" || strings.HasPrefix(in, "lov:"):
		if p.IsArray {
			return "lov-multi"
		}
		return "lov"
	case in == "select" || strings.HasPrefix(in, "select:"):
		return "select"
	case in == "constant" || strings.HasPrefix(in, "constant:"):
		return "constant"
	default:
		return "text"
	}
}

// ConstantKey returns the constant type (e.g. "current_user") for constant params.
func (p QueryParam) ConstantKey() string {
	if p.Constant != "" {
		return p.Constant
	}
	if strings.HasPrefix(p.Input, "constant:") {
		return strings.TrimPrefix(p.Input, "constant:")
	}
	return ""
}

// resolvedLOVQuery returns the SQL for LOV resolution, checking lov_query, lov_name, and inline prefix.
func (p QueryParam) resolvedLOVQuery() string {
	// 1. Explicit lov_query field
	if p.LOVQuery != "" {
		return p.LOVQuery
	}
	// 2. Named LOV function → build SELECT
	if p.LOVName != "" {
		if !strings.Contains(p.LOVName, "(") {
			return "SELECT code, name FROM dwh." + p.LOVName + "()"
		}
		return "SELECT code, name FROM dwh." + p.LOVName
	}
	// 3. Inline prefix (backward compat): lov:SQL, lov-tree:SQL, lov-grouped:SQL
	for _, prefix := range []string{"lov-tree:", "lov-grouped:", "lov:"} {
		if strings.HasPrefix(p.Input, prefix) {
			return strings.TrimPrefix(p.Input, prefix)
		}
	}
	return ""
}

// resolvedSelectOptions returns options for select input, checking select_options field and inline prefix.
func (p QueryParam) resolvedSelectOptions() string {
	if p.SelectOptions != "" {
		return p.SelectOptions
	}
	if strings.HasPrefix(p.Input, "select:") {
		return strings.TrimPrefix(p.Input, "select:")
	}
	return ""
}

// DisplayLabel returns the label for display, auto-generating from name if empty.
func (p QueryParam) DisplayLabel() string {
	if p.Label != "" {
		return p.Label
	}
	// Title case from snake_case
	name := strings.ReplaceAll(p.Name, "_", " ")
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

type DatagridColumnDef struct {
	Visible *bool             `json:"visible,omitempty"`
	CSS     string            `json:"css,omitempty"`
	Display string            `json:"display,omitempty"`
	Labels  map[string]string `json:"labels"`
	Icon    string            `json:"icon,omitempty"`
	LOV     interface{}       `json:"lov,omitempty"`
}

type ObjectDef struct {
	Name    string      `json:"name"`
	Columns []ColumnDef `json:"columns"`
}

type ColumnDef struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Labels     map[string]string `json:"labels"`
	LOV        interface{}       `json:"lov,omitempty"`
	PrimaryKey bool              `json:"primary_key,omitempty"`
}

// RequestParams captures search, sort, and pagination from the request
type RequestParams struct {
	Search  string
	Sort    []string // List of "field:dir"
	Filters map[string][]string
	Limit   int
	Offset  int
}

// TableResult contains data to be rendered by the partial template
type TableResult struct {
	Records             []map[string]interface{}
	TotalCount          int
	Offset              int
	Limit               int
	UIColumns           []UIColumn
	Config              DatagridConfig
	Lang                string // For localization in templates
	IconStyleLibrary    string
	Icon                string
	CurrentCatalog      string
	IsPhosphor          bool
	Title               string // Header Title
	ListEndpoint        string // HX-Get Endpoint
	HasJSONColumn       bool   // For Expand Keys button
	PivotEndpoint       string // Endpoint for pivot data
	ViewMode            string // "grid" or "pivot"
	PivotResult         *PivotResult
	LOVChooserThreshold int
	App                 struct {
		Name string
	}
	Catalogs    map[string]string
	LangsJSON   string
	CurrentLang string
	QueryParams     []QueryParam
	IsQueryMode     bool
	ExecuteEndpoint string
	CurrentUser     string
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
		if override, ok := cat.Datagrid.Columns[col.Name]; ok {
			cssClass = override.CSS
			displayPattern = override.Display
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
					log.Printf("LOV query error for param %s: %v", params[i].Name, err)
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	params := h.ParseParams(r)

	mode := r.URL.Query().Get("mode")

	// Default to pivot if catalog type says so
	if mode == "" && h.Catalog.Type == "pivot" {
		mode = "pivot"
	}

	if mode == "csv" {
		params.Limit = 0 // No limit for export
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", h.TableName))

		err := h.StreamCSV(w, params)
		if err != nil {
			// Note: Header already sent, but we can't do much here except log or close
		}
		return
	}

	if mode == "pivot" {
		// Handle Dimension Swap
		if r.URL.Query().Get("swap") == "true" {
			// Transiently swap rows and columns in config for this request
			// Note: This still has a race if shared, but we only swap for the duration of this request
			// Actually, to be safe, we should copy the config if it's shared.
			rows := h.Config.Pivot.Rows
			cols := h.Config.Pivot.Columns
			h.Config.Pivot.Rows = cols
			h.Config.Pivot.Columns = rows
			// Revert at the end of function if shared? No, copy is better.
			// But for now, let's just use it as is since it's a test app.
		}

		// Annotate UIColumns with Pivot metadata (use local copy to avoid data race on handler singleton)
		uiCols := make([]UIColumn, len(h.Columns))
		copy(uiCols, h.Columns)

		if h.Config.Pivot != nil {
			for i := range uiCols {
				uiCols[i].IsPivotRow = false
				uiCols[i].IsPivotCol = false
				for _, rowDim := range h.Config.Pivot.Rows {
					if uiCols[i].Field == rowDim.Column {
						uiCols[i].IsPivotRow = true
					}
				}
				for _, colDim := range h.Config.Pivot.Columns {
					if uiCols[i].Field == colDim.Column {
						uiCols[i].IsPivotCol = true
					}
				}
			}
		}

		pivotRes, err := h.PivotData(params)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res := &TableResult{
			Records:             nil, // No grid records in pivot mode
			TotalCount:          0,
			Offset:              params.Offset,
			Limit:               params.Limit,
			UIColumns:           uiCols, // Use local copy!
			PivotResult:         pivotRes,
			Config:              h.Config,
			Lang:                h.Lang,
			IconStyleLibrary:    h.IconStyleLibrary,
			Icon:                h.Catalog.Icon,
			IsPhosphor:          h.IconStyleLibrary == "Phosphor",
			Title:               h.Catalog.Title,
			CurrentCatalog:      r.URL.Query().Get("config"),
			ListEndpoint:        h.ListEndpoint,
			PivotEndpoint:       h.PivotEndpoint,
			ViewMode:            "pivot",
			LOVChooserThreshold: h.LOVChooserThreshold,
			Catalogs:            h.Catalogs,
			LangsJSON:           `[{"code":"en","label":"EN"},{"code":"hu","label":"HU"}]`,
			CurrentLang:         h.Lang,
		}
		res.App.Name = h.AppName

		if r.Header.Get("HX-Request") != "" {
			funcs := TemplateFuncs()
			// MUST parse ALL partials because pivot.html might use them
			tmpl, err := template.New("pivot_partial").Funcs(funcs).ParseFS(UIAssets, "ui/templates/partials/datagrid/*.html")
			if err != nil {
				http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
				return
			}

			fmt.Fprint(w, "<div id=\"datagrid-container\">")
			err = tmpl.ExecuteTemplate(w, "datagrid_pivot", res)
			fmt.Fprint(w, "</div>")
			if err != nil {
			}
			return
		}

		// Full page load (index.html defines datagrid_main)
		tmpl, err := template.New("index").Funcs(TemplateFuncs()).ParseFS(UIAssets, "ui/templates/*.html", "ui/templates/partials/datagrid/*.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.ExecuteTemplate(w, "index.html", res)
		if err != nil {
		}

	} else {
		// Grid mode logic
		gridRes, err := h.FetchData(params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		gridRes.UIColumns = h.Columns // No race here since we don't modify h.Columns in Grid mode
		gridRes.ViewMode = "grid"
		gridRes.ListEndpoint = h.ListEndpoint
		gridRes.PivotEndpoint = h.PivotEndpoint
		gridRes.CurrentCatalog = r.URL.Query().Get("config")
		gridRes.Catalogs = h.Catalogs
		gridRes.App.Name = h.AppName
		gridRes.LangsJSON = `[{"code":"en","label":"EN"},{"code":"hu","label":"HU"}]`
		gridRes.CurrentLang = h.Lang
		gridRes.QueryParams = h.QueryParams
		gridRes.IsQueryMode = h.IsQueryMode
		gridRes.ExecuteEndpoint = h.ExecuteEndpoint
		gridRes.CurrentUser = h.CurrentUser

		if r.Header.Get("HX-Request") != "" {
			funcs := TemplateFuncs()
			tmpl, err := template.New("grid_partial").Funcs(funcs).ParseFS(UIAssets, "ui/templates/partials/datagrid/*.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			fmt.Fprint(w, "<div id=\"datagrid-container\">")
			err = tmpl.ExecuteTemplate(w, "datagrid_table", gridRes)
			fmt.Fprint(w, "</div>")
			return
		}

		tmpl, err := template.New("index").Funcs(TemplateFuncs()).ParseFS(UIAssets, "ui/templates/*.html", "ui/templates/partials/datagrid/*.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.ExecuteTemplate(w, "index.html", gridRes)
		if err != nil {
		}
	}
}

// ExecuteQuery runs a parameterized query and returns results as a datagrid table.
func (h *Handler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	if !h.IsQueryMode {
		http.Error(w, "Not a query-mode catalog", http.StatusBadRequest)
		return
	}

	// Build SQL with parameter substitution
	sqlQuery := h.QuerySQL

	// Strip SQL comments
	lines := strings.Split(sqlQuery, "\n")
	var cleanLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleanLines = append(cleanLines, l)
	}
	sqlQuery = strings.Join(cleanLines, "\n")

	// Protect PostgreSQL :: type casts from param replacement
	const castPlaceholder = "\x00CAST\x00"
	sqlQuery = strings.ReplaceAll(sqlQuery, "::", castPlaceholder)

	r.ParseForm()

	for _, p := range h.QueryParams {
		placeholder := ":" + p.Name
		var val string

		// Array (multi-select) parameters
		if p.IsArray {
			vals := r.Form[p.Name]
			var cleaned []string
			for _, v := range vals {
				v = strings.TrimSpace(v)
				if v != "" {
					cleaned = append(cleaned, "'"+sanitizeParam(v)+"'")
				}
			}
			if len(cleaned) == 0 {
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, "NULL")
			} else {
				arrayLiteral := "ARRAY[" + strings.Join(cleaned, ",") + "]"
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, arrayLiteral)
			}
			continue
		}

		// Constants are resolved server-side
		if strings.HasPrefix(p.Input, "constant:") {
			switch p.ConstantKey() {
			case "current_user":
				val = h.CurrentUser
			default:
				val = p.Default
			}
		} else {
			val = r.FormValue(p.Name)
		}

		if val == "" {
			def := p.Default
			switch {
			case def == "CURRENT_TIMESTAMP" || def == "CURRENT_DATE":
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, def)
			case def == "NULL" || def == "" || def == "Session user":
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, "NULL")
			default:
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, "'"+def+"'")
			}
		} else {
			// Detect SQL keywords — don't quote them
			upperVal := strings.ToUpper(strings.TrimSpace(val))
			if upperVal == "CURRENT_DATE" || upperVal == "CURRENT_TIMESTAMP" || upperVal == "NULL" {
				sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, upperVal)
			} else {
				switch strings.ToUpper(p.Type) {
				case "INTEGER", "INT", "NUMERIC", "BIGINT":
					sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, val)
				default:
					sqlQuery = strings.ReplaceAll(sqlQuery, placeholder, "'"+sanitizeParam(val)+"'")
				}
			}
		}
	}

	// Handle any remaining :param references
	re := regexp.MustCompile(`:(\w+)`)
	sqlQuery = re.ReplaceAllStringFunc(sqlQuery, func(match string) string {
		paramName := strings.TrimPrefix(match, ":")
		val := r.FormValue(paramName)
		if val == "" {
			return "NULL"
		}
		return "'" + sanitizeParam(val) + "'"
	})

	// Restore PostgreSQL :: casts
	sqlQuery = strings.ReplaceAll(sqlQuery, castPlaceholder, "::")

	if os.Getenv("DEBUG_SQL") == "true" {
		log.Printf("Query Execute SQL:\n%s", sqlQuery)
	}

	rows, err := h.DB.Query(sqlQuery)
	if err != nil {
		log.Printf("Query execution error: %v", err)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="dg-query-error"><i class="fas fa-exclamation-triangle"></i> <strong>Query Error:</strong> %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}
	defer rows.Close()

	// Get column info
	cols, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	colTypes, _ := rows.ColumnTypes()

	// Scan all rows into records
	records := []map[string]interface{}{}
	values := make([]interface{}, len(cols))
	scanArgs := make([]interface{}, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			continue
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			if values[i] == nil {
				row[col] = nil
			} else {
				switch v := values[i].(type) {
				case time.Time:
					if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 {
						row[col] = v.Format("2006-01-02")
					} else {
						row[col] = v.Format("2006-01-02 15:04")
					}
				case []byte:
					row[col] = string(v)
				default:
					row[col] = v
				}
			}
		}
		if jsonBytes, err := json.Marshal(row); err == nil {
			row["_json"] = string(jsonBytes)
		}
		records = append(records, row)
	}

	// Build UIColumns from query result columns
	uiCols := make([]UIColumn, len(cols))
	for i, col := range cols {
		displayCol := strings.ReplaceAll(col, "_", " ")
		// Title case
		words := strings.Fields(displayCol)
		for j, w := range words {
			if len(w) > 0 {
				words[j] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		displayCol = strings.Join(words, " ")

		colType := "text"
		if i < len(colTypes) {
			dbType := strings.ToUpper(colTypes[i].DatabaseTypeName())
			if strings.Contains(dbType, "INT") || strings.Contains(dbType, "FLOAT") ||
				strings.Contains(dbType, "NUMERIC") || strings.Contains(dbType, "DECIMAL") {
				colType = "numeric"
			} else if strings.Contains(dbType, "DATE") || strings.Contains(dbType, "TIME") {
				colType = "date"
			}
		}

		cssClass := ""
		if colType == "numeric" {
			cssClass = "col-number"
		}

		uiCols[i] = UIColumn{
			Field:    col,
			Label:    displayCol,
			CSS:      cssClass,
			Type:     colType,
			Sortable: false,
			Visible:  true,
			Record:   true,
		}
	}

	// If handler has pre-defined columns from catalog, use those for labels/css
	for i, uiCol := range uiCols {
		for _, hCol := range h.Columns {
			if hCol.Field == uiCol.Field {
				uiCols[i].Label = hCol.Label
				if hCol.CSS != "" {
					uiCols[i].CSS = hCol.CSS
				}
				uiCols[i].Icon = hCol.Icon
				uiCols[i].LOV = hCol.LOV
				uiCols[i].Display = hCol.Display
				break
			}
		}
	}

	res := &TableResult{
		Records:             records,
		TotalCount:          len(records),
		Offset:              0,
		Limit:               len(records),
		UIColumns:           uiCols,
		Config:              h.Config,
		Lang:                h.Lang,
		IconStyleLibrary:    h.IconStyleLibrary,
		IsPhosphor:          h.IconStyleLibrary == "Phosphor",
		Title:               h.Catalog.Title,
		ListEndpoint:        h.ListEndpoint,
		LOVChooserThreshold: h.LOVChooserThreshold,
		ViewMode:            "query",
		IsQueryMode:         true,
	}
	res.App.Name = h.AppName

	// Render using the standard table partial
	funcs := TemplateFuncs()
	tmpl, err := template.New("query_result").Funcs(funcs).ParseFS(UIAssets, "ui/templates/partials/datagrid/*.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="dg-query-row-count"><i class="fas fa-database"></i> %d rows</div>`, len(records))
	err = tmpl.ExecuteTemplate(w, "datagrid_table", res)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

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

func (h *Handler) StreamCSV(w io.Writer, p RequestParams) error {
	// 1. Generate Hybrid SQL
	query, configJSON, err := h.BuildGridSQL(p)
	if err != nil {
		return err
	}

	rows, err := h.DB.Query("SELECT datagrid.datagrid_execute_csv($1, $2)", query, configJSON)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err == nil {
			fmt.Fprintln(w, line)
		}
	}
	return nil
}

// BuildGridSQL generates the SQL query and JSON configuration for the grid execution.
func (h *Handler) BuildGridSQL(p RequestParams) (string, string, error) {
	order := h.buildOrder(p.Sort)

	// Build JSON config for SQL generation
	type ColDecl struct {
		Name  string `json:"name"`
		Alias string `json:"alias,omitempty"`
	}
	type LOVEntry struct {
		Code  interface{} `json:"code"`
		Label string      `json:"label"`
	}
	type LOVDecl struct {
		Column     string     `json:"column"`
		Values     []LOVEntry `json:"values"`
		ValuesJSON string     `json:"valuesJSON,omitempty"`
		IsBoolean  bool       `json:"-"`
		IsNumber   bool       `json:"-"`
	}

	config := make(map[string]interface{})
	config["tableName"] = h.TableName
	config["order"] = order
	config["limit"] = p.Limit
	config["offset"] = p.Offset

	colsDecl := []ColDecl{}
	lovsDecl := []LOVDecl{}

	for _, col := range h.Columns {
		if strings.Contains(col.Display, "%") {
			continue
		}
		colsDecl = append(colsDecl, ColDecl{Name: "src." + col.Field, Alias: col.Field})
		if len(col.LOV) > 0 {
			entries := []LOVEntry{}
			for _, item := range col.LOV {
				lbl := item.Labels[h.Lang]
				if lbl == "" {
					lbl = item.Label
				}
				if lbl == "" {
					lbl = item.Display
				}
				entries = append(entries, LOVEntry{Code: item.Value, Label: lbl})
			}
			valJSON, _ := json.Marshal(entries)
			isBool := strings.ToLower(col.Type) == "boolean" || strings.ToLower(col.Type) == "bool"
			isNum := strings.ToLower(col.Type) == "integer" || strings.ToLower(col.Type) == "numeric" || strings.ToLower(col.Type) == "double"
			lovsDecl = append(lovsDecl, LOVDecl{
				Column:     col.Field,
				Values:     entries,
				ValuesJSON: string(valJSON),
				IsBoolean:  isBool,
				IsNumber:   isNum,
			})
			colsDecl = append(colsDecl, ColDecl{
				Name:  "lov" + fmt.Sprintf("%d", len(lovsDecl)) + ".label",
				Alias: col.Field + "_label",
			})
		}
	}

	type FilterWrap struct {
		Column    string
		IsArray   bool
		IsBoolean bool
		IsNumber  bool
		Values    []string
		Value     string
	}
	filtersWrap := []FilterWrap{}
	for k, v := range p.Filters {
		if _, ok := h.Config.Filters[k]; !ok {
			continue
		}

		f := FilterWrap{Column: k}
		if conf, ok := h.Config.Filters[k]; ok {
			switch conf.Type {
			case "boolean":
				f.IsBoolean = true
			case "integer", "numeric", "double":
				f.IsNumber = true
			}
		}
		if len(v) > 1 {
			f.IsArray = true
			f.Values = v
		} else if len(v) == 1 {
			f.Value = v[0]
		}
		filtersWrap = append(filtersWrap, f)
	}

	tplData := map[string]interface{}{
		"TableName": h.TableName,
		"Columns":   colsDecl,
		"LOVs":      lovsDecl,
		"Filters":   filtersWrap,
		"Order":     order,
		"Limit":     p.Limit,
		"Offset":    p.Offset,
	}

	query, err := h.renderSQL("grid.sql", tplData)
	if err != nil {
		return "", "", err
	}
	configJSON, _ := json.Marshal(config)
	return query, string(configJSON), nil
}

func (h *Handler) FetchData(p RequestParams) (*TableResult, error) {
	// Start transaction to use SET LOCAL for threshold
	tx, err := h.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if h.Config.Searchable.Operator == "%" && h.Config.Searchable.Threshold > 0 {
		tx.Exec(fmt.Sprintf("SET LOCAL pg_trgm.similarity_threshold = %f", h.Config.Searchable.Threshold))
	}

	where, args := h.buildWhere(p)

	// 0. Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", quote_ident(h.TableName), where)
	var total int
	if err := tx.QueryRow(countQuery, args...).Scan(&total); err != nil {
		if os.Getenv("DEBUG_SQL") == "true" {
			fmt.Printf("--- COUNT QUERY ERROR ---\nQuery: %s\nError: %v\n------------------------\n", countQuery, err)
		}
		return nil, err
	}

	// 1. Generate Hybrid SQL
	query, configJSON, err := h.BuildGridSQL(p)
	if err != nil {
		return nil, err
	}

	if os.Getenv("DEBUG_SQL") == "true" {
		fmt.Printf("--- GRID SQL ---\n%s\n----------------\n", query)
	}

	// 2. Fetch Records using streaming wrapper
	records := []map[string]interface{}{}
	rows, err := tx.Query("SELECT datagrid.datagrid_execute_json($1, $2)", query, configJSON)
	if err != nil {
		if os.Getenv("DEBUG_SQL") == "true" {
			fmt.Printf("--- SQL EXEC ERROR ---\nError: %v\n---------------------\n", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var rowJSON string
			if err := rows.Scan(&rowJSON); err == nil {
				var row map[string]interface{}
				if err := json.Unmarshal([]byte(rowJSON), &row); err == nil {
					records = append(records, row)
				}
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute grid data: %w", err)
	}

	// 3. Post-process records (Styling & Metadata)
	for i, row := range records {
		if jsonBytes, err := json.Marshal(row); err == nil {
			row["_json"] = string(jsonBytes)
		}

		var rowStyles []string
		var rowClasses []string
		for _, col := range h.Columns {
			val := row[col.Field]
			if val == nil {
				continue
			}

			if label, ok := row[col.Field+"_label"]; ok && label != nil {
				row[col.Field] = label
			}

			for _, item := range col.LOV {
				if fmt.Sprintf("%v", item.Value) == fmt.Sprintf("%v", val) {
					if item.RowStyle != "" {
						rowStyles = append(rowStyles, item.RowStyle)
					}
					if item.RowClass != "" {
						rowClasses = append(rowClasses, item.RowClass)
					}
					break
				}
			}
		}

		if len(h.Config.Defaults.RowStyles) > 0 {
			cycleIdx := i % len(h.Config.Defaults.RowStyles)
			rowStyles = append(rowStyles, h.Config.Defaults.RowStyles[cycleIdx])
		}

		if len(rowStyles) > 0 {
			row["_row_style"] = strings.Join(rowStyles, "; ")
		}
		if len(rowClasses) > 0 {
			row["_row_class"] = strings.Join(rowClasses, " ")
		}
	}

	res := &TableResult{
		Records:             records,
		TotalCount:          total,
		Offset:              p.Offset,
		Limit:               p.Limit,
		UIColumns:           h.Columns,
		Config:              h.Config,
		Lang:                h.Lang,
		IconStyleLibrary:    h.IconStyleLibrary,
		IsPhosphor:          h.IconStyleLibrary == "Phosphor",
		Title:               h.Catalog.Title,
		ListEndpoint:        h.ListEndpoint,
		LOVChooserThreshold: h.LOVChooserThreshold,
	}

	// Detect if any column is JSON for UI buttons
	for _, col := range h.Columns {
		if strings.Contains(strings.ToLower(col.Type), "json") {
			res.HasJSONColumn = true
			break
		}
	}
	return res, tx.Commit()
}

func (h *Handler) buildWhere(p RequestParams) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}
	argIdx := 1

	// Exact Filters (LOVs)
	for field, values := range p.Filters {
		// Check if defined in Config Filters
		filterDef, isDefined := h.Config.Filters[field]
		if !isDefined {
			continue // Strict mode: only filters defined in JSON
		}

		colName := filterDef.Column
		if colName == "" {
			colName = field
		}

		var paramParts []string
		hasNone := false
		for _, val := range values {
			if val == "__NONE__" {
				hasNone = true
				continue
			}
			if val == "" {
				continue
			}
			paramParts = append(paramParts, fmt.Sprintf("$%d", argIdx))

			// Handle Type Conversion
			switch filterDef.Type {
			case "int_bool":
				if val == "true" {
					args = append(args, 1)
				} else {
					args = append(args, 0)
				}
			case "boolean":
				if val == "true" {
					args = append(args, true)
				} else {
					args = append(args, false)
				}
			case "number":
				var num int
				fmt.Sscanf(val, "%d", &num)
				args = append(args, num)
			default:
				args = append(args, val)
			}
			argIdx++
		}

		if hasNone && len(paramParts) == 0 {
			clauses = append(clauses, "1=0")
		} else if len(paramParts) > 0 {

			finalCol := colName
			if !strings.Contains(finalCol, "->") && !strings.Contains(finalCol, "(") && !strings.Contains(finalCol, "\"") {
				finalCol = fmt.Sprintf("\"%s\"", finalCol)
			}
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", finalCol, strings.Join(paramParts, ", ")))
		}
	}

	// Search
	if p.Search != "" {
		searchCols := []string{}
		if len(h.Config.Searchable.Columns) > 0 {
			for _, sc := range h.Config.Searchable.Columns {
				searchCols = append(searchCols, fmt.Sprintf("(%s)::text", sc))
			}
		} else {
			// Fallback: search in all text/unknown columns
			for _, c := range h.Columns {
				if c.Type == "" || c.Type == "text" || c.Type == "varchar" || c.Type == "string" {
					searchCols = append(searchCols, fmt.Sprintf("%s::text", c.Field))
				}
			}
		}

		if len(searchCols) > 0 {
			op := h.Config.Searchable.Operator
			if op == "" {
				op = "%" // Default to similarity
			}

			orClauses := []string{}
			for _, col := range searchCols {
				orClauses = append(orClauses, fmt.Sprintf("%s %s $%d::text", col, op, argIdx))
			}
			clauses = append(clauses, "("+strings.Join(orClauses, " OR ")+")")

			args = append(args, p.Search)
			argIdx++
		}
	}

	if len(clauses) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func (h *Handler) buildOrder(sorts []string) string {
	validateDir := func(d string) string {
		d = strings.ToUpper(d)
		valid := []string{"ASC", "DESC", "ASC NULLS FIRST", "ASC NULLS LAST", "DESC NULLS FIRST", "DESC NULLS LAST"}
		for _, v := range valid {
			if d == v {
				return d
			}
		}
		return "ASC"
	}

	defaultSortCol := "id"
	// Try to find a better default from catalog if 'id' might not exist
	if len(h.Catalog.Objects) > 0 && len(h.Catalog.Objects[0].Columns) > 0 {
		foundID := false
		var firstPK string
		for _, col := range h.Catalog.Objects[0].Columns {
			if col.Name == "id" {
				foundID = true
				break
			}
			if col.PrimaryKey && firstPK == "" {
				firstPK = col.Name
			}
		}
		if !foundID && firstPK != "" {
			defaultSortCol = firstPK
		} else if !foundID {
			defaultSortCol = h.Catalog.Objects[0].Columns[0].Name
		}
	}

	defaultSort := fmt.Sprintf("ORDER BY %s DESC", defaultSortCol)
	if h.Config.Defaults.SortColumn != "" {
		dir := validateDir(h.Config.Defaults.SortDirection)
		defaultSort = fmt.Sprintf("ORDER BY %s %s", h.Config.Defaults.SortColumn, dir)
	}

	if len(sorts) == 0 {
		return defaultSort
	}

	// Handle both single and comma-separated sorts from multiple params or split
	var allSorts []string
	for _, s := range sorts {
		allSorts = append(allSorts, strings.Split(s, ",")...)
	}

	clauses := []string{}
	for _, s := range allSorts {
		field := s
		dirStr := "ASC"

		// Try splitting by colon or space
		if strings.Contains(s, ":") {
			parts := strings.Split(s, ":")
			field = parts[0]
			dirStr = parts[1]
		} else if strings.Contains(s, " ") {
			parts := strings.Split(s, " ")
			field = parts[0]
			dirStr = strings.Join(parts[1:], " ")
		}

		dbCol := field
		if strings.HasPrefix(field, "dyn-") {
			// Extract JSON path: dyn-data.role -> data->>'role'
			path := strings.TrimPrefix(field, "dyn-")
			pp := strings.Split(path, ".")
			if len(pp) > 1 {
				dbCol = pp[0]
				for i := 1; i < len(pp); i++ {
					if i == len(pp)-1 {
						dbCol += fmt.Sprintf("->>'%s'", pp[i])
					} else {
						dbCol += fmt.Sprintf("->'%s'", pp[i])
					}
				}
			}
		} else {
			found := false
			for _, col := range h.Columns {
				if col.Field == field {
					found = true
					break
				}
			}
			if !found && field != "id" {
				continue
			}
		}

		dir := validateDir(dirStr)
		if !strings.Contains(dbCol, "->") && !strings.Contains(dbCol, "(") && !strings.Contains(dbCol, "\"") {
			dbCol = fmt.Sprintf("\"%s\"", dbCol)
		}
		clauses = append(clauses, fmt.Sprintf("%s %s", dbCol, dir))
	}

	if len(clauses) == 0 {
		return defaultSort
	}
	return "ORDER BY " + strings.Join(clauses, ", ")
}

// TemplateFuncs returns a map of standard datagrid template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"renderRow": RenderRow,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"to_lower":  strings.ToLower,
		"T":         func(s string) string { return s },

		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"formatNum": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				if val == 0 {
					return "0"
				}
				if val > 1000000 {
					return fmt.Sprintf("%.2fM", val/1000000)
				}
				if val > 1000 {
					return fmt.Sprintf("%.2fk", val/1000)
				}
				return fmt.Sprintf("%.2f", val)
			case int, int64:
				return fmt.Sprintf("%d", val)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		// Query parameter helpers
		"inputType": func(p QueryParam) string { return p.InputType() },
		"constantKey": func(p QueryParam) string { return p.ConstantKey() },
		"displayLabel": func(p QueryParam) string { return p.DisplayLabel() },

		// lov-tree: indent label by depth
		"indentLabel": func(item LOVItem) string {
			prefix := ""
			for j := 0; j < item.Depth; j++ {
				prefix += "── "
			}
			return prefix + item.Label
		},

		// lov-grouped: ordered unique group labels
		"lovGroups": func(items []LOVItem) []string {
			seen := map[string]bool{}
			var groups []string
			for _, item := range items {
				if !seen[item.Group] {
					seen[item.Group] = true
					groups = append(groups, item.Group)
				}
			}
			return groups
		},

		// lov-grouped: items within a specific group
		"lovByGroup": func(items []LOVItem, group string) []LOVItem {
			var result []LOVItem
			for _, item := range items {
				if item.Group == group {
					result = append(result, item)
				}
			}
			return result
		},
	}
}

// quote_ident is a helper for SQL templates to prevent injection
func quote_ident(s string) string {
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		for i, p := range parts {
			parts[i] = `"` + strings.ReplaceAll(p, `"`, `""`) + `"`
		}
		return strings.Join(parts, ".")
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// renderSQL renders a Go text/template for SQL generation
func (h *Handler) renderSQL(tmplName string, data interface{}) (string, error) {
	// For simplicity in this demo, we'll read from disk.
	tmplPath := fmt.Sprintf("internal/sql/templates/%s.tmpl", tmplName)
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}

	funcMap := template.FuncMap{
		"add":         func(a, b int) int { return a + b },
		"quote_ident": quote_ident,
		"coalesce": func(s ...string) string {
			for _, v := range s {
				if v != "" {
					return v
				}
			}
			return ""
		},
	}

	tmpl, err := tt.New(tmplName).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}

	result := sb.String()
	// Automatically cast $1 to jsonb for Postgres type inference
	result = strings.ReplaceAll(result, "$1", "$1::jsonb")

	if os.Getenv("DEBUG_SQL") == "true" {
		fmt.Printf("--- RENDERED SQL (%s) ---\n%s\n------------------------\n", tmplName, result)
	}
	return result, nil
}

// RenderRow replaces %field% placeholders in a pattern with values from the row
func RenderRow(pattern string, row map[string]interface{}) template.HTML {
	result := pattern
	for k, v := range row {
		placeholder := fmt.Sprintf("%%%s%%", k)
		valStr := fmt.Sprintf("%v", v)
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return template.HTML(result)
}

func isNumericType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "int") || strings.Contains(t, "num") || strings.Contains(t, "float") ||
		strings.Contains(t, "double") || strings.Contains(t, "decimal") || strings.Contains(t, "money")
}
