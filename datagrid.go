package datagrid

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// UIColumn defines how a column is rendered in the grid
type UIColumn struct {
	Field    string    `json:"field"`
	Label    string    `json:"label"`
	Class    string    `json:"class"`
	Type     string    `json:"type"`
	Sortable bool      `json:"sortable"`
	Visible  bool      `json:"visible"`
	Record   bool      `json:"record"` // Include in detail sidebar
	LOV      []LOVItem `json:"lov,omitempty"`
}

type LOVItem struct {
	Value  interface{}       `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
	Label  string            `json:"label,omitempty"`
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
	Searchable []string                     `json:"searchable_columns"`
}

type DatagridDefaults struct {
	PageSize      int                    `json:"page_size"`
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
	Labels  map[string]string `json:"labels"`
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
		if override, ok := cat.Datagrid.Columns[col.Name]; ok {
			visible = override.Visible
			if l, ok := override.Labels[lang]; ok {
				label = l
			} else if l, ok := override.Labels["en"]; ok {
				label = l
			}
		}

		// Process LOV (Static list or Dynamic SQL)
		lovItems := []LOVItem{}
		// 1. Check Global LOVs in Datagrid Config
		if globalLov, ok := cat.Datagrid.LOVs[col.Name]; ok {
			for _, item := range globalLov {
				lovItems = append(lovItems, processLovItem(item, lang))
			}
		}

		// 2. Add Inline LOV if present
		switch v := col.LOV.(type) {
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
					}
					if s, ok := m["label"].(string); ok {
						li.Label = s
					} else if m["label"] != nil {
						li.Label = fmt.Sprintf("%v", m["label"])
					}
					lovItems = append(lovItems, li)
				}
			}
		}

		uiCols = append(uiCols, UIColumn{
			Field:    col.Name,
			Label:    label,
			Sortable: true,
			Visible:  visible,
			Record:   true,
			Type:     strings.ToLower(col.Type),
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
		Value:  item.Value,
		Labels: item.Labels,
		Label:  item.Label,
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
	} else if h.Config.Defaults.PageSize > 0 {
		limit = h.Config.Defaults.PageSize
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
	where, args := h.buildWhere(p)
	order := h.buildOrder(p.Sort)

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", h.TableName, where)
	var total int
	if err := h.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch records
	selectCols := []string{}
	for _, c := range h.Columns {
		selectCols = append(selectCols, c.Field)
	}

	query := fmt.Sprintf("SELECT %s FROM %s %s %s LIMIT %d OFFSET %d",
		strings.Join(selectCols, ", "), h.TableName, where, order, p.Limit, p.Offset)

	rows, err := h.DB.Query(query, args...)
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
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		if jsonBytes, err := json.Marshal(row); err == nil {
			row["_json"] = string(jsonBytes)
		}

		records = append(records, row)
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
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", colName, strings.Join(paramParts, ", ")))
		}
	}

	// Search
	if p.Search != "" {
		searchCols := []string{}
		if len(h.Config.Searchable) > 0 {
			for _, sc := range h.Config.Searchable {
				searchCols = append(searchCols, fmt.Sprintf("%s::text", sc))
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
			clauses = append(clauses, fmt.Sprintf("$%d %% ANY(ARRAY[%s])", argIdx, strings.Join(searchCols, ", ")))
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
	defaultSort := "ORDER BY id DESC"
	if h.Config.Defaults.SortColumn != "" {
		dir := "ASC"
		if strings.ToUpper(h.Config.Defaults.SortDirection) == "DESC" {
			dir = "DESC"
		}
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
		parts := strings.Split(s, ":")
		if len(parts) == 2 {
			field := parts[0]
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
			dir := strings.ToUpper(parts[1])
			if dir != "ASC" && dir != "DESC" {
				dir = "ASC"
			}
			clauses = append(clauses, fmt.Sprintf("%s %s", field, dir))
		}
	}

	if len(clauses) == 0 {
		return defaultSort
	}
	return "ORDER BY " + strings.Join(clauses, ", ")
}
