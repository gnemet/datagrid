package datagrid

import (
	"fmt"
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

	// Quote identifiers
	quote := func(s string) string {
		if strings.Contains(s, "\"") || strings.Contains(s, "(") || strings.Contains(s, " ") {
			return s
		}
		return "\"" + s + "\""
	}

	quotedRows := []string{}
	for _, r := range conf.Rows {
		quotedRows = append(quotedRows, quote(r.Column))
	}
	quotedCols := []string{}
	for _, c := range conf.Columns {
		quotedCols = append(quotedCols, quote(c.Column))
	}

	rowDims := strings.Join(quotedRows, ", ")
	colDims := strings.Join(quotedCols, ", ")

	// Build measure selections
	measureSelections := []string{}
	resMeasures := []string{}
	for _, v := range conf.Values {
		aggFunc := v.Func
		if strings.ToUpper(aggFunc) == "COUNT DISTINCT" {
			measureSelections = append(measureSelections, fmt.Sprintf("COUNT(DISTINCT %s)", quote(v.Column)))
		} else {
			measureSelections = append(measureSelections, fmt.Sprintf("%s(%s)", aggFunc, quote(v.Column)))
		}

		label := v.Label
		if label == "" {
			label = fmt.Sprintf("%s(%s)", v.Func, v.Column)
		}
		resMeasures = append(resMeasures, label)
	}

	tableName := h.TableName
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		tableName = fmt.Sprintf("\"%s\".\"%s\"", parts[0], parts[1])
	} else {
		tableName = fmt.Sprintf("\"%s\"", tableName)
	}

	where, args := h.buildWhere(p)

	// Build the aggregation query
	query := fmt.Sprintf(
		"SELECT %s, %s, %s FROM %s %s GROUP BY %s, %s",
		rowDims,
		colDims,
		strings.Join(measureSelections, ", "),
		tableName,
		where,
		rowDims,
		colDims,
	)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

	cols, _ := rows.Columns()
	numRows := len(conf.Rows)
	numCols := len(conf.Columns)
	numVals := len(conf.Values)

	// Prepare LOV mappings
	lovMap := make(map[string]map[string]string)
	for _, col := range h.Columns {
		if len(col.LOV) > 0 {
			m := make(map[string]string)
			for _, item := range col.LOV {
				valStr := strings.TrimSpace(fmt.Sprintf("%v", item.Value))
				label := item.Label
				if label == "" {
					label = item.Display
				}
				if label != "" {
					m[valStr] = label
				}
			}
			lovMap[col.Field] = m
		}
	}

	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rKeys := []string{}
		for i := 0; i < numRows; i++ {
			val := values[i]
			var valStr string
			if val == nil {
				valStr = "(null)"
			} else if b, ok := val.([]byte); ok {
				valStr = string(b)
			} else {
				valStr = fmt.Sprintf("%v", val)
			}
			valStr = strings.TrimSpace(valStr)

			field := conf.Rows[i].Column
			if m, ok := lovMap[field]; ok {
				if label, ok := m[valStr]; ok {
					valStr = label
				}
			}
			rKeys = append(rKeys, valStr)
		}
		rKey := strings.Join(rKeys, " | ")

		if !rowKeySet[rKey] {
			rowKeySet[rKey] = true
			res.Rows = append(res.Rows, rKey)
		}

		cKeys := []string{}
		for i := numRows; i < numRows+numCols; i++ {
			val := values[i]
			var valStr string
			if val == nil {
				valStr = "(null)"
			} else if b, ok := val.([]byte); ok {
				valStr = string(b)
			} else {
				valStr = fmt.Sprintf("%v", val)
			}
			valStr = strings.TrimSpace(valStr)

			field := conf.Columns[i-numRows].Column
			if m, ok := lovMap[field]; ok {
				if label, ok := m[valStr]; ok {
					valStr = label
				}
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
			valRaw := values[numRows+numCols+i]
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
			case []byte:
				fmt.Sscanf(string(v), "%f", &val)
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
