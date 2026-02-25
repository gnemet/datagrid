package datagrid

import (
	"fmt"
	"sort"
	"strings"
)

// Pivot2Config defines the hierarchical pivot configuration.
// Unlike PivotConfig (cross-tab), Pivot2 operates row-only with collapsible tree levels.
type Pivot2Config struct {
	Levels []Pivot2Level      `json:"levels" yaml:"levels"` // hierarchy: e.g. project → issue → user
	Values []PivotValueConfig `json:"values" yaml:"values"` // aggregated measures (reuse existing type)
}

// Pivot2Level defines a single hierarchy level.
type Pivot2Level struct {
	Column string `json:"column" yaml:"column"`          // field name in result set
	Label  string `json:"label,omitempty" yaml:"label"` // display name for the level
}

// Pivot2Result holds the hierarchical pivot output.
type Pivot2Result struct {
	Levels              []string           // level display names
	Measures            []string           // measure display labels
	MeasureCSS          []string           // CSS class per measure (optional)
	Tree                []*Pivot2Row       // root-level grouped rows
	GrandTotal          map[string]float64 // grand totals per measure
	FormattedGrandTotal map[string]string  // custom formatted grand totals
	TotalCount          int                // total leaf record count
}

// Pivot2Row is a node in the hierarchical tree.
type Pivot2Row struct {
	Depth          int                // 0 = root group, 1 = sub-group, etc.
	Label          string             // display value for this group
	Key            string             // unique composite key for JS toggle (e.g. "0:IIER|1:IIER-123")
	Values         map[string]float64 // aggregated measure values
	FormattedVals  map[string]string  // custom-formatted values (for Expr measures with Format)
	HiddenMeasures map[string]bool    // measures hidden at this depth (via ShowAt)
	CSSClasses     map[string]string  // CSS class per measure (from css_rules)
	Children       []*Pivot2Row       // child rows (nil for leaf)
	IsLeaf         bool               // true for detail-level rows
	Record         map[string]interface{} // original record (only for leaves)
}

// Pivot2Data builds a hierarchical tree from flat records.
// Records should be pre-fetched detail-level rows (after param substitution).
func Pivot2Data(records []map[string]interface{}, cfg *Pivot2Config) *Pivot2Result {
	if cfg == nil || len(cfg.Levels) == 0 {
		return &Pivot2Result{}
	}

	// Build measure labels
	measures := make([]string, len(cfg.Values))
	for i, v := range cfg.Values {
		if v.Label != "" {
			measures[i] = v.Label
		} else {
			measures[i] = fmt.Sprintf("%s(%s)", v.Func, v.Column)
		}
	}

	// Build level labels
	levels := make([]string, len(cfg.Levels))
	for i, l := range cfg.Levels {
		if l.Label != "" {
			levels[i] = l.Label
		} else {
			levels[i] = l.Column
		}
	}

	// Recursively group
	tree := groupRecords(records, cfg.Levels, cfg.Values, measures, 0, "")

	// Compute grand total
	grandTotal := make(map[string]float64)
	formattedGrandTotal := make(map[string]string)

	for _, row := range tree {
		for i, vc := range cfg.Values {
			if vc.Expr == "" {
				grandTotal[measures[i]] += row.Values[measures[i]]
			}
		}
	}
	for i, vc := range cfg.Values {
		mKey := measures[i]
		if vc.Expr != "" {
			grandTotal[mKey] = evaluateExpr(vc.Expr, grandTotal)
		}
		if vc.Format != "" {
			formattedGrandTotal[mKey] = fmt.Sprintf(vc.Format, grandTotal[mKey])
		}
	}

	return &Pivot2Result{
		Levels:              levels,
		Measures:            measures,
		Tree:                tree,
		GrandTotal:          grandTotal,
		FormattedGrandTotal: formattedGrandTotal,
		TotalCount:          len(records),
	}
}

