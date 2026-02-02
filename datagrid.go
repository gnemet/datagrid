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
)

// UIAssets embeds the standardized CSS, JS and Templates
//
//go:embed ui/*
var UIAssets embed.FS

// UIColumn defines how a column is rendered in the grid
type UIColumn struct {
	Field    string    `json:"field"`
	Label    string    `json:"label"`
	Class    string    `json:"class"`
	CSS      string    `json:"css"` // Custom CSS classes from Catalog
	Type     string    `json:"type"`
	Sortable bool      `json:"sortable"`
	Visible  bool      `json:"visible"`
	Record   bool      `json:"record"` // Include in detail sidebar
	Display  string    `json:"display,omitempty"`
	Icon     string    `json:"icon,omitempty"`
	LOV      []LOVItem `json:"lov,omitempty"`
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
	Datagrid DatagridConfig `json:"datagrid,omitempty"`
	Objects  []ObjectDef    `json:"objects"`
}

type DatagridConfig struct {
	Defaults   DatagridDefaults             `json:"defaults"`
	LOVs       map[string][]LOVItem         `json:"lovs"`
	Operations Operations                   `json:"operations"`
	Filters    map[string]FilterDef         `json:"filters"`
	Columns    map[string]DatagridColumnDef `json:"columns"`
	Searchable SearchableConfig             `json:"searchable"`
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
	Visible bool              `json:"visible"`
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
	Name       string            `json:"name"`
	Type       string            `json:"type"`
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
	Records    []map[string]interface{}
	TotalCount int
	Offset     int
	Limit      int
	UIColumns  []UIColumn
	Config     DatagridConfig
}

// Handler handles the grid data requests
type Handler struct {
	DB        *sql.DB
	TableName string
	Columns   []UIColumn
	Config    DatagridConfig
	Catalog   Catalog
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
		fmt.Printf("DEBUG DATAGRID: Unmarshal error: %v\n", err)
		return nil, err
	}

	// Populate PageSize helper for templates
	if len(cat.Datagrid.Defaults.PageSizes) > 0 {
		cat.Datagrid.Defaults.PageSize = cat.Datagrid.Defaults.PageSizes[0]
	}

	fmt.Printf("DEBUG DATAGRID: Catalog Version: %s, Objects: %d\n", cat.Version, len(cat.Objects))
	if len(cat.Objects) == 0 {
		return nil, fmt.Errorf("no objects found in catalog")
	}

	obj := cat.Objects[0]
	fmt.Printf("DEBUG DATAGRID: Object[0] Name: %s, Columns: %d\n", obj.Name, len(obj.Columns))
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
			visible = override.Visible
			icon = override.Icon
			if l, ok := override.Labels[lang]; ok {
				label = l
			} else if l, ok := override.Labels["en"]; ok {
				label = l
			}
			overrideLov = override.LOV
		}

		// Process LOV (Static list or Dynamic SQL)
		lovItems := []LOVItem{}
		// 1. Check Global LOVs in Datagrid Config
		if globalLov, ok := cat.Datagrid.LOVs[col.Name]; ok {
			for _, item := range globalLov {
				lovItems = append(lovItems, processLovItem(item, lang))
			}
		}

		// 2. Add Inline/Override LOV (Dynamic SQL or Static List)
		lovSource := col.LOV
		if overrideLov != nil {
			lovSource = overrideLov
		}

		switch v := lovSource.(type) {
		case string: // Dynamic SQL
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
		case []interface{}: // Static List
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					li := LOVItem{Value: m["value"]}
					if labels, ok := m["labels"].(map[string]interface{}); ok {
						li.Labels = make(map[string]string)
						for k, v := range labels {
							if s, ok := v.(string); ok {
								li.Labels[k] = s
							}
						}
					} else if labels, ok := m["label"].(map[string]interface{}); ok { // Handle "label" as object
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

					processed := processLovItem(li, lang)

					// Only add if not duplicate
					isDup := false
					for _, existing := range lovItems {
						if existing.Value == processed.Value {
							isDup = true
							break
						}
					}
					if !isDup {
						lovItems = append(lovItems, processed)
					}
				}
			}
		}

		var cssClass string
		var displayPattern string
		if override, ok := cat.Datagrid.Columns[col.Name]; ok {
			cssClass = override.CSS
			displayPattern = override.Display
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

	return &Handler{
		DB:        db,
		TableName: obj.Name,
		Columns:   uiCols,
		Config:    cat.Datagrid,
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
	params := h.ParseParams(r)
	result, err := h.FetchData(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
		_, err := tx.Exec(fmt.Sprintf("SET LOCAL pg_trgm.similarity_threshold = %f", h.Config.Searchable.Threshold))
		if err != nil {
			fmt.Printf("DEBUG DATAGRID: SET threshold error: %v\n", err)
		}
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

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", tableName, where)
	var total int
	if err := tx.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch records - Select all columns for Forensic DOM metadata
	query := fmt.Sprintf("SELECT * FROM %s %s %s LIMIT %d OFFSET %d",
		tableName, where, order, p.Limit, p.Offset)

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []map[string]interface{}{}
	cols, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		fullRow := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			var renderedVal interface{}
			if b, ok := val.([]byte); ok {
				renderedVal = string(b)
			} else {
				renderedVal = val
			}
			fullRow[col] = renderedVal
			fullRow[col] = renderedVal
			row[col] = renderedVal
		}

		if len(records) == 0 {
			fmt.Println("DEBUG ROW 0:")
			for k, v := range row {
				fmt.Printf("  Key: %s, Type: %T, Value: %v\n", k, v, v)
			}
		}

		// Forensic DOM: Attach full row metadata
		if jsonBytes, err := json.Marshal(fullRow); err == nil {
			row["_json"] = string(jsonBytes)
		}

		// Calculate Row Styling and Classes
		var rowStyles []string
		var rowClasses []string
		for _, col := range h.Columns {
			val := row[col.Field]
			// Find matching LOV item
			for _, item := range col.LOV {
				// Simple loose comparison via string representation to handle float vs int cases (JSON uses float64)
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
		if len(rowStyles) > 0 {
			row["_row_style"] = strings.Join(rowStyles, "; ")
		}
		if len(rowClasses) > 0 {
			row["_row_class"] = strings.Join(rowClasses, " ")
		}

		records = append(records, row)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &TableResult{
		Records:    records,
		TotalCount: total,
		Offset:     p.Offset,
		Limit:      p.Limit,
		UIColumns:  h.Columns,
		Config:     h.Config,
	}, nil
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
		for _, val := range values {
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

		if len(paramParts) > 0 {
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

	defaultSort := "ORDER BY id DESC"
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
	}
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
