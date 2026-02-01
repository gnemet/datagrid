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
	Value string      `json:"value"`
	Label interface{} `json:"label"` // Can be string or map[string]string
}

// Catalog structures for MIGR/JiraMntr compatibility
type Catalog struct {
	Version  string      `json:"version"`
	Objects  []ObjectDef `json:"objects"`
	Datagrid *struct {
		Columns map[string]struct {
			Visible bool              `json:"visible"`
			Labels  map[string]string `json:"labels"`
		} `json:"columns"`
	} `json:"datagrid,omitempty"`
}

type ObjectDef struct {
	Name    string      `json:"name"`
	Columns []ColumnDef `json:"columns"`
}

type ColumnDef struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
	LOV    interface{}       `json:"lov,omitempty"`
}

// RequestParams captures search, sort, and pagination from the request
type RequestParams struct {
	Search  string
	Sort    []string // List of "field:dir"
	Filters map[string]string
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
}

// Handler handles the grid data requests
type Handler struct {
	DB        *sql.DB
	TableName string
	Columns   []UIColumn
}

func NewHandler(db *sql.DB, tableName string, cols []UIColumn) *Handler {
	return &Handler{
		DB:        db,
		TableName: tableName,
		Columns:   cols,
	}
}

// NewHandlerFromCatalog initializes a Handler using a MIGR/JiraMntr JSON catalog
func NewHandlerFromCatalog(db *sql.DB, catalogPath string, lang string) (*Handler, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}

	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, err
	}

	if len(cat.Objects) == 0 {
		return nil, fmt.Errorf("no objects found in catalog")
	}

	obj := cat.Objects[0]
	uiCols := []UIColumn{}

	for _, col := range obj.Columns {
		label := col.Name
		if l, ok := col.Labels[lang]; ok {
			label = l
		} else if l, ok := col.Labels["en"]; ok {
			label = l
		}

		visible := true
		if cat.Datagrid != nil {
			if cfg, ok := cat.Datagrid.Columns[col.Name]; ok {
				visible = cfg.Visible
				if l, ok := cfg.Labels[lang]; ok {
					label = l
				}
			}
		}

		// Process LOV (Static list or Dynamic SQL)
		lovItems := []LOVItem{}
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
					li := LOVItem{Value: fmt.Sprintf("%v", m["value"])}
					labelRaw := m["label"]
					finalLabel := li.Value
					switch lv := labelRaw.(type) {
					case string:
						finalLabel = lv
					case map[string]interface{}:
						if l, ok := lv[lang].(string); ok {
							finalLabel = l
						} else if l, ok := lv["en"].(string); ok {
							finalLabel = l
						}
					}
					li.Label = finalLabel
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
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := h.parseParams(r)
	result, err := h.FetchData(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) parseParams(r *http.Request) RequestParams {
	q := r.URL.Query()
	limit := 10
	if l := q.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	filters := make(map[string]string)
	for key, values := range q {
		if key != "search" && key != "sort" && key != "limit" && key != "offset" {
			filters[key] = values[0]
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
	}, nil
}

func (h *Handler) buildWhere(p RequestParams) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}
	argIdx := 1

	// Exact Filters (LOVs)
	for field, val := range p.Filters {
		// Verify field exists and is valid for this table to prevent injection/errors
		validField := false
		for _, col := range h.Columns {
			if col.Field == field {
				validField = true
				break
			}
		}
		if validField && val != "" {
			clauses = append(clauses, fmt.Sprintf("%s = $%d", field, argIdx))
			args = append(args, val)
			argIdx++
		}
	}

	// Search
	if p.Search != "" {
		searchCols := []string{}
		for _, c := range h.Columns {
			if c.Type == "" || c.Type == "text" {
				searchCols = append(searchCols, fmt.Sprintf("%s::text", c.Field))
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
	if len(sorts) == 0 {
		return "ORDER BY id DESC"
	}

	clauses := []string{}
	for _, s := range sorts {
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
		return "ORDER BY id DESC"
	}
	return "ORDER BY " + strings.Join(clauses, ", ")
}