// groupRecords recursively groups records by the hierarchy levels.
func groupRecords(records []map[string]interface{}, levels []Pivot2Level, values []PivotValueConfig, measureLabels []string, depth int, parentKey string) []*Pivot2Row {
	if len(levels) == 0 || len(records) == 0 {
		return nil
	}

	currentLevel := levels[0]
	remainingLevels := levels[1:]

	// Group by current level's column
	groups := make(map[string][]map[string]interface{})
	groupOrder := []string{} // preserve insertion order

	for _, rec := range records {
		val := "(null)"
		if v, ok := rec[currentLevel.Column]; ok && v != nil {
			val = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if _, exists := groups[val]; !exists {
			groupOrder = append(groupOrder, val)
		}
		groups[val] = append(groups[val], rec)
	}

	// Sort group keys for deterministic output
	sort.Strings(groupOrder)

	result := make([]*Pivot2Row, 0, len(groupOrder))

	for _, key := range groupOrder {
		groupRecs := groups[key]
		rowKey := fmt.Sprintf("%d:%s", depth, key)
		if parentKey != "" {
			rowKey = parentKey + "|" + rowKey
		}

		row := &Pivot2Row{
			Depth:  depth,
			Label:  key,
			Key:    rowKey,
			Values: make(map[string]float64),
		}

		// Aggregate values for this group
		row.HiddenMeasures = make(map[string]bool)
		row.CSSClasses = make(map[string]string)
		row.FormattedVals = make(map[string]string)
		for i, vc := range values {
			mKey := measureLabels[i]
			if vc.Expr != "" {
				continue // computed measures are evaluated after all normal measures
			}
			if len(vc.ShowAt) > 0 && !intSliceContains(vc.ShowAt, depth) {
				row.Values[mKey] = 0
				row.HiddenMeasures[mKey] = true
			} else {
				row.Values[mKey] = aggregateValue(groupRecs, vc)
				if vc.Format != "" {
					row.FormattedVals[mKey] = fmt.Sprintf(vc.Format, row.Values[mKey])
				}
			}
		}

		// Evaluate computed (Expr) measures
		for i, vc := range values {
			if vc.Expr == "" {
				continue
			}
			mKey := measureLabels[i]
			if len(vc.ShowAt) > 0 && !intSliceContains(vc.ShowAt, depth) {
				row.Values[mKey] = 0
				row.HiddenMeasures[mKey] = true
			} else {
				row.Values[mKey] = evaluateExpr(vc.Expr, row.Values)
				if vc.Format != "" {
					row.FormattedVals[mKey] = fmt.Sprintf(vc.Format, row.Values[mKey])
				}
			}
		}

		// Apply CSS rules
		for i, vc := range values {
			if len(vc.CSSRules) == 0 {
				continue
			}
			mKey := measureLabels[i]
			if row.HiddenMeasures[mKey] {
				continue
			}
			row.CSSClasses[mKey] = matchCSSRules(row.Values[mKey], vc.CSSRules)
		}

		// Recurse into children if more levels remain
		if len(remainingLevels) > 0 {
			row.Children = groupRecords(groupRecs, remainingLevels, values, measureLabels, depth+1, rowKey)
			row.IsLeaf = false
		} else {
			// Leaf level: if only one record per group, attach the record
			row.IsLeaf = true
			if len(groupRecs) == 1 {
				row.Record = groupRecs[0]
			}
		}

		result = append(result, row)
	}

	return result
}

// aggregateValue computes an aggregate (SUM, COUNT, AVG, MIN, MAX) for a column over records.
func aggregateValue(records []map[string]interface{}, vc PivotValueConfig) float64 {
	fn := strings.ToUpper(vc.Func)
	var sum, min, max float64
	count := 0
	minSet := false

	for _, rec := range records {
		v := extractFloat(rec, vc.Column)

		switch fn {
		case "SUM", "AVG":
			sum += v
			count++
		case "COUNT":
			count++
		case "COUNT DISTINCT":
			// For simplicity, we count non-null entries
			if _, ok := rec[vc.Column]; ok && rec[vc.Column] != nil {
				count++
			}
		case "MIN":
			if !minSet || v < min {
				min = v
				minSet = true
			}
			count++
		case "MAX":
			if v > max {
				max = v
			}
			count++
		default:
			sum += v
			count++
		}
	}

	switch fn {
	case "SUM":
		return sum
	case "AVG":
		if count > 0 {
			return sum / float64(count)
		}
		return 0
	case "COUNT", "COUNT DISTINCT":
		return float64(count)
	case "MIN":
		return min
	case "MAX":
		return max
	default:
		return sum
	}
}

// extractFloat safely extracts a numeric value from a record field.
func extractFloat(rec map[string]interface{}, field string) float64 {
	v, ok := rec[field]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		var f float64
		fmt.Sscanf(fmt.Sprintf("%v", val), "%f", &f)
		return f
	}
}

// FlattenTree converts the hierarchical tree into a flat slice for template rendering.
// Each row is annotated with Depth, Key, and parent info for JS toggle.
func FlattenTree(tree []*Pivot2Row) []*Pivot2Row {
	var flat []*Pivot2Row
	for _, row := range tree {
		flat = append(flat, row)
		if row.Children != nil {
			flat = append(flat, FlattenTree(row.Children)...)
		}
	}
	return flat
}

// intSliceContains checks if a slice of ints contains a specific value.
func intSliceContains(slice []int, val int) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// evaluateExpr evaluates a simple arithmetic expression referencing measure labels.
// Example: "Belső óra - Ügyfél óra" → row.Values["Belső óra"] - row.Values["Ügyfél óra"]
// Supports: +, -, *, / operators. Measure labels are matched greedily.
func evaluateExpr(expr string, values map[string]float64) float64 {
	// Tokenize: replace known measure labels with their values, then evaluate
	// Strategy: sort labels by length (longest first) to avoid partial matches
	type labelVal struct {
		label string
		val   float64
	}
	var lvs []labelVal
	for k, v := range values {
		lvs = append(lvs, labelVal{k, v})
	}
	sort.Slice(lvs, func(i, j int) bool {
		return len(lvs[i].label) > len(lvs[j].label) // longest first
	})

	// Replace each label with a placeholder like __0__, __1__, etc.
	work := expr
	placeholderVals := make(map[string]float64)
	for i, lv := range lvs {
		ph := fmt.Sprintf("__%d__", i)
		work = strings.ReplaceAll(work, lv.label, ph)
		placeholderVals[ph] = lv.val
	}

	// Now parse the expression: split by operators, evaluate left to right
	// Tokenize into numbers and operators
	tokens := tokenizeExpr(work, placeholderVals)
	return evalTokens(tokens)
}

