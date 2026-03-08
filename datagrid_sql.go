package datagrid

import (
	"encoding/json"
	"fmt"
	"strings"
	"os"
)

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
