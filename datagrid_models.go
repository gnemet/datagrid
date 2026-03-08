package datagrid

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
	Version     string         `json:"version"`
	Title       string         `json:"title,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	Type        string         `json:"type,omitempty"`
	CSSClass    string         `json:"css_class,omitempty"`
	Source      string         `json:"source,omitempty"`      // v2.0: table name (e.g. "dwh.dim_issue")
	Description string         `json:"description,omitempty"` // v2.0: table description
	Datagrid    DatagridConfig `json:"datagrid,omitempty"`
	Objects     []ObjectDef    `json:"objects,omitempty"`
	Parameters  []QueryParam   `json:"parameters,omitempty"`
	SQL         string         `json:"sql,omitempty"`
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
	Pivot2           *Pivot2Config                `json:"pivot2,omitempty"`
	Links            map[string]string            `json:"links,omitempty"`
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
	Column   string         `json:"column" yaml:"column"`
	Func     string         `json:"func" yaml:"func"`                     // SUM, AVG, etc.
	Label    string         `json:"label,omitempty" yaml:"label"`         // Custom label for header
	ShowAt   []int          `json:"show_at,omitempty" yaml:"show_at"`     // If set, only show value at these depth levels
	Expr     string         `json:"expr,omitempty" yaml:"expr"`           // Computed: arithmetic on other measure labels (e.g., "Belső óra - Ügyfél óra")
	Format   string         `json:"format,omitempty" yaml:"format"`       // Printf format (e.g., "%.0f%%")
	Total    string         `json:"total,omitempty" yaml:"total"`         // Grand total mode: sum (default), avg, min, max, count
	CSSRules []PivotCSSRule `json:"css_rules,omitempty" yaml:"css_rules"` // Conditional CSS classes
}

// PivotCSSRule applies a CSS class when a measure value matches a condition.
type PivotCSSRule struct {
	When  string `json:"when" yaml:"when"`   // Condition: "> 10", "< 90", ">= 0"
	Class string `json:"class" yaml:"class"` // CSS class to apply
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
	Type            string    `json:"type"`    // DATE, TEXT, INTEGER, NUMERIC, TEXT[]
	Default         string    `json:"default"` // CURRENT_DATE, NULL, or literal
	Input           string    `json:"input"`   // date, number, text, select, lov, lov-tree, lov-grouped, constant
	Description     string    `json:"description"`
	Label           string    `json:"label,omitempty"`          // Display label (auto-generated from name if empty)
	LOVQuery        string    `json:"lov_query,omitempty"`      // SQL query for lov/lov-tree/lov-grouped options
	LOVName         string    `json:"lov_name,omitempty"`       // Named LOV function (e.g. "lov_department" → SELECT code, name FROM dwh.lov_department())
	SelectOptions   string    `json:"select_options,omitempty"` // Comma-separated options for select type (alternative to select:a,b,c)
	Constant        string    `json:"constant,omitempty"`       // Constant key (e.g. "current_user") — alternative to constant:key
	Options         []LOVItem `json:"options,omitempty"`        // Resolved at load time for select/lov
	ResolvedDefault string    `json:"-"`                        // Resolved default for HTML inputs
	IsArray         bool      `json:"isArray,omitempty"`        // True for array types (TEXT[], INTEGER[]) → renders multi-select
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

func (p QueryParam) resolvedSelectOptions() string {
	if p.SelectOptions != "" {
		return p.SelectOptions
	}
	if strings.HasPrefix(p.Input, "select:") {
		return strings.TrimPrefix(p.Input, "select:")
	}
	return ""
}

func (p QueryParam) resolvedLOVQuery() string {
	if p.LOVQuery != "" {
		return p.LOVQuery
	}
	if p.LOVName != "" {
		return fmt.Sprintf("SELECT code, name FROM dwh.lov_%s()", p.LOVName)
	}
	if strings.HasPrefix(p.Input, "lov:") || strings.HasPrefix(p.Input, "lov-tree:") || strings.HasPrefix(p.Input, "lov-grouped:") {
		parts := strings.SplitN(p.Input, ":", 2)
		if len(parts) == 2 {
			lovName := strings.TrimSpace(parts[1])
			return fmt.Sprintf("SELECT code, name FROM dwh.lov_%s()", lovName)
		}
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
	Type    string            `json:"type,omitempty"` // v2.0: column type (e.g. "TEXT", "BIGINT")
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
	ViewMode            string // "grid", "pivot", or "pivot2"
	PivotResult         *PivotResult
	Pivot2Result        *Pivot2Result
	LOVChooserThreshold int
	App                 struct {
		Name string
	}
	Catalogs        map[string]string
	LangsJSON       string
	CurrentLang     string
	QueryParams     []QueryParam
	IsQueryMode     bool
	ExecuteEndpoint string
	CurrentUser     string
}
