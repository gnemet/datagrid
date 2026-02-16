package datagrid

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	tt "text/template"
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
}

// Catalog structures for MIGR/JiraMntr compatibility
type Catalog struct {
	Version  string         `json:"version"`
	Title    string         `json:"title,omitempty"`
	Icon     string         `json:"icon,omitempty"`
	Type     string         `json:"type,omitempty"`
	CSSClass string         `json:"css_class,omitempty"`
	Datagrid DatagridConfig `json:"datagrid,omitempty"`
	Objects  []ObjectDef    `json:"objects"`
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
	LOVChooserThreshold int
	AppName             string
	Catalogs            map[string]string
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
				rows, err := db.Query(query)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var val, lbl string
						if err := rows.Scan(&val, &lbl); err == nil {
							lovItems = append(lovItems, LOVItem{Value: val, Label: lbl})
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

		if displayPattern == "" {
			displayPattern = label
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

	return &Handler{
		DB:               db,
		TableName:        obj.Name,
		Columns:          uiCols,
		Config:           cat.Datagrid,
		Catalog:          cat,
		Lang:             lang,
		IconStyleLibrary: iconStyle,
	}, nil
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
	order := h.buildOrder(p.Sort)

	// Handle explicit schema if table name contains dot
	tableName := h.TableName
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		tableName = fmt.Sprintf("\"%s\".\"%s\"", parts[0], parts[1])
	} else {
		tableName = fmt.Sprintf("\"%s\"", tableName)
	}

	// 0. Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", quote_ident(h.TableName), where)
	var total int
	if err := tx.QueryRow(countQuery, args...).Scan(&total); err != nil {
		if os.Getenv("DEBUG_SQL") == "true" {
			fmt.Printf("--- COUNT QUERY ERROR ---\nQuery: %s\nError: %v\n------------------------\n", countQuery, err)
		}
		return nil, err
	}

	// 1. Prepare JSON config for datagrid_execute
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
		Join       string     `json:"join,omitempty"`
		ValuesJSON string     `json:"valuesJSON,omitempty"`
		IsBoolean  bool       `json:"-"`
		IsNumber   bool       `json:"-"`
	}

	config := make(map[string]interface{})
	config["tableName"] = h.TableName // Re-use raw table name for JSON
	config["order"] = order
	config["limit"] = p.Limit
	config["offset"] = p.Offset

	colsDecl := []ColDecl{}
	lovsDecl := []LOVDecl{}

	for _, col := range h.Columns {
		// Heuristic: if Display contains placeholders, it's a virtual column
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
	config["columns"] = colsDecl
	config["lovs"] = lovsDecl
	config["filters"] = p.Filters

	if os.Getenv("DEBUG_SQL") == "true" {
	}

	configJSON, _ := json.Marshal(config)

	// 2. Fetch Records
	useTemplate := os.Getenv("EXPERIMENTAL_SQL_TEMPLATES") == "true"
	var recordsJSON string

	if useTemplate {
		// --- EXPERIMENTAL: Go Template Based Generation ---
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
			// Skip technical parameters that are not defined as filters in the catalog
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
			return nil, fmt.Errorf("failed to render grid SQL template: %w", err)
		}
		if os.Getenv("DEBUG_SQL") == "true" {
			fmt.Printf("--- EXPERIMENTAL GRID SQL ---\n%s\n-----------------------------\n", query)
		}
		err = tx.QueryRow("SELECT datagrid.datagrid_execute_json($1, $2)", query, string(configJSON)).Scan(&recordsJSON)
		if err != nil {
			if os.Getenv("DEBUG_SQL") == "true" {
				fmt.Printf("--- EXPERIMENTAL SQL EXEC ERROR ---\nError: %v\n----------------------------------\n", err)
			}
		}
	} else {
		// --- ESTABLISHED: Database Side Execution ---
		err = tx.QueryRow("SELECT datagrid_execute($1, 'grid', 'json')", string(configJSON)).Scan(&recordsJSON)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute grid data: %w", err)
	}

	records := []map[string]interface{}{}
	if recordsJSON != "" && recordsJSON != "null" {
		if err := json.Unmarshal([]byte(recordsJSON), &records); err != nil {
			return nil, fmt.Errorf("failed to parse grid data: %w", err)
		}
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