type exprToken struct {
	isOp bool
	op   byte
	val  float64
}

func tokenizeExpr(expr string, placeholders map[string]float64) []exprToken {
	var tokens []exprToken
	expr = strings.TrimSpace(expr)
	i := 0
	for i < len(expr) {
		ch := expr[i]
		if ch == ' ' {
			i++
			continue
		}
		if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			// Check if this is a negative sign (unary minus at start or after operator)
			if ch == '-' && (len(tokens) == 0 || tokens[len(tokens)-1].isOp) {
				// Unary minus: read the next number/placeholder
				i++
				val, newI := readValue(expr, i, placeholders)
				tokens = append(tokens, exprToken{val: -val})
				i = newI
				continue
			}
			tokens = append(tokens, exprToken{isOp: true, op: ch})
			i++
			continue
		}
		// Read a value (placeholder or number)
		val, newI := readValue(expr, i, placeholders)
		tokens = append(tokens, exprToken{val: val})
		i = newI
	}
	return tokens
}

func readValue(expr string, start int, placeholders map[string]float64) (float64, int) {
	// Check for placeholder __N__
	if start < len(expr) && expr[start] == '_' {
		end := strings.Index(expr[start+2:], "__")
		if end >= 0 {
			ph := expr[start : start+2+end+2]
			if v, ok := placeholders[ph]; ok {
				return v, start + len(ph)
			}
		}
	}
	// Read a number
	end := start
	for end < len(expr) && (expr[end] == '.' || (expr[end] >= '0' && expr[end] <= '9')) {
		end++
	}
	if end > start {
		var f float64
		fmt.Sscanf(expr[start:end], "%f", &f)
		return f, end
	}
	return 0, start + 1
}

func evalTokens(tokens []exprToken) float64 {
	if len(tokens) == 0 {
		return 0
	}
	// Simple left-to-right with * / precedence
	// First pass: handle * and /
	var pass1 []exprToken
	for i := 0; i < len(tokens); i++ {
		if tokens[i].isOp && (tokens[i].op == '*' || tokens[i].op == '/') {
			prev := pass1[len(pass1)-1]
			next := tokens[i+1]
			var result float64
			if tokens[i].op == '*' {
				result = prev.val * next.val
			} else if next.val != 0 {
				result = prev.val / next.val
			}
			pass1[len(pass1)-1] = exprToken{val: result}
			i++ // skip next
		} else {
			pass1 = append(pass1, tokens[i])
		}
	}
	// Second pass: handle + and -
	result := pass1[0].val
	for i := 1; i < len(pass1); i++ {
		if pass1[i].isOp {
			if i+1 < len(pass1) {
				switch pass1[i].op {
				case '+':
					result += pass1[i+1].val
				case '-':
					result -= pass1[i+1].val
				}
				i++
			}
		}
	}
	return result
}

// matchCSSRules checks a value against CSS rules and returns the first matching class.
func matchCSSRules(val float64, rules []PivotCSSRule) string {
	for _, rule := range rules {
		when := strings.TrimSpace(rule.When)
		if len(when) < 2 {
			continue
		}
		var op string
		var threshold float64

		// Parse operator and threshold
		if strings.HasPrefix(when, ">=") {
			op = ">="
			fmt.Sscanf(strings.TrimSpace(when[2:]), "%f", &threshold)
		} else if strings.HasPrefix(when, "<=") {
			op = "<="
			fmt.Sscanf(strings.TrimSpace(when[2:]), "%f", &threshold)
		} else if strings.HasPrefix(when, ">") {
			op = ">"
			fmt.Sscanf(strings.TrimSpace(when[1:]), "%f", &threshold)
		} else if strings.HasPrefix(when, "<") {
			op = "<"
			fmt.Sscanf(strings.TrimSpace(when[1:]), "%f", &threshold)
		} else if strings.HasPrefix(when, "=") {
			op = "="
			fmt.Sscanf(strings.TrimSpace(when[1:]), "%f", &threshold)
		}

		match := false
		switch op {
		case ">":
			match = val > threshold
		case "<":
			match = val < threshold
		case ">=":
			match = val >= threshold
		case "<=":
			match = val <= threshold
		case "=":
			match = val == threshold
		}
		if match {
			return rule.Class
		}
	}
	return ""
}
