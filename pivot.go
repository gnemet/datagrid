package datagrid

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PivotResult holds the aggregated data for the pivot view
type PivotResult struct {
	Rows          []string
	Cols          []string
	Measures      []string
	DisplayLabels map[string]string // Field -> Display Name
	DimensionCSS  map[string]string // Field -> CSS Class
	Data          map[string]map[string]map[string]float64
	RowTotals     map[string]map[string]float64
	ColTotals     map[string]map[string]float64
	GrandTotal    map[string]float64
}

// PivotData performs the aggregation logic for a pivot view
func (h *Handler) PivotData(p RequestParams) (*PivotResult, error) {
	if h.Config.Pivot == nil {
		return nil, fmt.Errorf("pivot configuration missing")
	}

	conf := h.Config.Pivot

	// 1. Prepare JSON config for datagrid_get_pivot_sql
	type DimDecl struct {
		Column string `json:"column"`
		IsLOV  bool   `json:"isLOV"`
		LovIdx int    `json:"lovIdx,omitempty"`
	}
	type MeasureDecl struct {
		Column string `json:"column"`
		Func   string `json:"func"`
		Alias  string `json:"alias"`
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

	dimsDecl := []DimDecl{}
	lovsDecl := []LOVDecl{}
	lovMapIdx := make(map[string]int)

	// Collect Dimensions
	for _, r := range conf.Rows {
		d := DimDecl{Column: r.Column}
		// Check if it's an LOV
		for _, col := range h.Columns {
			if col.Field == r.Column && len(col.LOV) > 0 {
				d.IsLOV = true
				if idx, ok := lovMapIdx[r.Column]; ok {
					d.LovIdx = idx
				} else {
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
						Column:     r.Column,
						Values:     entries,
						ValuesJSON: string(valJSON),
						IsBoolean:  isBool,
						IsNumber:   isNum,
					})
					d.LovIdx = len(lovsDecl)
					lovMapIdx[r.Column] = d.LovIdx
				}
				break
			}
		}
		dimsDecl = append(dimsDecl, d)
	}
	for _, c := range conf.Columns {
		d := DimDecl{Column: c.Column}
		for _, col := range h.Columns {
			if col.Field == c.Column && len(col.LOV) > 0 {
				d.IsLOV = true
				if idx, ok := lovMapIdx[c.Column]; ok {
					d.LovIdx = idx
				} else {
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
						Column:     c.Column,
						Values:     entries,
						ValuesJSON: string(valJSON),
						IsBoolean:  isBool,
						IsNumber:   isNum,
					})
					d.LovIdx = len(lovsDecl)
					lovMapIdx[c.Column] = d.LovIdx
				}
				break
			}
		}
		dimsDecl = append(dimsDecl, d)
	}

	measuresDecl := []MeasureDecl{}
	resMeasures := []string{}
	for i, v := range conf.Values {
		aggFunc := v.Func
		if strings.ToUpper(aggFunc) == "COUNT DISTINCT" {
			aggFunc = "COUNT(DISTINCT" // A bit hacky for the function, or handle in SQL
			// Actually, let's just use the SQL function to handle it or pass plain func
		}

		alias := fmt.Sprintf("val%d", i)
		measuresDecl = append(measuresDecl, MeasureDecl{
			Column: v.Column,
			Func:   v.Func,
			Alias:  alias,
		})

		label := v.Label
		if label == "" {
			label = fmt.Sprintf("%s(%s)", v.Func, v.Column)
		}
		resMeasures = append(resMeasures, label)
	}

	config["dimensions"] = dimsDecl
	config["measures"] = measuresDecl
	config["lovs"] = lovsDecl
	config["filters"] = p.Filters

	configJSON, _ := json.Marshal(config)

	// 2. Fetch Records

	// --- Hybrid Mode: Go Template Based Generation ---
	type DimWrap struct {
		Source string
		Column string
	}
	type MeasureWrap struct {
		Func   string
		Column string
		Alias  string
	}
	type FilterWrap struct {
		Column    string
		IsArray   bool
		IsBoolean bool
		IsNumber  bool
		Values    []string
		Value     string
	}

	dims := []DimWrap{}
	for _, d := range dimsDecl {
		src := "src." + quote_ident(d.Column)
		if d.IsLOV {
			src = "lov" + fmt.Sprintf("%d", d.LovIdx) + ".label"
		}
		dims = append(dims, DimWrap{Source: src, Column: d.Column})
	}

	measures := []MeasureWrap{}
	for _, m := range measuresDecl {
		measures = append(measures, MeasureWrap{
			Func:   m.Func,
			Column: m.Column,
			Alias:  m.Alias,
		})
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
		"TableName":  h.TableName,
		"Dimensions": dims,
		"Measures":   measures,
		"LOVs":       lovsDecl,
		"Filters":    filtersWrap,
	}

	query, err := h.renderSQL("pivot.sql", tplData)
	if err != nil {
		return nil, fmt.Errorf("failed to render pivot SQL template: %w", err)
	}
	if os.Getenv("DEBUG_SQL") == "true" {
		fmt.Printf("--- PIVOT SQL ---\n%s\n-----------------\n", query)
	}
	rows, err := h.DB.Query("SELECT datagrid.datagrid_execute_json($1, $2)", query, string(configJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to execute pivot template SQL: %w", err)
	}
	defer rows.Close()

	records := []map[string]interface{}{}
	for rows.Next() {
		var rowJSON string
		if err := rows.Scan(&rowJSON); err == nil {
			var row map[string]interface{}
			if err := json.Unmarshal([]byte(rowJSON), &row); err == nil {
				records = append(records, row)
			}
		}
	}

	res := &PivotResult{

		Measures:      resMeasures,
		DisplayLabels: make(map[string]string),
		DimensionCSS:  make(map[string]string),
		Data:          make(map[string]map[string]map[string]float64),
		RowTotals:     make(map[string]map[string]float64),
		ColTotals:     make(map[string]map[string]float64),
		GrandTotal:    make(map[string]float64),
	}

	// Build dimension metadata
	colMeta := make(map[string]string)
	for _, c := range h.Columns {
		colMeta[c.Field] = c.Display
	}

	for _, r := range conf.Rows {
		res.DisplayLabels[r.Column] = colMeta[r.Column]
		if res.DisplayLabels[r.Column] == "" {
			res.DisplayLabels[r.Column] = r.Column
		}
		res.DimensionCSS[r.Column] = r.CSS
	}
	for _, c := range conf.Columns {
		res.DisplayLabels[c.Column] = colMeta[c.Column]
		if res.DisplayLabels[c.Column] == "" {
			res.DisplayLabels[c.Column] = c.Column
		}
		res.DimensionCSS[c.Column] = c.CSS
	}

	rowKeySet := make(map[string]bool)
	colKeySet := make(map[string]bool)

	numRows := len(conf.Rows)
	numCols := len(conf.Columns)
	numVals := len(conf.Values)

	for _, row := range records {
		rKeys := []string{}
		for i := 0; i < numRows; i++ {
			field := conf.Rows[i].Column
			val := row[field]
			valStr := "(null)"
			if val != nil {
				valStr = strings.TrimSpace(fmt.Sprintf("%v", val))
			}
			rKeys = append(rKeys, valStr)
		}
		rKey := strings.Join(rKeys, " | ")

		if !rowKeySet[rKey] {
			rowKeySet[rKey] = true
			res.Rows = append(res.Rows, rKey)
		}

		cKeys := []string{}
		for i := 0; i < numCols; i++ {
			field := conf.Columns[i].Column
			val := row[field]
			valStr := "(null)"
			if val != nil {
				valStr = strings.TrimSpace(fmt.Sprintf("%v", val))
			}
			cKeys = append(cKeys, valStr)
		}
		cKey := strings.Join(cKeys, " | ")

		if !colKeySet[cKey] {
			colKeySet[cKey] = true
			res.Cols = append(res.Cols, cKey)
		}

		// Values extraction
		if res.Data[rKey] == nil {
			res.Data[rKey] = make(map[string]map[string]float64)
		}
		if res.Data[rKey][cKey] == nil {
			res.Data[rKey][cKey] = make(map[string]float64)
		}
		if res.RowTotals[rKey] == nil {
			res.RowTotals[rKey] = make(map[string]float64)
		}
		if res.ColTotals[cKey] == nil {
			res.ColTotals[cKey] = make(map[string]float64)
		}

		for i := 0; i < numVals; i++ {
			alias := fmt.Sprintf("val%d", i)
			valRaw := row[alias]
			mKey := resMeasures[i]
			var val float64

			switch v := valRaw.(type) {
			case float64:
				val = v
			case int64:
				val = float64(v)
			case float32:
				val = float64(v)
			case int:
				val = float64(v)
			case string:
				fmt.Sscanf(v, "%f", &val)
			default:
				fmt.Sscanf(fmt.Sprintf("%v", v), "%f", &val)
			}

			if conf.Multiplier != 0 {
				val *= conf.Multiplier
			}

			res.Data[rKey][cKey][mKey] = val
			res.RowTotals[rKey][mKey] += val
			res.ColTotals[cKey][mKey] += val
			res.GrandTotal[mKey] += val

			// Subtotal injection
			if conf.Subtotals {
				// Row subtotals
				if numRows > 1 {
					for length := 1; length < numRows; length++ {
						subRKey := strings.Join(rKeys[:length], " | ") + " (Total)"
						if res.Data[subRKey] == nil {
							res.Data[subRKey] = make(map[string]map[string]float64)
							if !rowKeySet[subRKey] {
								rowKeySet[subRKey] = true
								res.Rows = append(res.Rows, subRKey)
							}
						}
						if res.Data[subRKey][cKey] == nil {
							res.Data[subRKey][cKey] = make(map[string]float64)
						}
						res.Data[subRKey][cKey][mKey] += val
					}
				}
				// Col subtotals
				if numCols > 1 {
					for length := 1; length < numCols; length++ {
						subCKey := strings.Join(cKeys[:length], " | ") + " (Total)"
						if res.Data[rKey][subCKey] == nil {
							res.Data[rKey][subCKey] = make(map[string]float64)
							if !colKeySet[subCKey] {
								colKeySet[subCKey] = true
								res.Cols = append(res.Cols, subCKey)
							}
						}
						res.Data[rKey][subCKey][mKey] += val
					}
				}
			}
		}
	}

	return res, nil
}
